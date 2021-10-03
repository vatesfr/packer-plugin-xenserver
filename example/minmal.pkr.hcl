packer {
  required_plugins {
   xenserver= {
      version = ">= v0.3.2"
      source = "github.com/ddelnano/xenserver"
    }
  }
}

source "xenserver-iso" "minimal" {
  iso_url          =  "http://tinycorelinux.net/12.x/x86/release/Core-12.0.iso"
  iso_checksum     = "md5:157a4b94730c8fac2bded64ad378d8ad"
  tools_iso_name = "guest-tools.iso"

  
  disk_size = "10"
  
  remote_host = "localhost"
  remote_port = 42795
  remote_ssh_port = 45201
  remote_username = "root"
  remote_password = "changeme"
  
  ssh_username = "tc"
  ssh_password = "test"
  ssh_timeout = "20m"
  
  sr_iso_name = "Local storage"

  sr_name = "Local storage"
  
  vm_name = "minimal"

  shutdown_command = "sudo poweroff"

  boot_wait = "5s"
  
  http_content = {
    "/" = "Hello, World"
  }
  
  boot_command = [
     "<enter>",
     "<wait10>",
     "sh <<'EOF'<enter>",
     "tce-load -wi openssh<enter>",
     "sudo cp /usr/local/etc/ssh/sshd_config.orig /usr/local/etc/ssh/sshd_config<enter>",
     "sudo /usr/local/etc/init.d/openssh start<enter>",
     "echo tc:test | sudo chpasswd<enter>",
     "wget -O- http://{{ .HTTPIP }}:{{ .HTTPPort }}/<enter>",
     
     # EOF Heredoc
     "EOF<enter>"
  ]  
}

build {
  sources = ["sources.xenserver-iso.minimal"]
  
  provisioner "shell" {
    inline = [
      "whoami"
    ]
  }
  
  provisioner "shell" {
    expect_disconnect = true

    inline = [
      "nohup sudo /usr/local/etc/init.d/openssh restart"
    ]
  }
  
  provisioner "shell" {
    inline = [
      "whoami"
    ]
  }
}
