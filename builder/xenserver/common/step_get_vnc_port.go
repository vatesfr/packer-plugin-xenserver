package common

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"net"
)

type StepGetVNCPort struct {
	listener net.Listener
}

func (self *StepGetVNCPort) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Step: forward the instances VNC")

	location, err := GetVNCConsoleLocation(state)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	forwardingListener, err := CreateCustomPortForwarding(func() (net.Conn, error) {
		return CreateVNCConnection(state, location)
	})

	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	self.listener = forwardingListener

	ui.Say(fmt.Sprintf("VNC available on vnc://%s", self.listener.Addr().String()))

	return multistep.ActionContinue
}

func (self *StepGetVNCPort) Cleanup(state multistep.StateBag) {
	self.listener.Close()
}
