package common

/*
Heavily borrowed from builder/quemu/step_type_boot_command.go
*/

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"log"
)

type bootCommandTemplateData struct {
	Name     string
	HTTPIP   string
	HTTPPort uint
}

type StepTypeBootCommand struct {
	Ctx interpolate.Context
}

func (self *StepTypeBootCommand) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("commonconfig").(CommonConfig)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*Connection)
	httpPort := state.Get("http_port").(int)

	var httpIP string
	if config.HTTPAddress != "0.0.0.0" {
		httpIP = config.HTTPAddress
	} else {
		httpIP = state.Get("http_ip").(string)
	}

	// skip this step if we have nothing to type
	if len(config.BootCommand) == 0 {
		return multistep.ActionContinue
	}

	vmRef, err := c.client.VM.GetByNameLabel(c.session, config.VMName)

	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if len(vmRef) != 1 {
		ui.Error(fmt.Sprintf("expected to find a single VM, instead found '%d'. Ensure the VM name is unique", len(vmRef)))
	}

	consoles, err := c.client.VM.GetConsoles(c.session, vmRef[0])
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if len(consoles) != 1 {
		ui.Error(fmt.Sprintf("expected to find a VM console, instead found '%d'. Ensure there is only one console", len(consoles)))
		return multistep.ActionHalt
	}

	location, err := c.client.Console.GetLocation(c.session, consoles[0])

	ui.Say(fmt.Sprintf("Connecting to the VM console VNC over xapi via %s", location))

	vncConnectionWrapper, err := ConnectVNC(state, location)

	if err != nil {
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	defer vncConnectionWrapper.Close()

	log.Printf("Connected to the VNC console: %s", vncConnectionWrapper.Client.DesktopName)

	self.Ctx.Data = &bootCommandTemplateData{
		config.VMName,
		httpIP,
		uint(httpPort),
	}

	vncDriver := bootcommand.NewVNCDriver(vncConnectionWrapper.Client, config.VNCConfig.BootKeyInterval)

	ui.Say("Typing boot commands over VNC...")

	command, err := interpolate.Render(config.VNCConfig.FlatBootCommand(), &self.Ctx)

	if err != nil {
		err := fmt.Errorf("Error preparing boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	seq, err := bootcommand.GenerateExpressionSequence(command)

	if err != nil {
		err := fmt.Errorf("Error generating boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if err := seq.Do(ctx, vncDriver); err != nil {
		err := fmt.Errorf("Error running boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Finished typing.")

	return multistep.ActionContinue
}

func (self *StepTypeBootCommand) Cleanup(multistep.StateBag) {}
