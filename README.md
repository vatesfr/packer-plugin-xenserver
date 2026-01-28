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

## Developing the builder

### Dependencies
* Packer >= v1.7.1 (https://packer.io)
* XCP-ng / Citrix Hypervisor > 7.6
* Golang 1.20

## Compile the plugin

Once you have installed Packer, you must compile this plugin and install the
resulting binary.

Documentation for Plugins directory: [Official Docs](https://developer.hashicorp.com/packer/docs/configure#packer-s-plugin-directory)

To compile the plugin, you can use this commands:

```
make build
make dev
```
`make dev` should output

```
Successfully installed plugin github.com/vatesfr/xenserver from <path>/packer-plugin-xenserver/packer-plugin-xenserver to ~/.packer.d/plugins/github.com/vatesfr/xenserver/packer-plugin-xenserver_v0.9.0-dev_x5.0_linux_amd64
```

Then you can use your build file like usual.

# Documentation

For complete documentation on configuration commands, see [the
xenserver-iso docs](docs/builders/iso/xenserver-iso.html.markdown)

## Support

You can discuss any issues you have or feature requests directly on this repository, or on the [forum](https://xcp-ng.org/forum/).