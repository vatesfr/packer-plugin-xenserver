package common

import (
	"context"
	"fmt"
	config2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/proxy"
	ssh2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/ssh"
	"golang.org/x/crypto/ssh"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCreateProxy struct {
	sshClient   *ssh.Client
	proxyServer proxy.XenProxy
}

func (self *StepCreateProxy) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("commonconfig").(config2.CommonConfig)
	ui := state.Get("ui").(packer.Ui)

	var err error
	self.sshClient, err = ssh2.ConnectSSH(config.HostIp, config.HostSSHPort, config.Username, config.Password)
	if err != nil {
		err := fmt.Errorf("error connecting to hypervisor with ssh: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	self.proxyServer = proxy.CreateProxy(config.SkipNatMapping, ssh2.SSHDialer(self.sshClient))

	err = self.proxyServer.Start()
	if err != nil {
		err := fmt.Errorf("error creating socks proxy server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("xen_proxy", self.proxyServer)

	return multistep.ActionContinue
}

func (self *StepCreateProxy) Cleanup(_ multistep.StateBag) {
	if self.proxyServer != nil {
		err := self.proxyServer.Close()
		if err != nil {
			log.Printf("error cleaning up proxy server: %v", err)
			return
		}
	}

	if self.sshClient != nil {
		err := self.sshClient.Close()
		if err != nil {
			log.Printf("error cleaning up ssh client: %v", err)
			return
		}
	}
}
