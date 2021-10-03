//go:generate packer-sdc struct-markdown

package config

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

	/*
		Path to a private key to use for authenticating with SSH. By default this is not set (key-based auth won't be used).
		The associated public key is expected to already be configured on the VM being prepared by some other process
		(kickstart, etc.).
	*/
	DeprecatedSSHKeyPath        string `mapstructure:"ssh_key_path" undocumented:"true"`
	DeprecatedSSHSkipNatMapping bool   `mapstructure:"ssh_skip_nat_mapping" undocumented:"true"`
	DeprecatedHostPortMin       int    `mapstructure:"host_port_min" required:"false" undocumented:"true"`
	DeprecatedHostPortMax       int    `mapstructure:"host_port_max" required:"false" undocumented:"true"`
}

func (c *CommConfig) Prepare(ctx *interpolate.Context) (warnings []string, errs []error) {

	const removedHostPortFmt = "%s is deprecated because free ports are selected automatically. " +
		"Please remove %s from your template. " +
		"In future versions of Packer, inclusion of %s will error your builds."

	// Backwards compatibility
	if c.DeprecatedHostPortMin != 0 {
		warnings = append(warnings, fmt.Sprintf(removedHostPortFmt, "host_port_min", "host_port_min", "host_port_min"))
	}

	// Backwards compatibility
	if c.DeprecatedHostPortMax != 0 {
		warnings = append(warnings, fmt.Sprintf(removedHostPortFmt, "host_port_max", "host_port_max", "host_port_max"))
	}

	// Backwards compatibility
	if c.DeprecatedSSHKeyPath != "" {
		warnings = append(warnings, "ssh_key_path is deprecated and is being replaced by ssh_private_key_file. "+
			"Please, update your template to use ssh_private_key_file. In future versions of Packer, inclusion of ssh_key_path will error your builds.")
		c.Comm.SSHPrivateKeyFile = c.DeprecatedSSHKeyPath
	}

	// Backwards compatibility
	if c.DeprecatedSSHSkipNatMapping {
		warnings = append(warnings, "ssh_skip_nat_mapping is deprecated and is being replaced by skip_nat_mapping. "+
			"Please, update your template to use skip_nat_mapping. In future versions of Packer, inclusion of ssh_skip_nat_mapping will error your builds.")

		c.SkipNatMapping = c.DeprecatedSSHSkipNatMapping
	}

	if c.Comm.SSHHost == "" && c.SkipNatMapping {
		c.Comm.SSHHost = "127.0.0.1"
		c.Comm.WinRMHost = "127.0.0.1"
	}

	errs = c.Comm.Prepare(ctx)

	return
}
