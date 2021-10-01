package steps

import (
	"context"
	"fmt"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepSetVmToTemplate struct{}

func (StepSetVmToTemplate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*xen.Connection)
	instance_uuid := state.Get("instance_uuid").(string)

	instance, err := c.GetClient().VM.GetByUUID(c.GetSessionRef(), instance_uuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Could not get VM with UUID '%s': %s", instance_uuid, err.Error()))
		return multistep.ActionHalt
	}

	err = c.GetClient().VM.SetIsATemplate(c.GetSessionRef(), instance, true)

	if err != nil {
		ui.Error(fmt.Sprintf("failed to set VM '%s' as a template with error: %v", instance_uuid, err))
		return multistep.ActionHalt
	}

	ui.Message("Successfully set VM as a template")
	return multistep.ActionContinue
}

func (StepSetVmToTemplate) Cleanup(state multistep.StateBag) {}
