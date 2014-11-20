package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
)

// Regexp for invalid characters in keys
var invalidRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)

type runner struct {
	// Command is the slice of the command string and any arguments.
	command []string

	// exitCh is a channel for parent processes to read exit status values from
	// the child processes.
	exitCh chan int

	sanitize bool
	upcase   bool
	cleanEnv bool

	// data is the latest representation of the data from etcd.
	data KeyPairs

	outFile   *os.File
	outStream io.Writer
}

func newRunner(c *cli.Context, command []string) (*runner, error) {
	run := &runner{
		command:   command,
		sanitize:  !c.Bool("no-sanitize"),
		upcase:    !c.Bool("no-upcase"),
		cleanEnv:  c.Bool("clean-env"),
		outStream: os.Stdout,
	}

	if output := c.String("output"); len(output) > 0 {
		outFile, err := os.Create(output)
		if err != nil {
			return nil, err
		}
		run.outFile = outFile
		run.outStream = bufio.NewWriter(outFile)
	}

	return run, nil
}

// Run executes and manages the child process with the correct environment. The
// current enviornment is also copied into the child process environment.
func (r *runner) run() error {
	env := make(map[string]string)
	for key, value := range r.data {
		if r.sanitize {
			key = invalidRegexp.ReplaceAllString(key, "_")
		}

		if r.upcase {
			key = strings.ToUpper(key)
		}

		env[key] = value
	}

	// Create a new environment
	processEnv := os.Environ()
	if r.cleanEnv {
		processEnv = []string{}
	}

	cmdEnv := make([]string, len(processEnv), len(env)+len(processEnv))
	copy(cmdEnv, processEnv)
	for k, v := range env {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(r.command[0], r.command[1:]...)
	cmd.Stdout = r.outStream
	cmd.Stderr = os.Stderr
	cmd.Env = cmdEnv
	err := cmd.Start()
	if err != nil {
		return err
	}

	// Create a new exitCh so that previously invoked commands
	// (if any) don't cause us to exit, and start a goroutine
	// to wait for that process to end.
	r.exitCh = make(chan int, 1)
	go func(cmd *exec.Cmd, exitCh chan<- int, outFile *os.File) {
		err := cmd.Wait()

		if outFile != nil {
			writer, ok := cmd.Stdout.(*bufio.Writer)
			if ok {
				writer.Flush()

			}

			if err := outFile.Close(); err != nil {
				exitCh <- exitCodeError
				return
			}
		}

		if err == nil {
			exitCh <- exitCodeOK
			return
		}

		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitCh <- status.ExitStatus()
				return
			}
		}

		exitCh <- exitCodeError
	}(cmd, r.exitCh, r.outFile)

	return nil
}
