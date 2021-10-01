package common

import (
	"context"
	"fmt"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"
	"log"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	xenapi "github.com/terra-farm/go-xen-api-client"
)

type StepShutdown struct{}

func (StepShutdown) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("commonconfig").(CommonConfig)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*xen.Connection)
	instance_uuid := state.Get("instance_uuid").(string)

	instance, err := c.GetClient().VM.GetByUUID(c.GetSessionRef(), instance_uuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Could not get VM with UUID '%s': %s", instance_uuid, err.Error()))
		return multistep.ActionHalt
	}

	ui.Say("Step: Shutting down VM")

	// Shutdown the VM
	success := func() bool {
		if config.ShutdownCommand != "" {
			comm := state.Get("communicator").(packer.Communicator)
			ui.Say("Gracefully halting virtual machine...")
			log.Printf("Executing shutdown command: %s", config.ShutdownCommand)

			cmd := &packer.RemoteCmd{Command: config.ShutdownCommand}
			if err := cmd.RunWithUi(ctx, comm, ui); err != nil {
				ui.Error(fmt.Sprintf("Failed to send shutdown command: %s", err.Error()))
				return false
			}

			ui.Message(fmt.Sprintf("Waiting for VM to enter Halted state... Timeout after %s",
				config.ShutdownTimeout.String()))

			err = InterruptibleWait{
				Predicate: func() (bool, error) {
					power_state, err := c.GetClient().VM.GetPowerState(c.GetSessionRef(), instance)
					return power_state == xenapi.VMPowerStateHalted, err
				},
				PredicateInterval: 5 * time.Second,
				Timeout:           config.ShutdownTimeout,
			}.Wait(state)

			if err != nil {
				ui.Error(fmt.Sprintf("Error waiting for VM to halt: %s", err.Error()))
				return false
			}

		} else {
			ui.Message("Attempting to cleanly shutdown the VM...")

			err = c.GetClient().VM.CleanShutdown(c.GetSessionRef(), instance)
			if err != nil {
				ui.Error(fmt.Sprintf("Could not shut down VM: %s", err.Error()))
				return false
			}

		}
		return true
	}()

	if !success {
		ui.Say("WARNING: Forcing hard shutdown of the VM...")
		err = c.GetClient().VM.HardShutdown(c.GetSessionRef(), instance)
		if err != nil {
			ui.Error(fmt.Sprintf("Could not hard shut down VM -- giving up: %s", err.Error()))
			return multistep.ActionHalt
		}
	}

	ui.Message("Successfully shut down VM")
	return multistep.ActionContinue
}

func (StepShutdown) Cleanup(state multistep.StateBag) {}
