# XCP-ng packer.io builder

This builder plugin extends packer.io to support building images for XCP-ng.

This fork focuses on simplifying the builder’s logic and reducing the amount of code to make maintenance easier.

## Status

It works, but don't consider this as a stable/production version

## Using the builder

Put this line at the top of your pkr.hcl file

```
packer {
  required_plugins {
    xcp = {
      version = ">= 0.10.0"
      source  = "github.com/disruptivemindseu/xcp"
    }
  }
}
```

## Developing the builder

### Dependencies
* Packer >= v1.7.1 (https://packer.io)
* XCP-ng > 8.2
* Golang 1.20

## Compile the plugin

Once you have installed Packer, you must compile this plugin and install the
resulting binary.

Documentation for Plugins directory: [Official Docs](https://developer.hashicorp.com/packer/docs/configure#packer-s-plugin-directory)

### Linux/MacOS

```shell
make dev
```

It should output "Successfully installed plugin"

Don't use `packer init` when using `make dev`, packer don't support well dev version

# Documentation

For complete documentation on configuration commands, see [the
xcp-iso docs](docs/builders/xcp.md)

## Support

Special thanks to ddelnano and the Vates devops team for maintaining the upstream builder :)  
