package common

import (
	"errors"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/shutdowncommand"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	xenapi "github.com/terra-farm/go-xen-api-client"
)

type CommonConfig struct {
	bootcommand.VNCConfig  `mapstructure:",squash"`
	commonsteps.HTTPConfig `mapstructure:",squash"`

	Username    string `mapstructure:"remote_username"`
	Password    string `mapstructure:"remote_password"`
	HostIp      string `mapstructure:"remote_host"`
	HostPort    int    `mapstructure:"remote_port"`
	HostSSHPort int    `mapstructure:"remote_ssh_port"`

	VMName             string   `mapstructure:"vm_name"`
	VMDescription      string   `mapstructure:"vm_description"`
	SrName             string   `mapstructure:"sr_name"`
	SrISOName          string   `mapstructure:"sr_iso_name"`
	FloppyFiles        []string `mapstructure:"floppy_files"`
	NetworkNames       []string `mapstructure:"network_names"`
	ExportNetworkNames []string `mapstructure:"export_network_names"`

	shutdowncommand.ShutdownConfig `mapstructure:",squash"`

	RawBootWait string `mapstructure:"boot_wait"`
	BootWait    time.Duration

	ToolsIsoName string `mapstructure:"tools_iso_name"`

	CommConfig `mapstructure:",squash"`

	OutputDir string `mapstructure:"output_directory"`
	Format    string `mapstructure:"format"`
	KeepVM    string `mapstructure:"keep_vm"`
	IPGetter  string `mapstructure:"ip_getter"`
}

func (c *CommonConfig) Prepare(ctx *interpolate.Context, pc *common.PackerConfig) []error {
	var err error
	var errs []error

	// Set default values

	if c.HostPort == 0 {
		c.HostPort = 443
	}

	if c.HostSSHPort == 0 {
		c.HostSSHPort = 22
	}

	if c.RawBootWait == "" {
		c.RawBootWait = "5s"
	}

	if c.ToolsIsoName == "" {
		c.ToolsIsoName = "xs-tools.iso"
	}

	if c.HTTPPortMin == 0 {
		c.HTTPPortMin = 8000
	}

	if c.HTTPPortMax == 0 {
		c.HTTPPortMax = 9000
	}

	if c.FloppyFiles == nil {
		c.FloppyFiles = make([]string, 0)
	}

	if c.OutputDir == "" {
		c.OutputDir = fmt.Sprintf("output-%s", pc.PackerBuildName)
	}

	if c.VMName == "" {
		c.VMName = fmt.Sprintf("packer-%s-{{timestamp}}", pc.PackerBuildName)
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

	// Validation

	if c.Username == "" {
		errs = append(errs, errors.New("remote_username must be specified."))
	}

	if c.Password == "" {
		errs = append(errs, errors.New("remote_password must be specified."))
	}

	if c.HostIp == "" {
		errs = append(errs, errors.New("remote_host must be specified."))
	}

	if c.HTTPPortMin > c.HTTPPortMax {
		errs = append(errs, errors.New("the HTTP min port must be less than the max"))
	}

	c.BootWait, err = time.ParseDuration(c.RawBootWait)
	if err != nil {
		errs = append(errs, fmt.Errorf("Failed to parse boot_wait: %s", err))
	}

	switch c.Format {
	case "xva", "xva_compressed", "vdi_raw", "vdi_vhd", "none":
	default:
		errs = append(errs, errors.New("format must be one of 'xva', 'vdi_raw', 'vdi_vhd', 'none'"))
	}

	switch c.KeepVM {
	case "always", "never", "on_success":
	default:
		errs = append(errs, errors.New("keep_vm must be one of 'always', 'never', 'on_success'"))
	}

	switch c.IPGetter {
	case "auto", "tools", "http":
	default:
		errs = append(errs, errors.New("ip_getter must be one of 'auto', 'tools', 'http'"))
	}

	return errs
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

func (config CommonConfig) GetSR(c *Connection) (xenapi.SRRef, error) {
	var srRef xenapi.SRRef
	if config.SrName == "" {
		hostRef, err := c.GetClient().Session.GetThisHost(c.session, c.session)

		if err != nil {
			return srRef, err
		}

		pools, err := c.GetClient().Pool.GetAllRecords(c.session)

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
		srs, err := c.GetClient().SR.GetByNameLabel(c.session, config.SrName)

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

func (config CommonConfig) GetISOSR(c *Connection) (xenapi.SRRef, error) {
	var srRef xenapi.SRRef
	if config.SrISOName == "" {
		return srRef, errors.New("sr_iso_name must be specified in the packer configuration")

	} else {
		// Use the provided name label to find the SR to use
		srs, err := c.GetClient().SR.GetByNameLabel(c.session, config.SrName)

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
