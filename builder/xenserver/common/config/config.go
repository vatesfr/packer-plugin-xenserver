//go:generate packer-sdc mapstructure-to-hcl2 -type Config
package config

import (
	"time"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	CommonConfig        `mapstructure:",squash"`

	VCPUsMax        uint              `mapstructure:"vcpus_max"`
	VCPUsAtStartup  uint              `mapstructure:"vcpus_atstartup"`
	VMMemory        uint              `mapstructure:"vm_memory"`
	DiskSize        uint              `mapstructure:"disk_size"`
	AdditionalDisks []uint            `mapstructure:"additional_disks"`
	CloneTemplate   string            `mapstructure:"clone_template"`
	VMOtherConfig   map[string]string `mapstructure:"vm_other_config"`

	ISOChecksum     string   `mapstructure:"iso_checksum"`
	ISOChecksumType string   `mapstructure:"iso_checksum_type"`
	ISOUrls         []string `mapstructure:"iso_urls"`
	ISOUrl          string   `mapstructure:"iso_url"`
	ISOName         string   `mapstructure:"iso_name"`

	PlatformArgs map[string]string `mapstructure:"platform_args"`

	RawInstallTimeout string        `mapstructure:"install_timeout"`
	InstallTimeout    time.Duration ``
	SourcePath        string        `mapstructure:"source_path"`

	Firmware string `mapstructure:"firmware"`

	ctx interpolate.Context
}

func (c Config) GetInterpContext() *interpolate.Context {
	return &c.ctx
}
