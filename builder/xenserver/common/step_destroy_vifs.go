package common

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepDestroyVIFs struct{}

func (self *StepDestroyVIFs) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("client").(*Connection)
	config := state.Get("config").(Config)
	ui := state.Get("ui").(packer.Ui)

	if !config.DestroyVIFs {
		log.Printf("Not destroying VIFs")
		return multistep.ActionContinue
	}

	ui.Say("Step: Destroy VIFs")

	uuid := state.Get("instance_uuid").(string)
	instance, err := c.client.VM.GetByUUID(c.session, uuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to get VM from UUID '%s': %s", uuid, err.Error()))
		return multistep.ActionHalt
	}

	vifs, err := c.client.VM.GetVIFs(c.session, instance)
	if err != nil {
		ui.Error(fmt.Sprintf("Error getting VIFs: %s", err.Error()))
		return multistep.ActionHalt
	}

	for _, vif := range vifs {
		err = c.client.VIF.Destroy(c.session, vif)
		if err != nil {
			ui.Error(fmt.Sprintf("Error destroying VIF: %s", err.Error()))
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

func (self *StepDestroyVIFs) Cleanup(state multistep.StateBag) {}
