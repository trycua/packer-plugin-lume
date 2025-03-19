package lume

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCloneVM struct{}

func (s *stepCloneVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	var commonArgs []string

	ui.Say("Cloning virtual machine...")

	cmdArgs := []string{"clone", config.VMBaseName, config.VMName}
	cmdArgs = append(cmdArgs, commonArgs...)

	if _, err := LumeExec().
		WithContext(ctx).
		WithPackerUI(ui).
		WithArgs(cmdArgs...).
		Do(); err != nil {
		err := fmt.Errorf("Error cloning VM: %s", err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	state.Put("vm_name", config.VMName)

	return multistep.ActionContinue
}

func (s *stepCloneVM) Cleanup(state multistep.StateBag) {
	// nothing to clean up
}
