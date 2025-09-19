package common

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepWaitForIP struct {
	VmCleanup
	Chan    <-chan string
	Timeout time.Duration
}

func (self *StepWaitForIP) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("client").(*Connection)
	config := state.Get("commonconfig").(CommonConfig)

	ui.Say("Step: Wait for VM's IP to become known to us.")

	uuid := state.Get("instance_uuid").(string)
	instance, err := c.client.VM.GetByUUID(c.session, uuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Unable to get VM from UUID '%s': %s", uuid, err.Error()))
		return multistep.ActionHalt
	}

	go func(c Connection, ui packer.Ui, config CommonConfig) {
		state.Put("instance_ssh_address", "")
		var ip string

		for true {
			time.Sleep(500 * time.Millisecond)

			if config.IPGetter == "auto" || config.IPGetter == "http" {
				// Snoop IP from HTTP fetch
				select {
				case new_ip := <-self.Chan:
					if ip != new_ip {
						ip = new_ip
						ui.Message(fmt.Sprintf("Got IP '%s' from HTTP request", ip))
						state.Put("instance_ssh_address", ip)
					}
				default:
				}
			}

			if config.IPGetter == "auto" || config.IPGetter == "tools" {
				// Look for PV IP
				m, err := c.client.VM.GetGuestMetrics(c.session, instance)
				if err != nil {
					continue
				}
				if m != "" {
					metrics, err := c.client.VMGuestMetrics.GetRecord(c.session, m)
					if err != nil {
						continue
					}
					networks := metrics.Networks
					if new_ip, ok := networks["0/ip"]; ok {
						if new_ip != "" && ip != new_ip {
							ip = new_ip
							ui.Message(fmt.Sprintf("Got IP '%s' from XenServer tools", ip))
							state.Put("instance_ssh_address", ip)
						}
					}
				}

			}
		}
	}(*c, ui, config)

	time.Sleep(config.DhcpWait)

	var ip string

	err = InterruptibleWait{
		Timeout:           self.Timeout,
		PredicateInterval: 5 * time.Second,
		Predicate: func() (result bool, err error) {

			var ok bool
			if ip, ok = state.Get("instance_ssh_address").(string); ok && ip != "" {
				return true, nil
			}

			return false, nil
		},
	}.Wait(state)
	if err != nil {
		ui.Error(fmt.Sprintf("Could not get IP address of VM: %s", err.Error()))
		// @todo: give advice on what went wrong (no HTTP server? no PV drivers?)
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Got IP address '%s'", ip))
	state.Put("instance_ssh_address", ip)

	return multistep.ActionContinue
}

func InstanceSSHIP(state multistep.StateBag) (string, error) {
	ip := state.Get("instance_ssh_address").(string)
	return ip, nil
}

func InstanceSSHPort(state multistep.StateBag) (int, error) {
	return 22, nil
}
