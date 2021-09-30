package common

import (
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// Based upon https://github.com/hashicorp/packer-plugin-qemu/blob/a2121bb95d84288a1df5d7fbce94985a7cdfb793/builder/qemu/comm_config.go

type CommConfig struct {
	Comm communicator.Config `mapstructure:",squash"`

	// Defaults to false. When enabled, Packer
	// does not setup forwarded port mapping for communicator (SSH or WinRM) requests and uses ssh_port or winrm_port
	// on the host to communicate to the virtual machine.
	SkipNatMapping bool `mapstructure:"skip_nat_mapping" required:"false"`

	// These are deprecated, but we keep them around for backwards compatibility
	// TODO: remove later
	sshKeyPath        string `mapstructure:"ssh_key_path"`
	sshSkipNatMapping bool   `mapstructure:"ssh_skip_nat_mapping"`
	hostPortMin       int    `mapstructure:"host_port_min" required:"false"`
	hostPortMax       int    `mapstructure:"host_port_max" required:"false"`
}

func (c *CommConfig) Prepare(ctx *interpolate.Context) (warnings []string, errs []error) {

	const removedHostPortFmt = "%s is deprecated because free ports are selected automatically. " +
		"Please remove %s from your template. " +
		"In future versions of Packer, inclusion of %s will error your builds."

	// Backwards compatibility
	if c.hostPortMin != 0 {
		warnings = append(warnings, fmt.Sprintf(removedHostPortFmt, "host_port_min", "host_port_min", "host_port_min"))
	}

	// Backwards compatibility
	if c.hostPortMax != 0 {
		warnings = append(warnings, fmt.Sprintf(removedHostPortFmt, "host_port_max", "host_port_max", "host_port_max"))
	}

	// Backwards compatibility
	if c.sshKeyPath != "" {
		warnings = append(warnings, "ssh_key_path is deprecated and is being replaced by ssh_private_key_file. "+
			"Please, update your template to use ssh_private_key_file. In future versions of Packer, inclusion of ssh_key_path will error your builds.")
		c.Comm.SSHPrivateKeyFile = c.sshKeyPath
	}

	// Backwards compatibility
	if c.sshSkipNatMapping {
		warnings = append(warnings, "ssh_skip_nat_mapping is deprecated and is being replaced by skip_nat_mapping. "+
			"Please, update your template to use skip_nat_mapping. In future versions of Packer, inclusion of ssh_skip_nat_mapping will error your builds.")

		c.SkipNatMapping = c.sshSkipNatMapping
	}

	if c.Comm.SSHHost == "" && c.SkipNatMapping {
		c.Comm.SSHHost = "127.0.0.1"
		c.Comm.WinRMHost = "127.0.0.1"
	}

	errs = c.Comm.Prepare(ctx)

	return
}
