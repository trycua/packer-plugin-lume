package lume

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

var localIPv4Regex = regexp.MustCompile(`\b((?:[0-9]{1,3}\.){3}[0-9]{1,3})\b`)

func LumeMachineIP(ctx context.Context, vmName string, ui packer.Ui, ipExtraArgs []string) (string, error) {
	ipArgs := []string{fmt.Sprintf("lume get %s | tail -n 1 | awk '{ print $8 }'", vmName)}
	// ipArgs := []string{fmt.Sprintf("arp -a | grep -i $(cat /Users/administrator/.lume/%s/config.json | jq -r '.macAddress')", vmName)}
	outChan, errChan := LumeExec().
		WithContext(ctx).
		WithSleep(1).
		WithPackerUI(ui).
		WithSkipLumePrepend(true).
		WithArgs(ipArgs...).
		DoChanPty()

	var ip string

	for {
		select {
		case err, ok := <-errChan:
			ui.Say("Executing error chan logic")
			if !ok {
				// errChan is closed; set it to nil so this case is never selected again.
				ui.Sayf("Found error chan to be closed.")
				errChan = nil
				continue
			}
			ui.Errorf("[Error] While fetching vm IP: %v", err)
			return "", err
		case line, ok := <-outChan:
			ui.Say("Executing out chan logic")
			if !ok || line == nil {
				// outChan is closed; set it to nil.
				ui.Sayf("Found out chan to be closed.")
				outChan = nil
				continue
			}
			// Process the line from outChan.
			ui.Message(*line)
			matches := localIPv4Regex.FindStringSubmatch(*line)
			if len(matches) > 1 {
				ui.Say("Found IP. Output:")
				ui.Say(matches[1])
				ip = matches[1]
				outChan = nil
			}
		}
		// Break out of the loop if ip is found.
		if ip != "" {
			break
		}
	}

	ui.Say("Returing ip")
	return ip, nil
}
