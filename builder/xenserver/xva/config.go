//go:generate packer-sdc mapstructure-to-hcl2 -type Config
//go:generate packer-sdc struct-markdown

package xva

import (
	"fmt"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	xscommon "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
)

type Config struct {
	xscommon.CommonConfig `mapstructure:",squash"`

	SourcePath string `mapstructure:"source_path"`
}

func (c *Config) Prepare(raws ...interface{}) ([]string, []string, error) {
	var errs *packersdk.MultiError
	params, warnings, merrs := c.CommonConfig.Prepare(c, raws)
	if merrs != nil {
		errs = packersdk.MultiErrorAppend(errs, merrs)
	}

	// Validation

	if c.SourcePath == "" {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("A source_path must be specified"))
	}

	if errs != nil && len(errs.Errors) > 0 {
		return params, warnings, errs
	}
	return params, warnings, nil
}
