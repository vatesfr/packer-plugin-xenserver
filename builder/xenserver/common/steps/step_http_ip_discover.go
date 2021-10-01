package steps

import (
	"context"
	"fmt"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/ssh"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// Step to discover the http ip
// which guests use to reach the vm host
// To make sure the IP is set before boot command and http server steps
type StepHTTPIPDiscover struct{}

func (s *StepHTTPIPDiscover) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)

	// find local ip
	envVar, err := ssh.ExecuteApiHostSSHCmd(state, "echo $SSH_CLIENT")
	if err != nil {
		ui.Error(fmt.Sprintf("Error detecting local IP: %s", err))
		return multistep.ActionHalt
	}
	if envVar == "" {
		ui.Error("Error detecting local IP: $SSH_CLIENT was empty")
		return multistep.ActionHalt
	}
	hostIP := strings.Split(envVar, " ")[0]
	ui.Message(fmt.Sprintf("Found local IP: %s", hostIP))

	state.Put("http_ip", hostIP)

	return multistep.ActionContinue
}

func (s *StepHTTPIPDiscover) Cleanup(state multistep.StateBag) {}
