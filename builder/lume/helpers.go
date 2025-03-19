package lume

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/shell-local/localexec"
	"github.com/mitchellh/iochan"
)

// TODO: lume change
// const lumeCommand = "/Users/prashant/warpbuilds/lume/.build/arm64-apple-macosx/debug/lume"
const lumeCommand = "lume"

func PathInLumeHome(elem ...string) string {
	if home := os.Getenv("LUME_HOME"); home != "" {
		return path.Join(home, path.Join(elem...))
	}
	userHome, _ := os.UserHomeDir()
	return path.Join(userHome, ".lume", path.Join(elem...))
}

type execBuilder struct {
	ctx             context.Context
	args            []string
	sleepDuration   int64
	ui              packer.Ui
	skipLumePrepend bool
}

func LumeExec() *execBuilder {
	return &execBuilder{}
}

func (eb *execBuilder) WithSleep(durationSeconds int64) *execBuilder {
	eb.sleepDuration = durationSeconds
	return eb
}

func (eb *execBuilder) WithPackerUI(ui packer.Ui) *execBuilder {
	eb.ui = ui
	return eb
}

func (eb *execBuilder) WithContext(ctx context.Context) *execBuilder {
	eb.ctx = ctx
	return eb
}

func (eb *execBuilder) WithArgs(args ...string) *execBuilder {
	eb.args = append(eb.args, args...)
	return eb
}

func (eb *execBuilder) WithSkipLumePrepend(val bool) *execBuilder {
	eb.skipLumePrepend = val
	return eb
}

func (eb *execBuilder) Do() (string, error) {

	var cmd *exec.Cmd
	if eb.sleepDuration != 0 {
		var lumeCmdArgs []string
		if !eb.skipLumePrepend {
			lumeCmdArgs = append([]string{lumeCommand}, eb.args...)
		} else {
			lumeCmdArgs = eb.args
		}
		lumeCmdString := strings.Join(lumeCmdArgs, " ")
		sleepCmdString := fmt.Sprintf("sleep %v", eb.sleepDuration)
		completeCmdString := fmt.Sprintf("%v && %v", sleepCmdString, lumeCmdString)
		cmd = exec.CommandContext(eb.ctx, "/bin/bash", "-c", completeCmdString)
	} else {
		cmd = exec.CommandContext(eb.ctx, lumeCommand, eb.args...)
	}

	// Log the command being executed.
	args := make([]string, len(cmd.Args)-1)
	copy(args, cmd.Args[1:])
	eb.ui.Sayf("Executing: %s %q", cmd.Path, args)

	if eb.ui != nil {
		return "", localexec.RunAndStream(cmd, eb.ui, []string{})
	} else {
		stdout_r, stdout_w := io.Pipe()
		stderr_r, stderr_w := io.Pipe()
		defer stdout_w.Close()
		defer stderr_w.Close()

		args := make([]string, len(cmd.Args)-1)
		copy(args, cmd.Args[1:])

		log.Printf("Executing: %s %v", cmd.Path, args)
		cmd.Stdout = stdout_w
		cmd.Stderr = stderr_w
		if err := cmd.Start(); err != nil {
			return "", err
		}

		// Create the channels we'll use for data
		exitCh := make(chan int, 1)
		stdoutCh := iochan.DelimReader(stdout_r, '\n')
		stderrCh := iochan.DelimReader(stderr_r, '\n')

		// Start the goroutine to watch for the exit
		go func() {
			defer stdout_w.Close()
			defer stderr_w.Close()
			exitStatus := 0

			err := cmd.Wait()
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitStatus = 1

				// There is no process-independent way to get the REAL
				// exit status so we just try to go deeper.
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					exitStatus = status.ExitStatus()
				}
			}

			exitCh <- exitStatus
		}()

		// This waitgroup waits for the streaming to end
		var streamWg sync.WaitGroup
		streamWg.Add(2)

		streamFunc := func(ch <-chan string) {
			defer streamWg.Done()

			for data := range ch {
				eb.ui.Message(data)
			}
		}

		// Stream stderr/stdout
		go streamFunc(stderrCh)
		go streamFunc(stdoutCh)

		// Wait for the process to end and then wait for the streaming to end
		exitStatus := <-exitCh
		streamWg.Wait()

		if exitStatus != 0 {
			return "", fmt.Errorf("Bad exit status: %d", exitStatus)
		}

	}

	return "", nil
}

// DoChan executes a command and returns channels for streaming its stdout, stderr, and errors.
//
// This method constructs and executes a command using the execBuilder's context and arguments.
// If a sleep duration is specified, the command is prefixed with a sleep command to delay execution.
// The method returns three channels:
//   - stdoutCh: streams the command's stdout output line by line,
//   - stderrCh: streams the command's stderr output line by line, and
//   - errCh: asynchronously emits an error if the command fails to start or during execution.
//
// Usage:
//
//	stdoutCh, stderrCh, errCh := execBuilderInstance.DoChan()
//
//	// Consume stdout lines in a goroutine or via select.
//	go func() {
//	    for line := range stdoutCh {
//	        // process stdout line
//	    }
//	}()
//
//	// Consume stderr lines similarly.
//	go func() {
//	    for line := range stderrCh {
//	        // process stderr line
//	    }
//	}()
//
//	// Optionally handle errors from errCh.
//	if err, ok := <-errCh; ok {
//	    // handle error
//	}
func (eb *execBuilder) DoChan() (<-chan string, <-chan string, <-chan error) {
	var cmd *exec.Cmd
	if eb.sleepDuration != 0 {
		// Build the command string with a sleep prefix.
		var lumeCmdArgs []string
		if !eb.skipLumePrepend {
			lumeCmdArgs = append([]string{lumeCommand}, eb.args...)
		} else {
			lumeCmdArgs = eb.args
		}
		lumeCmdString := strings.Join(lumeCmdArgs, " ")
		sleepCmdString := fmt.Sprintf("sleep %v", eb.sleepDuration)
		completeCmdString := fmt.Sprintf("%v && %v", sleepCmdString, lumeCmdString)
		cmd = exec.CommandContext(eb.ctx, "/bin/bash", "-c", completeCmdString)
	} else {
		cmd = exec.CommandContext(eb.ctx, lumeCommand, eb.args...)
	}

	// Create pipes for stdout and stderr.
	stdout_r, stdout_w := io.Pipe()
	stderr_r, stderr_w := io.Pipe()

	// Assign command outputs.
	cmd.Stdout = stdout_w
	cmd.Stderr = stderr_w

	args := make([]string, len(cmd.Args)-1)
	copy(args, cmd.Args[1:])
	eb.ui.Sayf("Executing: %s %q", cmd.Path, args)
	// Create the error channel.
	errCh := make(chan error, 1)

	// Start the command.
	if err := cmd.Start(); err != nil {
		// On startup error, send the error and close writers.
		errCh <- err
		close(errCh)
		stdout_w.Close()
		stderr_w.Close()
		// Return channels so the caller can read the EOF if needed.
		return iochan.DelimReader(stdout_r, '\n'), iochan.DelimReader(stderr_r, '\n'), errCh
	}

	stdoutCh := iochan.DelimReader(stdout_r, '\n')
	stderrCh := iochan.DelimReader(stderr_r, '\n')

	// Wait for the command to finish and then signal via errCh.
	go func() {
		err := cmd.Wait()
		if err != nil {
			// Send any error (including non-zero exit status) into errCh.
			errCh <- err
		}
		// Close the pipe writers to signal EOF to the channel readers.
		stdout_w.Close()
		stderr_w.Close()
		close(errCh)
	}()

	return stdoutCh, stderrCh, errCh
}

// DoChanPty executes a command in a pseudo terminal (PTY) and returns channels for streaming its
// combined output (stdout and stderr) and errors.
//
// This method constructs and executes a command using the execBuilder's context and arguments.
// If a sleep duration is specified, the command is prefixed with a sleep command to delay execution.
// The command is executed in a PTY, which causes the output to be unbuffered and merged (stdout/stderr).
//
// It returns two channels:
//   - outCh: streams the command's combined output character by character,
//   - errCh: asynchronously emits an error if the command fails to start or during execution.
//
// Usage:
//
//	outCh, errCh := execBuilderInstance.DoChanPty()
//
//	// Consume output characters (which include both stdout and stderr) in a goroutine or via select.
//	go func() {
//	     for char := range outCh {
//	         // process output character
//	     }
//	}()
//
//	// Optionally handle errors from errCh.
//	if err, ok := <-errCh; ok {
//	     // handle error
//	}
func (eb *execBuilder) DoChanPty() (<-chan *string, <-chan error) {
	var cmd *exec.Cmd
	if eb.sleepDuration != 0 {
		var lumeCmdArgs []string
		if !eb.skipLumePrepend {
			lumeCmdArgs = append([]string{lumeCommand}, eb.args...)
		} else {
			lumeCmdArgs = eb.args
		}
		lumeCmdString := strings.Join(lumeCmdArgs, " ")
		sleepCmdString := fmt.Sprintf("sleep %v", eb.sleepDuration)
		completeCmdString := fmt.Sprintf("%v && %v", sleepCmdString, lumeCmdString)
		cmd = exec.CommandContext(eb.ctx, "/bin/bash", "-c", completeCmdString)
	} else {
		cmd = exec.CommandContext(eb.ctx, lumeCommand, eb.args...)
	}

	// Log the command being executed.
	args := make([]string, len(cmd.Args)-1)
	copy(args, cmd.Args[1:])
	eb.ui.Sayf("Executing: %s %q", cmd.Path, args)

	// Start the command in a pseudo terminal.
	ptyFile, err := pty.Start(cmd)
	errCh := make(chan error, 1)
	if err != nil {
		eb.ui.Errorf("Failed to start command in pseudo terminal. Error: %v", errCh)
		errCh <- err
		close(errCh)
		outCh := make(chan *string)
		close(outCh)
		return outCh, errCh
	}

	// Create a channel that emits each character immediately.
	outCh := readLines(ptyFile)

	// Wait for the command to finish and then signal via errCh.
	go func() {
		err := cmd.Wait()
		if err != nil {
			// If the process exited with a non-zero status, try to extract it.
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					eb.ui.Sayf("Process exited with status: %d", status.ExitStatus())
				}
			}
			errCh <- err
		}
		eb.ui.Sayf("Closing error chan. Process is complete.")
		close(errCh)
		ptyFile.Close()
	}()

	return outCh, errCh
}

// readChars reads from the provided io.Reader and sends each character (as a string)
// immediately on the returned channel.
func readChars(r io.Reader) <-chan string {
	ch := make(chan string)
	go func() {
		br := bufio.NewReader(r)
		for {
			rn, _, err := br.ReadRune()
			if err != nil {
				close(ch)
				break
			}
			ch <- string(rn)
		}
	}()
	return ch
}

// readLines reads from the provided io.Reader and sends each line (ending with '\n')
// on the returned channel.
func readLines(r io.Reader) <-chan *string {
	ch := make(chan *string, 1)
	go func() {
		br := bufio.NewReader(r)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				// If there's a partial line, send it before closing.
				if len(line) > 0 {
					ch <- &line
				}
				ch <- nil
				close(ch)
				return
			}
			ch <- &line
		}
	}()
	return ch
}
