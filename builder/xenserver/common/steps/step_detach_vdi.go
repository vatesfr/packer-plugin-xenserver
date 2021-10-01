package steps

import (
	"context"
	"fmt"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepDetachVdi struct {
	VdiUuidKey string
}

func (self *StepDetachVdi) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*xen.Connection)

	var vdiUuid string
	if vdiUuidRaw, ok := state.GetOk(self.VdiUuidKey); ok {
		vdiUuid = vdiUuidRaw.(string)
	} else {
		log.Printf("Skipping detach of '%s'", self.VdiUuidKey)
		return multistep.ActionContinue
	}

	vdi, err := c.GetClient().VDI.GetByUUID(c.GetSessionRef(), vdiUuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to get VDI from UUID '%s': %s", vdiUuid, err.Error()))
		return multistep.ActionHalt
	}

	uuid := state.Get("instance_uuid").(string)
	instance, err := c.GetClient().VM.GetByUUID(c.GetSessionRef(), uuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to get VM from UUID '%s': %s", uuid, err.Error()))
		return multistep.ActionHalt
	}

	err = xen.DisconnectVdi(c, instance, vdi)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to detach VDI '%s': %s", vdiUuid, err.Error()))
		//return multistep.ActionHalt
		return multistep.ActionContinue
	}

	log.Printf("Detached VDI '%s'", vdiUuid)

	return multistep.ActionContinue
}

func (self *StepDetachVdi) Cleanup(state multistep.StateBag) {}
