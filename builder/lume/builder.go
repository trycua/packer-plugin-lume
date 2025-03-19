//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package lume

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

const BuilderId = "lume.builder"

type Config struct {
	common.PackerConfig    `mapstructure:",squash"`
	bootcommand.VNCConfig  `mapstructure:",squash"`
	commonsteps.HTTPConfig `mapstructure:",squash"`
	CommunicatorConfig     communicator.Config `mapstructure:",squash"`

	IPSW       string `mapstructure:"ipsw"`
	VMBaseName string `mapstructure:"vm_base_name"`
	VMName     string `mapstructure:"vm_name"`

	CpuCount          uint8         `mapstructure:"cpu_count"`
	VNCGraceTime      time.Duration `mapstructure:"vnc_grace_time"`
	DiskSize          string        `mapstructure:"disk_size"`
	RecoveryPartition string        `mapstructure:"recovery_partition"`
	Display           string        `mapstructure:"display"`
	Headless          bool          `mapstructure:"headless"`
	Memory            string        `mapstructure:"memory"`
	RecoveryMode      bool          `mapstructure:"recovery_mode"`
	Rosetta           string        `mapstructure:"rosetta"`
	RunExtraArgs      []string      `mapstructure:"run_extra_args"`
	IpExtraArgs       []string      `mapstructure:"ip_extra_args"`

	ctx interpolate.Context
}

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) (generatedVars []string, warnings []string, err error) {
	err = config.Decode(&b.config, &config.DecodeOpts{
		PluginType:  "packer.builder.lume",
		Interpolate: true,
		InterpolateFilter: &interpolate.RenderFilter{
			// Postpone the boot_command interpolation because
			// we don't know the HTTPIP and HTTPPort yet
			Exclude: []string{"boot_command"},
		},
		InterpolateContext: &b.config.ctx,
	}, raws...)
	if err != nil {
		return nil, nil, err
	}

	fromArgs := []bool{
		b.config.IPSW != "",
		b.config.VMBaseName != "",
	}

	fromArgsSet := 0
	for _, v := range fromArgs {
		if v {
			fromArgsSet++
			if fromArgsSet > 1 {
				return nil, nil, fmt.Errorf("ipsw and vm_base_name are mutually exclusive")
			}
		}
	}

	if errs := b.config.CommunicatorConfig.Prepare(&b.config.ctx); len(errs) != 0 {
		return nil, nil, packer.MultiErrorAppend(nil, errs...)
	}

	return nil, nil, nil
}

func (b *Builder) Run(ctx context.Context, ui packer.Ui, hook packer.Hook) (packer.Artifact, error) {
	steps := []multistep.Step{
		new(stepCleanVM), // cleanup the VM if the build is cancelled or halted
	}

	if b.config.HTTPDir != "" || len(b.config.HTTPContent) != 0 {
		if errs := b.config.HTTPConfig.Prepare(interpolate.NewContext()); len(errs) != 0 {
			return nil, packer.MultiErrorAppend(nil, errs...)
		}

		steps = append(steps, commonsteps.HTTPServerFromHTTPConfig(&b.config.HTTPConfig))
	}

	if b.config.VMName == "" {
		return nil, errors.New("\"vm_name\" is required")
	}

	if b.config.IPSW != "" {
		steps = append(steps, new(stepCreateVM))
	} else if b.config.VMBaseName != "" {
		// TODO: fix this, this logic probably doesn't work
		steps = append(steps, new(stepCloneVM))
	}

	communicatorConfigured := b.config.CommunicatorConfig.Type != "none"
	if len(b.config.BootCommand) > 0 || communicatorConfigured {
		steps = append(steps, new(stepRun))
	}

	if !b.config.RecoveryMode && communicatorConfigured {

		ui.Say("SSH")
		ui.Sayf("%+v", b.config.CommunicatorConfig.SSH)
		b.config.CommunicatorConfig.PauseBeforeConnect = time.Minute
		// b.config.CommunicatorConfig.SSHWaitTimeout = 10 * time.Minute
		// b.config.CommunicatorConfig.SSHTimeout = 5 * time.Minute

		steps = append(steps,
			// &communicator.StepSSHKeyGen{
			// 	CommConf:            &b.config.CommunicatorConfig,
			// 	SSHTemporaryKeyPair: b.config.CommunicatorConfig.SSH.SSHTemporaryKeyPair,
			// },
			&communicator.StepConnect{
				Config: &b.config.CommunicatorConfig,
				Host: func(state multistep.StateBag) (string, error) {
					return LumeMachineIP(ctx, b.config.VMName, ui, b.config.IpExtraArgs)
				},
				SSHConfig: b.config.CommunicatorConfig.SSHConfigFunc(),
			},
			&commonsteps.StepProvision{},
		)
	}

	// Setup the state bag and initial state for the steps
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("hook", hook)
	state.Put("ui", ui)

	// Run
	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("build was cancelled")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("build was halted")
	}

	artifact := &LumeVMArtifact{
		VMName:    b.config.VMName,
		StateData: map[string]interface{}{"generated_data": state.Get("generated_data")},
	}
	return artifact, nil
}
