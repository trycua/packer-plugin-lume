package lume

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/mitchellh/go-vnc"
)

var ErrFailedToDetectHostIP = errors.New("failed to detect host IP")

var vncRegexp = regexp.MustCompile(`local=(vnc://[a-zA-Z0-9\-]*:[a-zA-Z0-9\-]*@[0-9\.]*:[0-9]{1,5})`)

type stepRun struct{}

type bootCommandTemplateData struct {
	HTTPIP   string
	HTTPPort int
}

func (s *stepRun) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Starting the virtual machine...")
	var message string
	sleepDuration := time.Second * 1
	message = fmt.Sprintf("Waiting %v before starting"+
		"...", sleepDuration)
	ui.Say(message)
	time.Sleep(sleepDuration)

	// TODO: lume change
	runArgs := []string{"echo", "$PATH", "&&", lumeCommand, "run"}
	if config.Headless {
		runArgs = append(runArgs, "--no-display")
	}
	if config.RecoveryMode {
		runArgs = append(runArgs, "--recovery-mode=true")
	}
	if config.Rosetta != "" {
		runArgs = append(runArgs, fmt.Sprintf("--rosetta=%s", config.Rosetta))
	}
	if len(config.RunExtraArgs) > 0 {
		runArgs = append(runArgs, config.RunExtraArgs...)
	}
	runArgs = append(runArgs, config.VMName)

	ui.Say("Exec run")
	var stdoutChan <-chan *string
	var errChan <-chan error
	var ctxWithCancel context.Context
	var cancel context.CancelFunc
	ctxWithCancel, cancel = context.WithCancel(ctx)
	stdoutChan, errChan = LumeExec().
		WithContext(ctxWithCancel).
		WithPackerUI(ui).
		WithSkipLumePrepend(true).
		WithSleep(1).
		WithArgs(runArgs...).
		DoChanPty()

	if len(config.BootCommand) > 0 && !config.DisableVNC {
		if !typeBootCommandOverVNC(ctx, state, config, ui, stdoutChan) {
			return multistep.ActionHalt
		}
	} else {
		// Consume stdout lines in a goroutine or via select.
		go func() {
			for line := range stdoutChan {
				if line != nil {
					// process stdout line
					ui.Message(*line)
				}
			}
		}()
	}

	sleepDuration = 1 * time.Minute
	ui.Sayf("Sleeping for %v", sleepDuration)
	time.Sleep(sleepDuration)

	for {
		select {
		case err, ok := <-errChan:
			if ok {
				ui.Errorf("[Error] While running vm run: %v", err)
				return multistep.ActionHalt
			}
		default:
			ui.Sayf("Run step completed. Moving to next...")
			// // TODO: remove the below line
			// ui.Sayf("Sleeping for 1hr. Check for ssh enablement")
			// time.Sleep(time.Hour)
			state.Put("run/cancel-func", cancel)
			state.Put("run/stdout-chan", stdoutChan)
			state.Put("run/error-chan", errChan)
			return multistep.ActionContinue
		}
	}

}

type uiWriter struct {
	ui packersdk.Ui
}

func (u uiWriter) Write(p []byte) (n int, err error) {
	u.ui.Error(strings.TrimSpace(string(p)))
	return len(p), nil
}

// Cleanup stops the VM.
func (s *stepRun) Cleanup(state multistep.StateBag) {
	// config := state.Get("config").(*Config)
	// ui := state.Get("ui").(packersdk.Ui)
	// cancel := state.Get("run/cancel-func").(context.CancelFunc)
	// if cancel == nil {
	// 	return // Nothing to shut down
	// }

	// communicator := state.Get("communicator")
	// if communicator != nil {
	// 	ui.Say("Gracefully shutting down the VM...")
	// 	shutdownCmd := packersdk.RemoteCmd{
	// 		Command: fmt.Sprintf("echo %s | sudo -S -p '' shutdown -h now", config.CommunicatorConfig.Password()),
	// 	}

	// 	err := shutdownCmd.RunWithUi(context.Background(), communicator.(packersdk.Communicator), ui)
	// 	if err != nil {
	// 		ui.Say("Failed to gracefully shutdown VM...")
	// 		ui.Error(err.Error())
	// 	}
	// } else {
	// 	ui.Say("Shutting down the VM...")
	// 	cancel()
	// }

	// ui.Say("Waiting for the process to exit...")
	// // TODO: make this an actual wait
	// time.Sleep(time.Second * 5)
}

func typeBootCommandOverVNC(
	ctx context.Context,
	state multistep.StateBag,
	config *Config,
	ui packersdk.Ui,
	stdoutChan <-chan *string,
) bool {
	ui.Say("Typing boot commands over VNC...")

	if config.HTTPDir != "" || len(config.HTTPContent) != 0 {
		ui.Say("Detecting host IP...")

		hostIP, err := detectHostIP(ctx, ui, config)
		if err != nil {
			err := fmt.Errorf("Failed to detect the host IP address: %v", err)
			state.Put("error", err)
			ui.Error(err.Error())

			return false
		}

		ui.Say(fmt.Sprintf("Host IP is assumed to be %s", hostIP))
		state.Put("http_ip", hostIP)

		// Should be already filled by the Packer's commonsteps.StepHTTPServer
		httpPort := state.Get("http_port").(int)

		config.ctx.Data = &bootCommandTemplateData{
			HTTPIP:   hostIP,
			HTTPPort: httpPort,
		}
	}

	ui.Say("Waiting for the VNC server credentials from Lume...")

	vncTimeoutDuration := 30 * time.Second
	vncCtx, cancel := context.WithTimeout(ctx, vncTimeoutDuration)
	defer cancel()

	// var vncPassword string
	// var vncHost string
	// var vncPort string
	var vncAddress string

	// Consume stdout lines in a goroutine or via select.
	go func() {
		for line := range stdoutChan {
			if line != nil {
				// process stdout line
				ui.Message(*line)
				matches := vncRegexp.FindStringSubmatch(*line)
				if vncAddress == "" && (len(matches) == 1+vncRegexp.NumSubexp()) {
					vncAddress = matches[1]
				}
			}
		}
	}()

	for {
		if vncAddress != "" {
			ui.Sayf("VNC address found to be '%v'.", vncAddress)
			break
		}

		select {
		case <-vncCtx.Done():
			ui.Errorf("Unable to find vnc address in the duration %v. Exiting...", vncTimeoutDuration)
			return false
		case <-time.After(time.Second):
			// continue
		}
	}

	ui.Say("Retrieved VNC credentials, connecting...")

	// Parse the URL to extract host, port, and password
	parsedURL, err := url.Parse(vncAddress)
	if err != nil {
		ui.Errorf("Failed to parse VNC address: %v", err)
		return false
	}

	vncHost := parsedURL.Hostname()             // should be "127.0.0.1"
	vncPort := parsedURL.Port()                 // should be "49185" (example)
	vncPassword, _ := parsedURL.User.Password() // extracts "beach-willow-purple-bold" (example)

	ui.Message(fmt.Sprintf(
		"If you want to view the screen of the VM, connect via VNC with the password \"%s\" to\n"+
			"vnc://%s:%s", vncPassword, vncHost, vncPort))

	var message string
	sleepDuration := time.Second * 1
	message = fmt.Sprintf("Waiting %v to let the run process complete "+
		"to finish correctly...", sleepDuration)
	ui.Say(message)
	time.Sleep(sleepDuration)

	dialer := net.Dialer{}
	netConn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", vncHost, vncPort))
	if err != nil {
		err := fmt.Errorf("Failed to connect to the Lume's VNC server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}
	defer netConn.Close()

	vncClient, err := vnc.Client(netConn, &vnc.ClientConfig{
		Auth: []vnc.ClientAuth{
			&vnc.PasswordAuth{Password: vncPassword},
		},
	})
	if err != nil {
		err := fmt.Errorf("Failed to connect to the Lume's VNC server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}
	defer vncClient.Close()

	ui.Say("Connected to the VNC!")

	if config.VNCConfig.BootWait > 0 {
		message := fmt.Sprintf("Waiting %v after the VM has booted...", config.VNCConfig.BootWait)
		ui.Say(message)
		time.Sleep(config.VNCConfig.BootWait)
	}

	message = fmt.Sprintf("Typing commands with key interval %v...", config.BootKeyInterval)
	ui.Say(message)

	vncDriver := bootcommand.NewVNCDriver(vncClient, config.BootKeyInterval)

	command, err := interpolate.Render(config.VNCConfig.FlatBootCommand(), &config.ctx)
	if err != nil {
		err := fmt.Errorf("Failed to render the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	seq, err := bootcommand.GenerateExpressionSequence(command)
	if err != nil {
		err := fmt.Errorf("Failed to parse the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	if err := seq.Do(ctx, vncDriver); err != nil {
		err := fmt.Errorf("Failed to run the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	ui.Say("Done typing commands!")

	return true
}

func detectHostIP(ctx context.Context, ui packer.Ui, config *Config) (string, error) {
	if config.HTTPAddress != "0.0.0.0" {
		return config.HTTPAddress, nil
	}

	vmIPRaw, err := LumeMachineIP(ctx, config.VMName, ui, config.IpExtraArgs)
	if err != nil {
		return "", fmt.Errorf("%w: while running \"lume ip fetch\": %v",
			ErrFailedToDetectHostIP, err)
	}
	vmIP := net.ParseIP(vmIPRaw)

	// Find the interface that has this IP
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("%w: while retrieving interfaces: %v",
			ErrFailedToDetectHostIP, err)
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", fmt.Errorf("%w: while retrieving interface addresses: %v",
				ErrFailedToDetectHostIP, err)
		}

		for _, addr := range addrs {
			_, net, err := net.ParseCIDR(addr.String())
			if err != nil {
				return "", fmt.Errorf("%w: while parsing interface CIDR: %v",
					ErrFailedToDetectHostIP, err)
			}

			if net.Contains(vmIP) {
				gatewayIP, err := cidr.Host(net, 1)
				if err != nil {
					return "", fmt.Errorf("%w: while calculating gateway IP: %v",
						ErrFailedToDetectHostIP, err)
				}

				return gatewayIP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("%w: no suitable interface found", ErrFailedToDetectHostIP)
}
