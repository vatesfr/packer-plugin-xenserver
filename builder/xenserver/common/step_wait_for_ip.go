package common

import (
	"context"
	"fmt"
	"net"
	"strings"
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

	var ip string
	err = InterruptibleWait{
		Timeout:           self.Timeout,
		PredicateInterval: 5 * time.Second,
		Predicate: func() (result bool, err error) {

			if config.IPGetter == "auto" || config.IPGetter == "http" {

				// Snoop IP from HTTP fetch
				select {
				case ip = <-self.Chan:
					ui.Message(fmt.Sprintf("Got IP '%s' from HTTP request", ip))
					return true, nil
				default:
				}

			}

			ping := func(address string) bool {
				protocol := "ip4:icmp"
				if strings.Contains(address, ":") {
					protocol = "ip6:icmp"
				}
				_, err = net.DialTimeout(protocol, address, time.Second*2)
				return err == nil
			}

			trynetwork := func(network string, address string) bool {
				if address != "" && ping(address) {
					ui.Message(fmt.Sprintf("Got IP '%s' (%s) from XenServer tools.", address, network))
					return true
				}
				ui.Message(fmt.Sprintf("Got IP '%s' (%s) from XenServer tools. Not accessible.", address, network))
				return false
			}

			if config.IPGetter == "auto" || config.IPGetter == "tools" {

				// Look for PV IP
				m, err := c.client.VM.GetGuestMetrics(c.session, instance)
				if err != nil {
					return false, err
				}
				if m != "" {
					metrics, err := c.client.VMGuestMetrics.GetRecord(c.session, m)
					if err != nil {
						return false, err
					}
					networks := metrics.Networks
					var ok bool
					if ip, ok = networks["0/ip"]; ok {
						if trynetwork("0/ip", ip) {
							return true, nil
						}
					}
					for _, protocol := range []string{"ipv4", "ipv6"} {
						for network, address := range networks {
							if strings.Contains(network, protocol) && trynetwork(network, address) {
								ip = address
								return true, nil
							}
						}
					}
				}

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
