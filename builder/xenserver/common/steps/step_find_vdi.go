package steps

import (
	"context"
	"fmt"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepFindVdi struct {
	VdiName       string
	ImagePathFunc func() string
	VdiUuidKey    string
}

func (self *StepFindVdi) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*xen.Connection)

	// Ignore if VdiName is not specified
	if self.VdiName == "" {
		return multistep.ActionContinue
	}

	vdis, err := c.GetClient().VDI.GetByNameLabel(c.GetSessionRef(), self.VdiName)

	switch {
	case len(vdis) == 0:
		ui.Error(fmt.Sprintf("Couldn't find a VDI named '%s'", self.VdiName))
		return multistep.ActionHalt
	case len(vdis) > 1:
		ui.Error(fmt.Sprintf("Found more than one VDI with name '%s'. Name must be unique", self.VdiName))
		return multistep.ActionHalt
	}

	vdi := vdis[0]

	vdiUuid, err := c.GetClient().VDI.GetUUID(c.GetSessionRef(), vdi)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to get UUID of VDI '%s': %s", self.VdiName, err.Error()))
		return multistep.ActionHalt
	}
	state.Put(self.VdiUuidKey, vdiUuid)

	return multistep.ActionContinue
}

func (self *StepFindVdi) Cleanup(state multistep.StateBag) {}
