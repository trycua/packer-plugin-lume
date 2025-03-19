package main

import (
	"fmt"
	"os"

	"github.com/trycua/packer-plugin-lume/builder/lume"
	"github.com/trycua/packer-plugin-lume/version"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("cli", new(lume.Builder))
	pps.SetVersion(version.PluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
