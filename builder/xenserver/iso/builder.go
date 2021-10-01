package iso

import (
	"context"
	"errors"
	"fmt"
	config2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	commonsteps "github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	hconfig "github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	xsclient "github.com/terra-farm/go-xen-api-client"
	xscommon "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common"
)

type Builder struct {
	config config2.Config
	runner multistep.Runner
}

func (self *Builder) ConfigSpec() hcldec.ObjectSpec { return self.config.FlatMapstructure().HCL2Spec() }

func (self *Builder) Prepare(raws ...interface{}) (params []string, warns []string, retErr error) {

	var errs *packer.MultiError

	err := hconfig.Decode(&self.config, &hconfig.DecodeOpts{
		Interpolate: true,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
			},
		},
	}, raws...)

	if err != nil {
		errs = packer.MultiErrorAppend(errs, err)
	}

	commonWarnings, commonErrors := self.config.CommonConfig.Prepare(self.config.GetInterpContext(), &self.config.PackerConfig)
	errs = packer.MultiErrorAppend(errs, commonErrors...)
	warns = append(warns, commonWarnings...)

	// Set default values

	if self.config.RawInstallTimeout == "" {
		self.config.RawInstallTimeout = "200m"
	}

	if self.config.DiskSize == 0 {
		self.config.DiskSize = 40000
	}

	if self.config.VCPUsMax == 0 {
		self.config.VCPUsMax = 1
	}

	if self.config.VCPUsAtStartup == 0 {
		self.config.VCPUsAtStartup = 1
	}

	if self.config.VCPUsAtStartup > self.config.VCPUsMax {
		self.config.VCPUsAtStartup = self.config.VCPUsMax
	}

	if self.config.VMMemory == 0 {
		self.config.VMMemory = 1024
	}

	if self.config.CloneTemplate == "" {
		self.config.CloneTemplate = "Other install media"
	}

	if self.config.Firmware == "" {
		self.config.Firmware = "bios"
	}

	if len(self.config.PlatformArgs) == 0 {
		pargs := make(map[string]string)
		pargs["viridian"] = "false"
		pargs["nx"] = "true"
		pargs["pae"] = "true"
		pargs["apic"] = "true"
		pargs["timeoffset"] = "0"
		pargs["acpi"] = "1"
		self.config.PlatformArgs = pargs
	}

	// Template substitution

	templates := map[string]*string{
		"clone_template":    &self.config.CloneTemplate,
		"iso_checksum":      &self.config.ISOChecksum,
		"iso_checksum_type": &self.config.ISOChecksumType,
		"iso_url":           &self.config.ISOUrl,
		"iso_name":          &self.config.ISOName,
		"install_timeout":   &self.config.RawInstallTimeout,
	}
	for i := range self.config.ISOUrls {
		templates[fmt.Sprintf("iso_urls[%d]", i)] = &self.config.ISOUrls[i]
	}

	// Validation

	self.config.InstallTimeout, err = time.ParseDuration(self.config.RawInstallTimeout)
	if err != nil {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Failed to parse install_timeout: %s", err))
	}

	if self.config.ISOName == "" {

		// If ISO name is not specified, assume a URL and checksum has been provided.

		if self.config.ISOChecksumType == "" {
			errs = packer.MultiErrorAppend(
				errs, errors.New("The iso_checksum_type must be specified."))
		} else {
			self.config.ISOChecksumType = strings.ToLower(self.config.ISOChecksumType)
			if self.config.ISOChecksumType != "none" {
				if self.config.ISOChecksum == "" {
					errs = packer.MultiErrorAppend(
						errs, errors.New("Due to the file size being large, an iso_checksum is required."))
				} else {
					self.config.ISOChecksum = strings.ToLower(self.config.ISOChecksum)
				}
			}
		}

		if len(self.config.ISOUrls) == 0 {
			if self.config.ISOUrl == "" {
				errs = packer.MultiErrorAppend(
					errs, errors.New("One of iso_url or iso_urls must be specified."))
			} else {
				self.config.ISOUrls = []string{self.config.ISOUrl}
			}
		} else if self.config.ISOUrl != "" {
			errs = packer.MultiErrorAppend(
				errs, errors.New("Only one of iso_url or iso_urls may be specified."))
		}
	} else {

		// An ISO name has been provided. It should be attached from an available SR.

	}

	if len(errs.Errors) > 0 {
		retErr = errors.New(errs.Error())
	}

	return nil, warns, retErr

}

func (self *Builder) Run(ctx context.Context, ui packer.Ui, hook packer.Hook) (packer.Artifact, error) {
	c, err := xen.NewXenAPIClient(self.config.HostIp, self.config.HostPort, self.config.Username, self.config.Password)

	if err != nil {
		return nil, err
	}
	ui.Say("XAPI client session established")

	c.GetClient().Host.GetAll(c.GetSessionRef())

	//Share state between the other steps using a statebag
	state := new(multistep.BasicStateBag)
	state.Put("client", c)
	state.Put("config", self.config)
	state.Put("commonconfig", self.config.CommonConfig)
	state.Put("hook", hook)
	state.Put("ui", ui)

	httpReqChan := make(chan string, 1)

	//Build the steps
	download_steps := []multistep.Step{
		&commonsteps.StepDownload{
			Checksum:    self.config.ISOChecksum,
			Description: "ISO",
			ResultKey:   "iso_path",
			Url:         self.config.ISOUrls,
		},
	}

	steps := []multistep.Step{
		&xscommon.StepPrepareOutputDir{
			Force: self.config.PackerForce,
			Path:  self.config.OutputDir,
		},
		&commonsteps.StepCreateFloppy{
			Files: self.config.FloppyFiles,
		},
		&xscommon.StepUploadVdi{
			VdiNameFunc: func() string {
				return "Packer-floppy-disk"
			},
			ImagePathFunc: func() string {
				if floppyPath, ok := state.GetOk("floppy_path"); ok {
					return floppyPath.(string)
				}
				return ""
			},
			VdiUuidKey: "floppy_vdi_uuid",
		},
		&xscommon.StepUploadVdi{
			VdiNameFunc: func() string {
				if len(self.config.ISOUrls) > 0 {
					return path.Base(self.config.ISOUrls[0])
				}
				return ""
			},
			ImagePathFunc: func() string {
				if isoPath, ok := state.GetOk("iso_path"); ok {
					return isoPath.(string)
				}
				return ""
			},
			VdiUuidKey: "iso_vdi_uuid",
		},
		&xscommon.StepFindVdi{
			VdiName:    self.config.ToolsIsoName,
			VdiUuidKey: "tools_vdi_uuid",
		},
		&xscommon.StepFindVdi{
			VdiName:    self.config.ISOName,
			VdiUuidKey: "isoname_vdi_uuid",
		},
		new(stepCreateInstance),
		&xscommon.StepAttachVdi{
			VdiUuidKey: "floppy_vdi_uuid",
			VdiType:    xsclient.VbdTypeFloppy,
		},
		&xscommon.StepAttachVdi{
			VdiUuidKey: "iso_vdi_uuid",
			VdiType:    xsclient.VbdTypeCD,
		},
		&xscommon.StepAttachVdi{
			VdiUuidKey: "isoname_vdi_uuid",
			VdiType:    xsclient.VbdTypeCD,
		},
		&xscommon.StepAttachVdi{
			VdiUuidKey: "tools_vdi_uuid",
			VdiType:    xsclient.VbdTypeCD,
		},
		new(xscommon.StepStartVmPaused),
		new(xscommon.StepSetVmHostSshAddress),
		new(xscommon.StepHTTPIPDiscover),
		&xscommon.StepCreateProxy{},
		commonsteps.HTTPServerFromHTTPConfig(&self.config.HTTPConfig),
		new(xscommon.StepBootWait),
		&xscommon.StepTypeBootCommand{
			Ctx: *self.config.GetInterpContext(),
		},
		/*
			VNC is only available after boot command because xenserver doesn't seem to support two vnc connections at the same time
		*/
		&xscommon.StepGetVNCPort{},
		&xscommon.StepWaitForIP{
			Chan:    httpReqChan,
			Timeout: self.config.InstallTimeout, // @todo change this
		},
		&xscommon.StepCreateForwarding{Targets: []xscommon.ForwardTarget{
			{
				Host: xscommon.InstanceCommIP,
				Port: xscommon.InstanceCommPort,
				Key:  "local_comm_address",
			},
		}},
		&communicator.StepConnect{
			Config: &self.config.Comm,
			Host: func(state multistep.StateBag) (string, error) {
				return xscommon.GetForwardedHost(state, "local_comm_address")
			},
			SSHConfig: self.config.Comm.SSHConfigFunc(),
			SSHPort: func(state multistep.StateBag) (int, error) {
				return xscommon.GetForwardedPort(state, "local_comm_address")
			},
			WinRMPort: func(state multistep.StateBag) (int, error) {
				return xscommon.GetForwardedPort(state, "local_comm_address")
			},
		},
		new(commonsteps.StepProvision),
		new(xscommon.StepShutdown),
		new(xscommon.StepSetVmToTemplate),
		&xscommon.StepDetachVdi{
			VdiUuidKey: "iso_vdi_uuid",
		},
		&xscommon.StepDetachVdi{
			VdiUuidKey: "isoname_vdi_uuid",
		},
		&xscommon.StepDetachVdi{
			VdiUuidKey: "tools_vdi_uuid",
		},
		&xscommon.StepDetachVdi{
			VdiUuidKey: "floppy_vdi_uuid",
		},
		new(xscommon.StepExport),
	}

	if self.config.ISOName == "" {
		steps = append(download_steps, steps...)
	}

	self.runner = &multistep.BasicRunner{Steps: steps}
	self.runner.Run(ctx, state)

	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("Build was cancelled.")
	}
	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("Build was halted.")
	}

	artifact, _ := xscommon.NewArtifact(self.config.OutputDir)

	return artifact, nil
}
