package main

import (
	"fmt"
	"os"

	"github.com/disruptivemindseu/packer-plugin-xcp/builder/xcp/common"
	"github.com/disruptivemindseu/packer-plugin-xcp/version"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("iso", new(common.Builder))
	pps.SetVersion(version.PluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
