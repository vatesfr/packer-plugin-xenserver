package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	xsclient "github.com/terra-farm/go-xen-api-client"
)

// Removes old templates with the same name as the current build.
//
// # Inputs (via multistep.StateBag):
//   - "existing_templates": []xsclient.VMRef, set by StepGetVmTemplate.
//   - "ui": packer.Ui, used for user messages.
//   - "client": *Connection, used to interact with XenServer.
//
// # Behavior:
//   - If Force is true, deletes all existing templates.
//   - If Force is false or no templates exist, proceeds without deletion.
type StepCleanUpTemplate struct {
	Force bool
}

func (self *StepCleanUpTemplate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*Connection)

	existing_templates := state.Get("existing_templates")

	ui.Say("Step: Cleaning up templates")

	if existing_templates != nil {
		templates := existing_templates.([]xsclient.VMRef)

		if self.Force {
			ui.Message(fmt.Sprintf("Deleting %d templates since -force was specified!", len(templates)))
			for _, template := range templates {
				c.client.VM.Destroy(c.session, template)
			}
			return multistep.ActionContinue
		} else {
			ui.Message(fmt.Sprintf("Ignoring %d templates since -force was NOT specified!", len(templates)))
			return multistep.ActionContinue
		}
	} else {
		ui.Message("No existing templates to clean up.")
		return multistep.ActionContinue
	}
}

func (self *StepCleanUpTemplate) Cleanup(state multistep.StateBag) {}
