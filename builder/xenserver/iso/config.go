//go:generate packer-sdc mapstructure-to-hcl2 -type Config
//go:generate packer-sdc struct-markdown

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

	/*
		A URL to the ISO containing the installation image.
		This URL can be either an HTTP URL or a file URL (or path to a file).
		If this is an HTTP URL, Packer will download it and cache it between runs.
	*/
	ISOUrl string `mapstructure:"iso_url" required:"true"`

	/*
		The checksum for the OS ISO file. Because ISO files are so large, this is required and Packer will verify it prior
		to booting a virtual machine with the ISO attached.
		The type of the checksum is specified with `iso_checksum_type`, documented below.
	*/
	ISOChecksum string `mapstructure:"iso_checksum" required:"true"`

	/*
		The type of the checksum specified in `iso_checksum`. Valid values are "none", "md5", "sha1", "sha256", or
		"sha512" currently. While "none" will skip checksumming, this is not recommended since ISO files are
		generally large and corruption does happen from time to time.
	*/
	ISOChecksumType string `mapstructure:"iso_checksum_type" required:"true"`

	/*
		Multiple URLs for the ISO to download. Packer will try these in order. If anything goes wrong attempting to
		download or while downloading a single URL, it will move on to the next.
		All URLs must point to the same file (same checksum). By default this is empty and `iso_url` is used.
		Only one of `iso_url` or `iso_urls` can be specified.
	*/
	ISOUrls []string `mapstructure:"iso_urls"`

	ISOName string `mapstructure:"iso_name"`

	/*
		The size, in megabytes, of the hard disk to create for the VM. By default, this is 40000 (about 40 GB).
	*/
	DiskSize        uint   `mapstructure:"disk_size"`
	AdditionalDisks []uint `mapstructure:"additional_disks"`

	/*
		The template to clone. Defaults to "Other install media", this is "other", but you can get
		_dramatic_ performance improvements by setting this to the proper value. To view all available values for this
		run `xe template-list`. Setting the correct value hints to XenServer how to optimize the virtual hardware
		to work best with that operating system.
	*/
	CloneTemplate string            `mapstructure:"clone_template"`
	VMOtherConfig map[string]string `mapstructure:"vm_other_config"`

	/*
		The amount of time to wait after booting the VM for the installer to shut itself down.
		If it doesn't shut down in this time, it is an error. By default, the timeout is "200m", or over three hours.
	*/
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
		c.InstallTimeout = 200 * time.Minute
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
