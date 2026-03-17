# XCP-ng packer.io builder

This builder plugin extends packer.io to support building images for XCP-ng.

This is a fork of the original builder since the original project was abandoned and no longer compiled with recent versions of Go or worked with Xenserver 7.6 and later.

It improves the original project in the following ways:
1. Developed alongside the [Xenorchestra terraform provider](https://github.com/vatesfr/terraform-provider-xenorchestra) to ensure the hashicorp ecosystem is interoperable.
2. Reimplements how the boot commands are sent over VNC to be compatible with later versions of Xenserver (Citrix hypervisor) and XCP-ng
3. New feature are added on regular basics, following request made via issues or PR

## Status

At the time of this writing (January 2026) the packer builder has been verified to work with XCP-ng 8.3 and we can create VMs with the templates built by this Packer plugin either through the xenorchestra terraform provider or using Xen Orchestra web UI.

The following list contains things that are incomplete but will be worked on soon:

- XVA builder is untested
- Lots of dead code to remove from upstream

## Using the builder

The packer builder can be installed via `packer init` as long as the packer template includes the following in it's `pkr.hcl` file
```
packer {
  required_plugins {
    xenserver= {
      version = ">= v0.9.0"
      source = "github.com/vatesfr/xenserver"
    }
  }
}
```

The following command will install the packer plugin using the Ubuntu example provided in this repository.

```
packer init examples/ubuntu/ubuntu-2004.pkr.hcl
```

### Replacing Existing Templates

The builder supports the usage of the Packer [`-force`](https://developer.hashicorp.com/packer/docs/commands/build#force) flag to clean up templates with the **same name** as your current build. If you want to continously update a single "golden image", using the `-force` flag will let you achieve that.

Alternatively, if you omit the `-force` flag, a new template will always be created. (Provided you elected to create one with the [`skip_set_template`](docs/builders/iso/xenserver-iso.html.markdown#L158-L159) parameter.)

Examples:

```hcl
source "xenserver-iso" "ubuntu-2404" {
  # If '-force' is used, all templates with the same name as below will be *REMOVED*.
  vm_name        = "Ubuntu 24.04 LTS"
```

## Developing the builder

### Dependencies
* Packer >= v1.7.1 (https://packer.io)
* XCP-ng > 8.2.1
* Golang 1.26

## Compile the plugin

Once you have installed Packer, you must compile this plugin and install the
resulting binary.

Documentation for Plugins directory: [Official Docs](https://developer.hashicorp.com/packer/docs/configure#packer-s-plugin-directory)

To compile the plugin, you can use this commands:

```
make install-packer-sdc
make build
make dev
```
`make dev` should output

```
Successfully installed plugin github.com/vatesfr/xenserver from <path>/packer-plugin-xenserver/packer-plugin-xenserver to ~/.packer.d/plugins/github.com/vatesfr/xenserver/packer-plugin-xenserver_v0.9.0-dev_x5.0_linux_amd64
```

Then you can use your build file like usual.

To enable Packer logs: 

```sh
export PACKER_LOG=1
```

# Documentation

For complete documentation on configuration commands, see [the
xenserver-iso docs](docs/builders/iso/xenserver-iso.html.markdown)

## Support

You can discuss any issues you have or feature requests directly on this repository, or on the [forum](https://xcp-ng.org/forum/).