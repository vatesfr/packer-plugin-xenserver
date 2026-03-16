//go:generate packer-sdc mapstructure-to-hcl2 -type Config,DiskConfig
package common

import (
	"time"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	CommonConfig        `mapstructure:",squash"`

	VCPUsMax       uint              `mapstructure:"vcpus_max"`
	VCPUsAtStartup uint              `mapstructure:"vcpus_atstartup"`
	VMMemory       uint              `mapstructure:"vm_memory"`
	CloneTemplate  string            `mapstructure:"clone_template"`
	VMOtherConfig  map[string]string `mapstructure:"vm_other_config"`

	ISOChecksum string   `mapstructure:"iso_checksum"`
	ISOUrls     []string `mapstructure:"iso_urls"`
	ISOUrl      string   `mapstructure:"iso_url"`
	ISOName     string   `mapstructure:"iso_name"`

	PlatformArgs map[string]string `mapstructure:"platform_args"`

	RawInstallTimeout string        `mapstructure:"install_timeout"`
	InstallTimeout    time.Duration `mapstructure-to-hcl2:",skip"`
	SourcePath        string        `mapstructure:"source_path"`

	Firmware        string `mapstructure:"firmware"`
	SkipSetTemplate bool   `mapstructure:"skip_set_template"`

	ctx interpolate.Context
}

func (c Config) GetInterpContext() *interpolate.Context {
	return &c.ctx
}
