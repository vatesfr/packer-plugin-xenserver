package common

import (
	"errors"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// Based upon https://github.com/hashicorp/packer-plugin-qemu/blob/a2121bb95d84288a1df5d7fbce94985a7cdfb793/builder/qemu/comm_config.go

type CommConfig struct {
	Comm communicator.Config `mapstructure:",squash"`

	// The minimum port to use for the Communicator port on the host machine which is forwarded
	// to the SSH or WinRM port on the guest machine. By default this is 2222.
	HostPortMin int `mapstructure:"host_port_min" required:"false"`
	// The maximum port to use for the Communicator port on the host machine which is forwarded
	// to the SSH or WinRM port on the guest machine. Because Packer often runs in parallel,
	// Packer will choose a randomly available port in this range to use as the
	// host port. By default this is 4444.
	HostPortMax int `mapstructure:"host_port_max" required:"false"`
	// Defaults to false. When enabled, Packer
	// does not setup forwarded port mapping for communicator (SSH or WinRM) requests and uses ssh_port or winrm_port
	// on the host to communicate to the virtual machine.
	SkipNatMapping bool `mapstructure:"skip_nat_mapping" required:"false"`

	// These are deprecated, but we keep them around for backwards compatibility
	// TODO: remove later
	sshHostPortMin int `mapstructure:"ssh_host_port_min" required:"false"`
	// TODO: remove later
	sshHostPortMax int `mapstructure:"ssh_host_port_max"`
	// TODO: remove later
	sshKeyPath string `mapstructure:"ssh_key_path"`
	// TODO: remove later
	sshSkipNatMapping bool `mapstructure:"ssh_skip_nat_mapping"`
}

func (c *CommConfig) Prepare(ctx *interpolate.Context) (warnings []string, errs []error) {

	// Backwards compatibility
	if c.sshHostPortMin != 0 {
		warnings = append(warnings, "ssh_host_port_min is deprecated and is being replaced by host_port_min. "+
			"Please, update your template to use host_port_min. In future versions of Packer, inclusion of ssh_host_port_min will error your builds.")
		c.HostPortMin = c.sshHostPortMin
	}

	// Backwards compatibility
	if c.sshHostPortMax != 0 {
		warnings = append(warnings, "ssh_host_port_max is deprecated and is being replaced by host_port_max. "+
			"Please, update your template to use host_port_max. In future versions of Packer, inclusion of ssh_host_port_max will error your builds.")
		c.HostPortMax = c.sshHostPortMax
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

	if c.HostPortMin == 0 {
		c.HostPortMin = 2222
	}

	if c.HostPortMax == 0 {
		c.HostPortMax = 4444
	}

	errs = c.Comm.Prepare(ctx)
	if c.HostPortMin > c.HostPortMax {
		errs = append(errs,
			errors.New("host_port_min must be less than host_port_max"))
	}

	if c.HostPortMin < 0 {
		errs = append(errs, errors.New("host_port_min must be positive"))
	}

	return
}
