package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	xsclient "github.com/terra-farm/go-xen-api-client"
)

// Looks up existing templates using the same name as the current build and stores them in the state bag.
//
// # Inputs (via multistep.StateBag):
//   - "ui": packer.Ui, used for user messages.
//   - "client": *Connection, the XenServer connection.
//   - "config": Config, contains the VMName to look up.
//
// Output:
//   - If templates are found, stores them in the StateBag under the key "existing_templates"
type StepGetVmTemplate struct{}

func (StepGetVmTemplate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*Connection)
	config := state.Get("config").(Config)
	vmRefs, err := c.client.VM.GetByNameLabel(c.session, config.VMName)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to get VMs with name '%s': %v", config.VMName, err.Error()))
		return multistep.ActionHalt
	}

	ui.Say("Step: Checking for pre-existing templates")

	var templates []xsclient.VMRef

	// Figure out which VMs with the same name as the current build are templates.
	for _, vm := range vmRefs {
		isTemplate, err := c.client.VM.GetIsATemplate(c.session, vm)

		if err != nil {
			ui.Error(fmt.Sprintf("Failed to check if existing VM '%s' is a template with error: %v", vm, err.Error()))
			return multistep.ActionHalt
		}

		if !isTemplate {
			continue
		}

		templates = append(templates, vm)
	}

	if len(templates) == 0 {
		ui.Message("Didn't find any existing templates. Proceeding with build.")
		return multistep.ActionContinue
	} else {
		ui.Message(fmt.Sprintf("Found %d existing templates with current build name: '%s'", len(templates), config.VMName))
		state.Put("existing_templates", templates)
	}
	return multistep.ActionContinue
}

func (StepGetVmTemplate) Cleanup(state multistep.StateBag) {}
