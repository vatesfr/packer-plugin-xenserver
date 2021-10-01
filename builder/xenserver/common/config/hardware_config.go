package config

import (
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type HardwareConfig struct {
	VCPUsMax       uint `mapstructure:"vcpus_max"`
	VCPUsAtStartup uint `mapstructure:"vcpus_atstartup"`
	VMMemory       uint `mapstructure:"vm_memory"`
}

func (c *HardwareConfig) Prepare(ctx *interpolate.Context) (warnings []string, errs []error) {
	// Default values

	if c.VCPUsMax == 0 {
		c.VCPUsMax = 1
	}

	if c.VCPUsAtStartup == 0 {
		c.VCPUsAtStartup = 1
	}

	if c.VCPUsAtStartup > c.VCPUsMax {
		c.VCPUsAtStartup = c.VCPUsMax
	}

	if c.VMMemory == 0 {
		c.VMMemory = 1024
	}

	// Validation

	return
}
