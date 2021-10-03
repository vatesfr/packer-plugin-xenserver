//go:generate packer-sdc mapstructure-to-hcl2 -type Config
//go:generate packer-sdc struct-markdown

package iso

import (
	"errors"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	xscommon "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"time"
)

type Config struct {
	xscommon.CommonConfig `mapstructure:",squash"`

	commonsteps.ISOConfig `mapstructure:",squash"`

	ISOName string `mapstructure:"iso_name"`

	/*
		The type of the checksum specified in `iso_checksum`. Valid values are "none", "md5", "sha1", "sha256", or
		"sha512" currently. While "none" will skip checksumming, this is not recommended since ISO files are
		generally large and corruption does happen from time to time.
	*/
	// TODO Deprecated
	DeprecatedISOChecksumType string `mapstructure:"iso_checksum_type" undocumented:"true"`

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

	params, warnings, merrs := c.CommonConfig.Prepare(c, raws...)
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

	if c.DeprecatedISOChecksumType != "" {
		warnings = append(warnings, "iso_checksum_type is deprecated. Please use combined iso_checksum format.")
	}

	/*
		Validate the ISO configuration.
		Either a pre-uploaded ISO should be referenced in iso_name,
		OR a URL (possibly to a local file) to an ISO file that will be downloaded and then uploaded to Xen.
	*/

	if c.ISOName == "" {
		isoWarnings, isoErrors := c.ISOConfig.Prepare(c.GetInterpContext())
		errs = packersdk.MultiErrorAppend(errs, isoErrors...)
		warnings = append(warnings, isoWarnings...)
	}

	if (c.ISOName == "" && len(c.ISOUrls) == 0) || (c.ISOName != "" && len(c.ISOUrls) > 0) {
		errs = packersdk.MultiErrorAppend(errs,
			errors.New("either iso_name or iso_url, but not both, must be specified"))
	}

	if errs != nil && len(errs.Errors) > 0 {
		return params, warnings, errs
	}

	return params, warnings, nil
}
