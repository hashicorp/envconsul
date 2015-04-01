package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/zvelo/envetcd"
)

type runner struct {
	// Command is the slice of the command string and any arguments.
	command []string

	// exitCh is a channel for parent processes to read exit status values from
	// the child processes.
	exitCh chan int

	// data is the latest representation of the data from etcd.
	data envetcd.KeyPairs

	outFile   *os.File
	outStream io.Writer
}

func newRunner(command ...string) (*runner, error) {
	run := &runner{
		command:   command,
		outStream: os.Stdout,
	}

	if len(config.Output) > 0 {
		outFile, err := os.Create(config.Output)
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
	// Create a new environment
	processEnv := os.Environ()
	if config.CleanEnv {
		processEnv = []string{}
	}

	for k, v := range r.data {
		processEnv = append(processEnv, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(r.command[0], r.command[1:]...)
	cmd.Stdout = r.outStream
	cmd.Stderr = os.Stderr
	cmd.Env = processEnv
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
