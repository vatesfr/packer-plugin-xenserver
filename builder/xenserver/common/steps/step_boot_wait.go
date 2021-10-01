package steps

import (
	"context"
	"fmt"
	config2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/util"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepBootWait struct{}

func (self *StepBootWait) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("client").(*xen.Connection)
	config := state.Get("commonconfig").(config2.CommonConfig)
	ui := state.Get("ui").(packer.Ui)

	instance, _ := c.GetClient().VM.GetByUUID(c.GetSessionRef(), state.Get("instance_uuid").(string))
	ui.Say("Unpausing VM " + state.Get("instance_uuid").(string))
	xen.Unpause(c, instance)

	if int64(config.BootWait) > 0 {
		ui.Say(fmt.Sprintf("Waiting %s for boot...", config.BootWait))
		err := util.InterruptibleWait{Timeout: config.BootWait}.Wait(state)
		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}
	return multistep.ActionContinue
}

func (self *StepBootWait) Cleanup(state multistep.StateBag) {}
