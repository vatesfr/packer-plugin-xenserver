//go:generate packer-sdc struct-markdown

package config

import (
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type HardwareConfig struct {
	/*
		The maximum number of VCPUs for the VM. By default this is 1.
	*/
	VCPUsMax uint `mapstructure:"vcpus_max"`

	/*
		The number of startup VCPUs for the VM. By default this is 1.
	*/
	VCPUsAtStartup uint `mapstructure:"vcpus_atstartup"`

	/*
		The size, in megabytes, of the amount of memory to allocate for the VM. By default, this is 1024 (1 GB).
	*/
	VMMemory uint `mapstructure:"vm_memory"`
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
