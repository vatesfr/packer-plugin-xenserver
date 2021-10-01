package steps

import (
	"context"
	"fmt"
	config2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepStartVmPaused struct {
	VmCleanup
}

func (self *StepStartVmPaused) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {

	c := state.Get("client").(*xen.Connection)
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(config2.CommonConfig)

	ui.Say("Step: Start VM Paused")

	uuid := state.Get("instance_uuid").(string)
	instance, err := c.GetClient().VM.GetByUUID(c.GetSessionRef(), uuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to get VM from UUID '%s': %s", uuid, err.Error()))
		return multistep.ActionHalt
	}

	// note that here "cd" means boot from hard drive ('c') first, then CDROM ('d')
	err = c.GetClient().VM.SetHVMBootPolicy(c.GetSessionRef(), instance, "BIOS order")

	if err != nil {
		ui.Error(fmt.Sprintf("Unable to set HVM boot params: %s", err.Error()))
		return multistep.ActionHalt
	}

	err = c.GetClient().VM.SetHVMBootParams(c.GetSessionRef(), instance, map[string]string{"order": "cd", "firmware": config.Firmware})
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to set HVM boot params: %s", err.Error()))
		return multistep.ActionHalt
	}

	err = c.GetClient().VM.Start(c.GetSessionRef(), instance, true, false)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to start VM with UUID '%s': %s", uuid, err.Error()))
		return multistep.ActionHalt
	}

	domid, err := c.GetClient().VM.GetDomid(c.GetSessionRef(), instance)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to get domid of VM with UUID '%s': %s", uuid, err.Error()))
		return multistep.ActionHalt
	}
	state.Put("domid", domid)

	return multistep.ActionContinue
}
