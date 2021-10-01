//go:generate packer-sdc mapstructure-to-hcl2 -type Config
package iso

import (
	"errors"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	xscommon "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"strings"
	"time"
)

type Config struct {
	xscommon.CommonConfig `mapstructure:",squash"`

	DiskSize        uint              `mapstructure:"disk_size"`
	AdditionalDisks []uint            `mapstructure:"additional_disks"`
	CloneTemplate   string            `mapstructure:"clone_template"`
	VMOtherConfig   map[string]string `mapstructure:"vm_other_config"`

	ISOChecksum     string   `mapstructure:"iso_checksum"`
	ISOChecksumType string   `mapstructure:"iso_checksum_type"`
	ISOUrls         []string `mapstructure:"iso_urls"`
	ISOUrl          string   `mapstructure:"iso_url"`
	ISOName         string   `mapstructure:"iso_name"`

	InstallTimeout time.Duration `mapstructure:"install_timeout"`
}

func (c *Config) Prepare(raws ...interface{}) ([]string, []string, error) {
	var errs *packersdk.MultiError

	params, warnings, merrs := c.CommonConfig.Prepare(c, raws)
	if merrs != nil {
		errs = packersdk.MultiErrorAppend(errs, merrs)
	}

	// Set default values

	if c.InstallTimeout == 0 {
		c.InstallTimeout = 3 * time.Hour
	}

	if c.DiskSize == 0 {
		c.DiskSize = 40000
	}

	if c.CloneTemplate == "" {
		c.CloneTemplate = "Other install media"
	}

	// Validation

	if c.ISOName == "" {

		// If ISO name is not specified, assume a URL and checksum has been provided.

		if c.ISOChecksumType == "" {
			errs = packersdk.MultiErrorAppend(
				errs, errors.New("The iso_checksum_type must be specified."))
		} else {
			c.ISOChecksumType = strings.ToLower(c.ISOChecksumType)
			if c.ISOChecksumType != "none" {
				if c.ISOChecksum == "" {
					errs = packersdk.MultiErrorAppend(
						errs, errors.New("Due to the file size being large, an iso_checksum is required."))
				} else {
					c.ISOChecksum = strings.ToLower(c.ISOChecksum)
				}
			}
		}

		if len(c.ISOUrls) == 0 {
			if c.ISOUrl == "" {
				errs = packersdk.MultiErrorAppend(
					errs, errors.New("One of iso_url or iso_urls must be specified."))
			} else {
				c.ISOUrls = []string{c.ISOUrl}
			}
		} else if c.ISOUrl != "" {
			errs = packersdk.MultiErrorAppend(
				errs, errors.New("Only one of iso_url or iso_urls may be specified."))
		}
	} else {

		// An ISO name has been provided. It should be attached from an available SR.

	}

	if errs != nil && len(errs.Errors) > 0 {
		return params, warnings, errs
	}

	return params, warnings, nil
}
