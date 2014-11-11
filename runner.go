package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	"github.com/zvelo/envetcd/util"
)

// Regexp for invalid characters in keys
var InvalidRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)

type runner struct {
	// Prefix is the KeyPrefixDependency associated with this runner.
	Prefix *util.KeyPrefixDependency

	// Command is the slice of the command string and any arguments.
	Command []string

	// ExitCh is a channel for parent processes to read exit status values from
	// the child processes.
	ExitCh chan int

	// config is the internal config struct.
	config *Config

	// data is the latest representation of the data from etcd.
	data []*util.KeyPair

	// env is the last compiled environment.
	env map[string]string

	// cmd is the last known instance of the running command for this runner.
	cmd *exec.Cmd

	// outStream and errStream are the io.Writer streams where the runner will
	// write information. These default to stdout and stderr, but can be
	// changed for testing purposes
	outStream, errStream io.Writer
}

func newRunner(s string, config *Config, command []string) (*runner, error) {
	if s == "" {
		return nil, fmt.Errorf("runner: missing prefix")
	}

	if config == nil {
		return nil, fmt.Errorf("runner: missing config")
	}

	if len(command) == 0 {
		return nil, fmt.Errorf("runner: missing command")
	}

	prefix, err := util.ParseKeyPrefixDependency(s)
	if err != nil {
		return nil, err
	}

	run := &runner{
		Prefix:    prefix,
		Command:   command,
		config:    config,
		outStream: os.Stdout,
		errStream: os.Stderr,
	}

	return run, nil
}

// Dependencies returns the list of dependencies for this runner. At this time,
// this function will always return a slice with exactly one element.
func (r *runner) Dependencies() []util.Dependency {
	return []util.Dependency{r.Prefix}
}

// Receive accepts data from etcd and maps that data to the prefix.
func (r *runner) Receive(data interface{}) {
	r.data = data.([]*util.KeyPair)
}

// Wait for the child process to finish (if one exists).
func (r *runner) Wait() int {
	return <-r.ExitCh
}

// Run executes and manages the child process with the correct environment. The
// current enviornment is also copied into the child process environment.
func (r *runner) Run() error {
	env := make(map[string]string)
	for _, pair := range r.data {
		key := pair.Key

		if r.config.Sanitize {
			key = InvalidRegexp.ReplaceAllString(key, "_")
		}

		if r.config.Upcase {
			key = strings.ToUpper(key)
		}

		env[key] = string(pair.Value)
	}

	// If the resulting map is the same, do not do anything
	if reflect.DeepEqual(r.env, env) {
		return nil
	}

	// Update the environment
	r.env = env

	// Restart the current process if it exists
	if r.cmd != nil && r.cmd.Process != nil {
		r.restartProcess()
	}

	// Create a new environment
	processEnv := os.Environ()
	cmdEnv := make([]string, len(processEnv), len(r.env)+len(processEnv))
	copy(cmdEnv, processEnv)
	for k, v := range r.env {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(r.Command[0], r.Command[1:]...)
	cmd.Stdout = r.outStream
	cmd.Stderr = r.errStream
	cmd.Env = cmdEnv
	err := cmd.Start()
	if err != nil {
		return err
	}

	r.cmd = cmd

	// Create a new exitCh so that previously invoked commands
	// (if any) don't cause us to exit, and start a goroutine
	// to wait for that process to end.
	r.ExitCh = make(chan int, 1)
	go func(cmd *exec.Cmd, exitCh chan<- int) {
		err := cmd.Wait()
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
	}(cmd, r.ExitCh)

	return nil
}

// Restart the current process in the runner by sending a SIGTERM. It is
// assumed that the process is set on the runner!
func (r *runner) restartProcess() {
	// Kill the process
	exited := false

	if err := r.cmd.Process.Signal(syscall.SIGTERM); err == nil {
		// Wait a few seconds for it to exit
		killCh := make(chan struct{})
		go func() {
			defer close(killCh)
			r.cmd.Process.Wait()
		}()

		select {
		case <-killCh:
			exited = true
		}
	}

	// If we still haven't exited from a SIGKILL
	if !exited {
		r.cmd.Process.Kill()
	}

	r.cmd = nil
}
