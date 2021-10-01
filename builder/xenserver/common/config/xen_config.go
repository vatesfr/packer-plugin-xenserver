package config

import (
	"errors"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type XenConfig struct {
	Username    string `mapstructure:"remote_username"`
	Password    string `mapstructure:"remote_password"`
	HostIp      string `mapstructure:"remote_host"`
	HostPort    int    `mapstructure:"remote_port"`
	HostSSHPort int    `mapstructure:"remote_ssh_port"`
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
