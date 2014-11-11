package main

import (
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/zvelo/envetcd/util"
)

// Exit codes are int values that represent an exit code for a particular error.
// Sub-systems may check this unique error to determine the cause of an error
// without parsing the output or help text.
const (
	exitCodeOK int = 0

	// Errors start at 10
	exitCodeError = 10 + iota
	exitCodeParseFlagsError
	exitCodeRunnerError
	exitCodeWatcherError
)

type cli struct {
	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func (c *cli) Run(args []string) int {
	var ver bool
	var config = new(Config)

	// Parse the flags and options
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(c.errStream)
	flags.Usage = func() {
		fmt.Fprintf(c.errStream, usage, name)
	}
	flags.StringVar(&config.Etcd, "etcd", "", "address of the etcd instance")
	flags.StringVar(&config.CAFile, "ca-file", "", "certificate authority file")
	flags.StringVar(&config.CertFile, "cert-file", "", "tls client certificate file")
	flags.StringVar(&config.KeyFile, "key-file", "", "tls client key file")
	flags.BoolVar(&config.Sanitize, "sanitize", true, "remove bad characters from values")
	flags.BoolVar(&config.Upcase, "upcase", true, "convert all environment keys to uppercase")
	flags.BoolVar(&ver, "version", false, "display the version")

	// If there was a parser error, stop
	if err := flags.Parse(args[1:]); err != nil {
		return c.handleError(err, exitCodeParseFlagsError)
	}

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if ver {
		fmt.Fprintf(c.errStream, "%s v%s\n", name, version)
		return exitCodeOK
	}

	args = flags.Args()
	if len(args) < 2 {
		err := fmt.Errorf("cli: missing required arguments prefix and command")
		return c.handleError(err, exitCodeParseFlagsError)
	}

	prefix, command := args[0], args[1:]

	runner, err := newRunner(prefix, config, command)
	if err != nil {
		return c.handleError(err, exitCodeRunnerError)
	}

	client, err := c.getClient(config)
	if err != nil {
		return c.handleError(err, exitCodeError)
	}

	watcher, err := util.NewWatcher(client, runner.Dependencies())
	if err != nil {
		return c.handleError(err, exitCodeWatcherError)
	}

	go watcher.Watch()

	for {
		select {
		case data := <-watcher.DataCh:
			// Tell the Runner about the data
			runner.Receive(data.Data)

			if err := runner.Run(); err != nil {
				return c.handleError(err, exitCodeRunnerError)
			}
		case err := <-watcher.ErrCh:
			log.Printf("[INFO] (cli) watcher got error")
			return c.handleError(err, exitCodeError)
		case <-watcher.FinishCh:
			return runner.Wait()
		case exitCode := <-runner.ExitCh:
			log.Printf("[INFO] (cli) subprocess exited")

			if exitCode == exitCodeOK {
				return exitCodeOK
			}

			err := fmt.Errorf("unexpected exit from subprocess (%d)", exitCode)
			return c.handleError(err, exitCode)
		}
	}
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (c *cli) handleError(err error, status int) int {
	log.Printf("[ERR] %s", err.Error())
	return status
}

const usage = `Usage: %s [options]

  Sets environment variables from etcd

Options:

  -etcd=<address>      Sets the address of the etcd instance (default: "127.0.0.1:4001")
  -sanitize            Replace invalid characters in keys to underscores
  -upcase              Convert all environment variable keys to uppercase
  -version             Print the version of this daemon
`
