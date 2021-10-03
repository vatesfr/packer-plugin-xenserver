package iso

import (
	"context"
	"errors"
	artifact2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/artifact"
	steps2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/steps"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/workaround"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"
	"log"
	"net"
	"net/http"
	"path"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	commonsteps "github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	xsclient "github.com/terra-farm/go-xen-api-client"
)

type Builder struct {
	config Config
	runner multistep.Runner
}

func (self *Builder) ConfigSpec() hcldec.ObjectSpec { return self.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) (params []string, warns []string, errors error) {
	return b.config.Prepare(raws...)
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

	httpServerStep := workaround.HTTPServerFromHTTPConfig(&self.config.HTTPConfig)
	httpServerStep.AddCallback(func(req *http.Request) {
		log.Printf("HTTP: %s %s %s", req.RemoteAddr, req.Method, req.URL)
		ip, _, err := net.SplitHostPort(req.RemoteAddr)

		if err == nil && ip != "" {
			select {
			case httpReqChan <- ip:
				log.Printf("Remembering remote address '%s'", ip)
			default:
				// if ch is already full, don't block waiting to send the address, just drop it
			}
		}
	})

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
		&steps2.StepPrepareOutputDir{
			Force: self.config.PackerForce,
			Path:  self.config.OutputDir,
		},
		&commonsteps.StepCreateFloppy{
			Files:       self.config.FloppyFiles,
			Directories: self.config.FloppyDirectories,
			Label:       self.config.FloppyLabel,
		},
		&steps2.StepUploadVdi{
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
		&steps2.StepUploadVdi{
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
		&steps2.StepFindVdi{
			VdiName:    self.config.ToolsIsoName,
			VdiUuidKey: "tools_vdi_uuid",
		},
		&steps2.StepFindVdi{
			VdiName:    self.config.ISOName,
			VdiUuidKey: "isoname_vdi_uuid",
		},
		new(stepCreateInstance),
		&steps2.StepAttachVdi{
			VdiUuidKey: "floppy_vdi_uuid",
			VdiType:    xsclient.VbdTypeFloppy,
		},
		&steps2.StepAttachVdi{
			VdiUuidKey: "iso_vdi_uuid",
			VdiType:    xsclient.VbdTypeCD,
		},
		&steps2.StepAttachVdi{
			VdiUuidKey: "isoname_vdi_uuid",
			VdiType:    xsclient.VbdTypeCD,
		},
		&steps2.StepAttachVdi{
			VdiUuidKey: "tools_vdi_uuid",
			VdiType:    xsclient.VbdTypeCD,
		},
		new(steps2.StepStartVmPaused),
		new(steps2.StepSetVmHostSshAddress),
		new(steps2.StepHTTPIPDiscover),
		&steps2.StepCreateProxy{},
		httpServerStep,
		new(steps2.StepBootWait),
		&steps2.StepTypeBootCommand{
			Ctx: *self.config.GetInterpContext(),
		},
		/*
			VNC is only available after boot command because xenserver doesn't seem to support two vnc connections at the same time
		*/
		&steps2.StepGetVNCPort{},
		&steps2.StepWaitForIP{
			Chan:    httpReqChan,
			Timeout: self.config.InstallTimeout, // @todo change this
		},
		&steps2.StepCreateForwarding{Targets: []steps2.ForwardTarget{
			{
				Host: steps2.InstanceCommIP,
				Port: steps2.InstanceCommPort,
				Key:  "local_comm_address",
			},
		}},
		&communicator.StepConnect{
			Config: &self.config.Comm,
			Host: func(state multistep.StateBag) (string, error) {
				return steps2.GetForwardedHost(state, "local_comm_address")
			},
			SSHConfig: self.config.Comm.SSHConfigFunc(),
			SSHPort: func(state multistep.StateBag) (int, error) {
				return steps2.GetForwardedPort(state, "local_comm_address")
			},
			WinRMPort: func(state multistep.StateBag) (int, error) {
				return steps2.GetForwardedPort(state, "local_comm_address")
			},
		},
		new(commonsteps.StepProvision),
		new(steps2.StepShutdown),
		new(steps2.StepSetVmToTemplate),
		&steps2.StepDetachVdi{
			VdiUuidKey: "iso_vdi_uuid",
		},
		&steps2.StepDetachVdi{
			VdiUuidKey: "isoname_vdi_uuid",
		},
		&steps2.StepDetachVdi{
			VdiUuidKey: "tools_vdi_uuid",
		},
		&steps2.StepDetachVdi{
			VdiUuidKey: "floppy_vdi_uuid",
		},
		new(steps2.StepExport),
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

	artifact, _ := artifact2.NewArtifact(self.config.OutputDir)

	return artifact, nil
}
