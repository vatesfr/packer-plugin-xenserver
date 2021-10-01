package steps

import (
	"fmt"
	config2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type VmCleanup struct{}

func (self *VmCleanup) Cleanup(state multistep.StateBag) {
	config := state.Get("commonconfig").(config2.CommonConfig)
	c := state.Get("client").(*xen.Connection)

	if config.ShouldKeepVM(state) {
		return
	}

	uuid := state.Get("instance_uuid").(string)
	instance, err := c.GetClient().VM.GetByUUID(c.GetSessionRef(), uuid)
	if err != nil {
		log.Printf(fmt.Sprintf("Unable to get VM from UUID '%s': %s", uuid, err.Error()))
		return
	}

	err = c.GetClient().VM.HardShutdown(c.GetSessionRef(), instance)
	if err != nil {
		log.Printf(fmt.Sprintf("Unable to force shutdown VM '%s': %s", uuid, err.Error()))
	}
}
