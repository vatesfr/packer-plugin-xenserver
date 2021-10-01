package common

import (
	"context"
	"fmt"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	xsclient "github.com/terra-farm/go-xen-api-client"
)

type StepAttachVdi struct {
	VdiUuidKey string
	VdiType    xsclient.VbdType

	vdi xsclient.VDIRef
}

func (self *StepAttachVdi) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*xen.Connection)

	log.Printf("Running attach vdi for key %s\n", self.VdiUuidKey)
	var vdiUuid string
	if vdiUuidRaw, ok := state.GetOk(self.VdiUuidKey); ok {
		vdiUuid = vdiUuidRaw.(string)
	} else {
		log.Printf("Skipping attach of '%s'", self.VdiUuidKey)
		return multistep.ActionContinue
	}

	var err error
	self.vdi, err = c.GetClient().VDI.GetByUUID(c.GetSessionRef(), vdiUuid)
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

	err = xen.ConnectVdi(c, instance, self.vdi, self.VdiType)
	if err != nil {
		ui.Error(fmt.Sprintf("Error attaching VDI '%s': '%s'", vdiUuid, err.Error()))
		return multistep.ActionHalt
	}

	log.Printf("Attached VDI '%s'", vdiUuid)

	return multistep.ActionContinue
}

func (self *StepAttachVdi) Cleanup(state multistep.StateBag) {
	config := state.Get("commonconfig").(CommonConfig)
	c := state.Get("client").(*xen.Connection)
	if config.ShouldKeepVM(state) {
		return
	}

	if self.vdi == "" {
		return
	}

	uuid := state.Get("instance_uuid").(string)
	vmRef, err := c.GetClient().VM.GetByUUID(c.GetSessionRef(), uuid)
	if err != nil {
		log.Printf("Unable to get VM from UUID '%s': %s", uuid, err.Error())
		return
	}

	vdiUuid := state.Get(self.VdiUuidKey).(string)

	err = xen.DisconnectVdi(c, vmRef, self.vdi)
	if err != nil {
		log.Printf("Unable to disconnect VDI '%s': %s", vdiUuid, err.Error())
		return
	}
	log.Printf("Detached VDI '%s'", vdiUuid)
}
