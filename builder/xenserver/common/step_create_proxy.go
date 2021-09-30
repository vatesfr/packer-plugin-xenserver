package common

import (
	"context"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCreateProxy struct {
	sshClient     *ssh.Client
	socksListener net.Listener
}

func (self *StepCreateProxy) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("commonconfig").(CommonConfig)
	ui := state.Get("ui").(packer.Ui)

	var err error
	self.sshClient, err = connectSSH(config.HostIp, config.HostSSHPort, config.Username, config.Password)
	if err != nil {
		err := fmt.Errorf("error connecting to hypervisor with ssh: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	proxyServer, err := setupProxyServer(sshDialer(self.sshClient))
	if err != nil {
		err := fmt.Errorf("error creating socks proxy server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	self.socksListener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		err := fmt.Errorf("error creating socks listener: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	go func() {
		err := proxyServer.Serve(self.socksListener)
		if err != nil {
			log.Printf("error in proxy server: %v", err)
		}
	}()

	state.Put("xen_proxy_address", self.socksListener.Addr().String())

	return multistep.ActionContinue
}

func (self *StepCreateProxy) Cleanup(_ multistep.StateBag) {
	if self.socksListener != nil {
		err := self.socksListener.Close()
		if err != nil {
			log.Printf("error cleaning up socket listener: %v", err)
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
