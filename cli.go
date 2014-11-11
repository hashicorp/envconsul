package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
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
	exitCodeEtcdError
)

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func run(c *cli.Context) {
	args := c.Args()
	if len(args) < 2 {
		err := fmt.Errorf("cli: missing required arguments prefix and command")
		cli.ShowAppHelp(c)
		handleError(err, exitCodeParseFlagsError)
	}

	prefix, command := args[0], args[1:]

	runner := newRunner(c, command)

	client, err := getClient(c)
	if err != nil {
		handleError(err, exitCodeEtcdError)
	}

	if runner.data, err = getKeyPairs(client, prefix); err != nil {
		handleError(err, exitCodeEtcdError)
	}

	if err := runner.run(); err != nil {
		handleError(err, exitCodeRunnerError)
	}

	for {
		select {
		case exitCode := <-runner.exitCh:
			os.Exit(exitCode)
		}
	}
}

// handleError outputs the given error's Error() and returns the given exit status.
func handleError(err error, status int) {
	log.Printf("[ERR] %s", err.Error())
	os.Exit(status)
}
