packer {
  required_plugins {
   xenserver= {
      version = ">= v0.9.0"
      source = "github.com/vatesfr/xenserver"
    }
  }
}

data "null" "version" {
  input = "3.23.2"
}

locals {
  timestamp = regex_replace(timestamp(), "[- TZ:]", "")
  version = data.null.version.output
  sha256_version = regex_replace(local.version, "\\.[0-9]+$", "")

  template_name = {
    latest = "NHT Alpine"
  }
}


# local "sha256" {
#   expression = regex("^([a-f0-9]+)", data.http.sha256.body)
# }



# Fetch Alpine ISO SHA256
# data "http" "sha256" {
#   url = "https://dl-cdn.alpinelinux.org/alpine/v${local.version}/releases/x86_64/alpine-virt-${local.version}.iso.sha256"
# }


variable "remote_host" {
  type        = string
  description = "The ip or fqdn of your XenServer. This will be pulled from the env var 'PKR_VAR_XAPI_HOST'"
  sensitive   = true
  default     = null
}

variable "remote_password" {
  type        = string
  description = "The password used to interact with your XenServer. This will be pulled from the env var 'PKR_VAR_XAPI_PASSWORD'"
  sensitive   = true
  default     = null
}

variable "remote_username" {
  type        = string
  description = "The username used to interact with your XenServer. This will be pulled from the env var 'PKR_VAR_XAPI_USERNAME'"
  sensitive   = true
  default     = null
}

variable "sr_iso_name" {
  type        = string
  default     = ""
  description = "The ISO-SR to packer will use"
}

variable "sr_name" {
  type        = string
  default     = ""
  description = "The name of the SR to packer will use"
}

source "xenserver-iso" "alpine" {
  iso_checksum      = "c328a553ba9861e4ccb3560d69e426256955fa954bc6f084772e6e6cd5b0a4d0"
  iso_url           = "https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/x86_64/alpine-virt-3.23.2-x86_64.iso"

  sr_iso_name    = var.sr_iso_name
  sr_name        = var.sr_name
  tools_iso_name = "guest-tools.iso"

  remote_host     = var.remote_host
  remote_password = var.remote_password
  remote_username = var.remote_username

  # Use the Alpine template
  clone_template = local.template_name["latest"]
  vm_name        = "packer-alpine-${data.null.version.output}-${local.timestamp}"
  vm_description = "Build started: ${local.timestamp}"
  vm_memory      = 4096
  disk_size      = 30720

  ssh_username            = "testuser"
  ssh_password            = "alpine"
  ssh_wait_timeout        = "60000s"
  ssh_handshake_attempts  = 10000

  output_directory = "packer-alpine-iso"
  keep_vm          = "always"
}

build {
  sources = ["xenserver-iso.alpine"]
  # Wait for cloud-init to finish everything. We need to do this as a
  # packer provisioner to prevent packer-plugin-xenserver from shutting
  # the VM down before all cloud-init processing is complete.
  provisioner "shell" {
    inline = [
      "echo alpine | sudo -S cloud-init status --wait"
    ]
    valid_exit_codes = [0, 2]
  }
}
