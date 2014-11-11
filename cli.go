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
	ExitCodeOK int = 0

	// Errors start at 10
	ExitCodeError = 10 + iota
	ExitCodeParseFlagsError
	ExitCodeRunnerError
	ExitCodeWatcherError
)

type CLI struct {
	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func (cli *CLI) Run(args []string) int {
	var version bool
	var config = new(Config)

	// Parse the flags and options
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprintf(cli.errStream, usage, Name)
	}
	flags.StringVar(&config.Etcd, "etcd", "", "address of the etcd instance")
	flags.StringVar(&config.CAFile, "ca-file", "", "certificate authority file")
	flags.StringVar(&config.CertFile, "cert-file", "", "tls client certificate file")
	flags.StringVar(&config.KeyFile, "key-file", "", "tls client key file")
	flags.BoolVar(&config.Sanitize, "sanitize", true, "remove bad characters from values")
	flags.BoolVar(&config.Upcase, "upcase", true, "convert all environment keys to uppercase")
	flags.BoolVar(&version, "version", false, "display the version")

	// If there was a parser error, stop
	if err := flags.Parse(args[1:]); err != nil {
		return cli.handleError(err, ExitCodeParseFlagsError)
	}

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if version {
		fmt.Fprintf(cli.errStream, "%s v%s\n", Name, Version)
		return ExitCodeOK
	}

	args = flags.Args()
	if len(args) < 2 {
		err := fmt.Errorf("cli: missing required arguments prefix and command")
		return cli.handleError(err, ExitCodeParseFlagsError)
	}

	prefix, command := args[0], args[1:]

	runner, err := NewRunner(prefix, config, command)
	if err != nil {
		return cli.handleError(err, ExitCodeRunnerError)
	}

	client, err := cli.getClient(config)
	if err != nil {
		return cli.handleError(err, ExitCodeError)
	}

	watcher, err := util.NewWatcher(client, runner.Dependencies())
	if err != nil {
		return cli.handleError(err, ExitCodeWatcherError)
	}

	go watcher.Watch()

	for {
		select {
		case data := <-watcher.DataCh:
			// Tell the Runner about the data
			runner.Receive(data.Data)

			if err := runner.Run(); err != nil {
				return cli.handleError(err, ExitCodeRunnerError)
			}
		case err := <-watcher.ErrCh:
			log.Printf("[INFO] (cli) watcher got error")
			return cli.handleError(err, ExitCodeError)
		case <-watcher.FinishCh:
			return runner.Wait()
		case exitCode := <-runner.ExitCh:
			log.Printf("[INFO] (cli) subprocess exited")

			if exitCode == ExitCodeOK {
				return ExitCodeOK
			}

			err := fmt.Errorf("unexpected exit from subprocess (%d)", exitCode)
			return cli.handleError(err, exitCode)
		}
	}
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (cli *CLI) handleError(err error, status int) int {
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
