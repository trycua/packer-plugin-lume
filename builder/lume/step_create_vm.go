package lume

import (
	"context"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateVM struct{}

func (s *stepCreateVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Creating virtual machine...")

	createArguments := []string{"create"}
	// ?
	// bash-3.2$ lume create -h
	// Error: Missing expected argument '<name>'
	// Help:  <name>  Name for the virtual machine
	// Usage: lume create <name> [--os <os>] [--cpu <cpu>] [--memory <memory>] [--disk-size <disk-size>] [--display <display>] [--ipsw <ipsw>]
	//   See 'lume create --help' for more information.
	if config.IPSW != "" {
		createArguments = append(createArguments, "--ipsw", config.IPSW)
	}
	if config.CpuCount != 0 {
		createArguments = append(createArguments, "--cpu", strconv.Itoa(int(config.CpuCount)))
	}
	if len(config.Memory) > 0 {
		createArguments = append(createArguments, "--memory", config.Memory)
	}
	if len(config.DiskSize) > 0 {
		createArguments = append(createArguments, "--disk-size", config.DiskSize)
	}

	createArguments = append(createArguments, config.VMName)

	outChan, errChan := LumeExec().
		WithContext(ctx).
		WithPackerUI(ui).
		WithArgs(createArguments...).
		DoChanPty()

	// Consume stdout lines in a goroutine or via select.
	go func() {
		for line := range outChan {
			if line != nil {
				// process stdout line
				ui.Message(*line)
			}
		}
	}()

	if err, ok := <-errChan; ok {
		state.Put("error", err)
		ui.Errorf("[Error] While creating vm: %v", err)
		return multistep.ActionHalt
	}

	state.Put("vm_name", config.VMName)

	return multistep.ActionContinue
}

func (s *stepCreateVM) Cleanup(state multistep.StateBag) {
	// nothing to clean up
}
