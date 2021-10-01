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
	config2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/vnc"
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
	config := state.Get("commonconfig").(config2.CommonConfig)
	ui := state.Get("ui").(packer.Ui)
	httpPort := state.Get("http_port").(int)

	if config.VNCConfig.DisableVNC {
		return multistep.ActionContinue
	}

	// skip this step if we have nothing to type
	if len(config.BootCommand) == 0 {
		return multistep.ActionContinue
	}

	var httpIP string
	if config.HTTPAddress != "0.0.0.0" {
		httpIP = config.HTTPAddress
	} else {
		httpIP = state.Get("http_ip").(string)
	}

	location, err := vnc.GetVNCConsoleLocation(state)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Connecting to the VM console VNC over xapi via %s", location))

	vncClient, err := vnc.CreateVNCClient(state, location)

	if err != nil {
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	defer vncClient.Close()

	log.Printf("Connected to the VNC console: %s", vncClient.DesktopName)

	self.Ctx.Data = &bootCommandTemplateData{
		config.VMName,
		httpIP,
		uint(httpPort),
	}

	vncDriver := bootcommand.NewVNCDriver(vncClient, config.VNCConfig.BootKeyInterval)

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
