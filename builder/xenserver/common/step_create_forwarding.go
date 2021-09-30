package common

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type ForwardTarget struct {
	Host func(multistep.StateBag) (string, error)
	Port func(multistep.StateBag) (int, error)
	Key  string

	listener net.Listener
}

type StepCreateForwarding struct {
	Targets []ForwardTarget
}

func (self *StepCreateForwarding) close() {
	for _, target := range self.Targets {
		if target.listener != nil {
			target.listener.Close()
		}
	}
}

func (self *StepCreateForwarding) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	proxyAddress, err := GetXenProxyAddress(state)

	if err != nil {
		err := fmt.Errorf("could not get proxy address: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	for _, target := range self.Targets {
		host, err := target.Host(state)

		if err != nil {
			self.close()
			err := fmt.Errorf("could not get host for forwarding: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		port, err := target.Port(state)

		if err != nil {
			self.close()
			err := fmt.Errorf("could not get port for forwarding: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		address := fmt.Sprintf("%s:%d", host, port)

		target.listener, err = CreatePortForwarding(proxyAddress, address)

		if err != nil {
			self.close()
			err := fmt.Errorf("could not create forwarding: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		listenerAddr := target.listener.Addr().(*net.TCPAddr)

		state.Put(target.Key, listenerAddr)
	}

	return multistep.ActionContinue
}

func (self *StepCreateForwarding) Cleanup(_ multistep.StateBag) {
	self.close()
}

func GetForwardedHost(state multistep.StateBag, key string) (string, error) {
	address, ok := state.GetOk(key)
	if !ok {
		return "", fmt.Errorf("key '%s' does not exist", key)
	}

	return address.(*net.TCPAddr).IP.String(), nil
}

func GetForwardedPort(state multistep.StateBag, key string) (int, error) {
	address, ok := state.GetOk(key)
	if !ok {
		return 0, fmt.Errorf("key '%s' does not exist", key)
	}

	return address.(*net.TCPAddr).Port, nil
}
