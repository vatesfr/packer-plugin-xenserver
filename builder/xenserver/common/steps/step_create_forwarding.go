package steps

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/proxy"
)

type ForwardTarget struct {
	Host func(multistep.StateBag) (string, error)
	Port func(multistep.StateBag) (int, error)
	Key  string

	forwarding proxy.ProxyForwarding
}

type forwardingInfo struct {
	host string
	port int
}

type StepCreateForwarding struct {
	Targets []ForwardTarget
}

func (self *StepCreateForwarding) close() {
	for _, target := range self.Targets {
		if target.forwarding != nil {
			target.forwarding.Close()
		}
	}
}

func (self *StepCreateForwarding) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	xenProxy := state.Get("xen_proxy").(proxy.XenProxy)

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

		forwarding := xenProxy.CreateForwarding(host, port)
		err = forwarding.Start()

		if err != nil {
			self.close()
			err := fmt.Errorf("could not create forwarding: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		info := forwardingInfo{
			host: forwarding.GetServiceHost(),
			port: forwarding.GetServicePort(),
		}

		state.Put(target.Key, info)
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

	return address.(forwardingInfo).host, nil
}

func GetForwardedPort(state multistep.StateBag, key string) (int, error) {
	address, ok := state.GetOk(key)
	if !ok {
		return 0, fmt.Errorf("key '%s' does not exist", key)
	}

	return address.(forwardingInfo).port, nil
}
