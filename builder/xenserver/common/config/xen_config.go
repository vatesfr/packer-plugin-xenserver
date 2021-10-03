package config

import (
	"errors"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type XenConfig struct {
	/*
		The XenServer username used to access the remote machine.
	*/
	Username string `mapstructure:"remote_username" required:"true"`

	/*
		The XenServer password for access to the remote machine.
	*/
	Password string `mapstructure:"remote_password" required:"true"`

	/*
		The host of the Xenserver / XCP-ng pool primary.
		Typically these will be specified through environment variables as seen in the
		[examples](../../examples/centos8.json).
	*/
	HostIp string `mapstructure:"remote_host" required:"true"`

	/*
		The port of the Xenserver API. Defaults to 443.
	*/
	HostPort int `mapstructure:"remote_port"`

	/*
		The ssh port of the Xenserver pool primary. Defaults to 22.
	*/
	HostSSHPort int `mapstructure:"remote_ssh_port"`
}

func (c *XenConfig) Prepare(ctx *interpolate.Context) (warnings []string, errs []error) {
	// Default values

	if c.HostPort == 0 {
		c.HostPort = 443
	}

	if c.HostSSHPort == 0 {
		c.HostSSHPort = 22
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

	return
}
