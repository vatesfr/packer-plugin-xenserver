//go:generate packer-sdc struct-markdown

package config

import (
	"errors"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/shutdowncommand"
	hconfig "github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	xenapi "github.com/terra-farm/go-xen-api-client"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"
)

type CommonConfig struct {
	common.PackerConfig    `mapstructure:",squash"`
	bootcommand.VNCConfig  `mapstructure:",squash"`
	commonsteps.HTTPConfig `mapstructure:",squash"`

	XenConfig `mapstructure:",squash"`

	commonsteps.FloppyConfig `mapstructure:",squash"`

	/*
		This is the name of the new virtual machine, without the file extension.
		By default this is "packer-BUILDNAME-TIMESTAMP", where "BUILDNAME" is the name of the build.
	*/
	VMName string `mapstructure:"vm_name"`

	/*
		The description of the new virtual machine. By default, this is the empty string.
	*/
	VMDescription string `mapstructure:"vm_description"`
	SrName        string `mapstructure:"sr_name"`
	SrISOName     string `mapstructure:"sr_iso_name"`

	/*
		A list of networks identified by their name label which will be used for the VM during creation.
		The first network will correspond to the VM's first network interface (VIF),
		the second will corespond to the second VIF and so on.
	*/
	NetworkNames       []string `mapstructure:"network_names"`
	ExportNetworkNames []string `mapstructure:"export_network_names"`

	/*
		The platform args. Defaults to
		```javascript
		{
		    "viridian": "false",
		    "nx": "true",
		    "pae": "true",
		    "apic": "true",
		    "timeoffset": "0",
		    "acpi": "1",
		    "cores-per-socket": "1"
		}
		```
	*/
	PlatformArgs map[string]string `mapstructure:"platform_args"`

	shutdowncommand.ShutdownConfig `mapstructure:",squash"`

	/*
		The name of the XenServer Tools ISO. Defaults to "xs-tools.iso".
	*/
	ToolsIsoName string `mapstructure:"tools_iso_name"`

	CommConfig `mapstructure:",squash" `

	/*
		This is the path to the directory where the resulting virtual machine will be created.
		This may be relative or absolute. If relative, the path is relative to the working directory when `packer`
		is executed. This directory must not exist or be empty prior to running the builder.
		By default this is "output-BUILDNAME" where "BUILDNAME" is the name of the build.
	*/
	OutputDir string `mapstructure:"output_directory"`

	/*
		Either "xva", "vdi_raw" or "none", this specifies the output format of the exported virtual machine.
		This defaults to "xva". Set to "vdi_raw" to export just the raw disk image. Set to "none" to export nothing;
		this is only useful with "keep_vm" set to "always" or "on_success".
	*/
	Format string `mapstructure:"format"`

	/*
		Determine when to keep the VM and when to clean it up. This can be "always", "never" or "on_success".
		By default this is "never", and Packer always deletes the VM regardless of whether the process succeeded
		and an artifact was produced. "always" asks Packer to leave the VM at the end of the process regardless of success.
		"on_success" requests that the VM only be cleaned up if an artifact was produced.
		The latter is useful for debugging templates that fail.
	*/
	KeepVM   string `mapstructure:"keep_vm"`
	IPGetter string `mapstructure:"ip_getter"`

	/*
		Set the firmware to use. Can be "bios" or "uefi".
	*/
	Firmware       string `mapstructure:"firmware"`
	HardwareConfig `mapstructure:",squash"`

	ctx interpolate.Context
}

func (c *CommonConfig) GetInterpContext() *interpolate.Context {
	return &c.ctx
}

func (c *CommonConfig) Prepare(upper interface{}, raws ...interface{}) ([]string, []string, error) {

	err := hconfig.Decode(upper, &hconfig.DecodeOpts{
		Interpolate: true,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
			},
		},
	}, raws...)

	if err != nil {
		return nil, nil, err
	}

	var errs *packersdk.MultiError
	var warnings []string

	// Set default values

	if c.Firmware == "" {
		c.Firmware = "bios"
	}

	if c.ToolsIsoName == "" {
		c.ToolsIsoName = "xs-tools.iso"
	}

	if c.OutputDir == "" {
		c.OutputDir = fmt.Sprintf("output-%s", c.PackerConfig.PackerBuildName)
	}

	if c.VMName == "" {
		c.VMName = fmt.Sprintf("packer-%s-{{timestamp}}", c.PackerConfig.PackerBuildName)
	}

	if c.Format == "" {
		c.Format = "xva"
	}

	if c.KeepVM == "" {
		c.KeepVM = "never"
	}

	if c.IPGetter == "" {
		c.IPGetter = "auto"
	}

	if len(c.PlatformArgs) == 0 {
		pargs := make(map[string]string)
		pargs["viridian"] = "false"
		pargs["nx"] = "true"
		pargs["pae"] = "true"
		pargs["apic"] = "true"
		pargs["timeoffset"] = "0"
		pargs["acpi"] = "1"
		c.PlatformArgs = pargs
	}

	// Validation

	// Lower bound is not checked in commonsteps.HTTPConfig
	if c.HTTPPortMin < 0 {
		errs = packersdk.MultiErrorAppend(errs, errors.New("the HTTP min port must greater than zero"))
	}

	switch c.Format {
	case "xva", "xva_compressed", "vdi_raw", "vdi_vhd", "none":
	default:
		errs = packersdk.MultiErrorAppend(errs, errors.New("format must be one of 'xva', 'vdi_raw', 'vdi_vhd', 'none'"))
	}

	switch c.KeepVM {
	case "always", "never", "on_success":
	default:
		errs = packersdk.MultiErrorAppend(errs, errors.New("keep_vm must be one of 'always', 'never', 'on_success'"))
	}

	switch c.IPGetter {
	case "auto", "tools", "http":
	default:
		errs = packersdk.MultiErrorAppend(errs, errors.New("ip_getter must be one of 'auto', 'tools', 'http'"))
	}

	innerWarnings, es := c.CommConfig.Prepare(&c.ctx)
	errs = packersdk.MultiErrorAppend(errs, es...)
	warnings = append(warnings, innerWarnings...)

	innerWarnings, es = c.XenConfig.Prepare(&c.ctx)
	errs = packersdk.MultiErrorAppend(errs, es...)
	warnings = append(warnings, innerWarnings...)

	innerWarnings, es = c.HardwareConfig.Prepare(&c.ctx)
	errs = packersdk.MultiErrorAppend(errs, es...)
	warnings = append(warnings, innerWarnings...)

	errs = packersdk.MultiErrorAppend(errs, c.VNCConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.HTTPConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.FloppyConfig.Prepare(&c.ctx)...)

	return nil, warnings, errs
}

// steps should check config.ShouldKeepVM first before cleaning up the VM
func (c CommonConfig) ShouldKeepVM(state multistep.StateBag) bool {
	switch c.KeepVM {
	case "always":
		return true
	case "never":
		return false
	case "on_success":
		// only keep instance if build was successful
		_, cancelled := state.GetOk(multistep.StateCancelled)
		_, halted := state.GetOk(multistep.StateHalted)
		return !(cancelled || halted)
	default:
		panic(fmt.Sprintf("Unknown keep_vm value '%s'", c.KeepVM))
	}
}

func (config CommonConfig) GetSR(c *xen.Connection) (xenapi.SRRef, error) {
	var srRef xenapi.SRRef
	if config.SrName == "" {
		hostRef, err := c.GetClient().Session.GetThisHost(c.GetSessionRef(), c.GetSessionRef())

		if err != nil {
			return srRef, err
		}

		pools, err := c.GetClient().Pool.GetAllRecords(c.GetSessionRef())

		if err != nil {
			return srRef, err
		}

		for _, pool := range pools {
			if pool.Master == hostRef {
				return pool.DefaultSR, nil
			}
		}

		return srRef, errors.New(fmt.Sprintf("failed to find default SR on host '%s'", hostRef))

	} else {
		// Use the provided name label to find the SR to use
		srs, err := c.GetClient().SR.GetByNameLabel(c.GetSessionRef(), config.SrName)

		if err != nil {
			return srRef, err
		}

		switch {
		case len(srs) == 0:
			return srRef, fmt.Errorf("Couldn't find a SR with the specified name-label '%s'", config.SrName)
		case len(srs) > 1:
			return srRef, fmt.Errorf("Found more than one SR with the name '%s'. The name must be unique", config.SrName)
		}

		return srs[0], nil
	}
}

func (config CommonConfig) GetISOSR(c *xen.Connection) (xenapi.SRRef, error) {
	var srRef xenapi.SRRef
	if config.SrISOName == "" {
		return srRef, errors.New("sr_iso_name must be specified in the packer configuration")

	} else {
		// Use the provided name label to find the SR to use
		srs, err := c.GetClient().SR.GetByNameLabel(c.GetSessionRef(), config.SrName)

		if err != nil {
			return srRef, err
		}

		switch {
		case len(srs) == 0:
			return srRef, fmt.Errorf("Couldn't find a SR with the specified name-label '%s'", config.SrName)
		case len(srs) > 1:
			return srRef, fmt.Errorf("Found more than one SR with the name '%s'. The name must be unique", config.SrName)
		}

		return srs[0], nil
	}
}
