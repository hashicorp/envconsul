package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/hashicorp/logutils"
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

func initLogger(c *cli.Context) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERR"},
		MinLevel: logutils.LogLevel(c.GlobalString("log-level")),
		Writer:   os.Stderr,
	}

	log.SetOutput(filter)
	log.Printf("[INFO] log level set to %s", filter.MinLevel)
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func run(c *cli.Context) {
	initLogger(c)

	args := c.Args()
	if len(args) < 1 {
		err := fmt.Errorf("cli: missing command")
		cli.ShowAppHelp(c)
		handleError(err, exitCodeParseFlagsError)
	}

	command := args[0:]

	log.Printf("[DEBUG] (cli) creating Runner")
	runner := newRunner(c, command)

	log.Printf("[DEBUG] (cli) creating etcd API client")
	etcdConfig := newEtcdConfig(c)
	client, err := getClient(etcdConfig)
	if err != nil {
		handleError(err, exitCodeEtcdError)
	}

	log.Printf("[DEBUG] (cli) getting data from etcd")
	runner.data = getKeyPairs(etcdConfig, client)

	log.Printf("[INFO] (cli) invoking Runner")
	if err := runner.run(); err != nil {
		handleError(err, exitCodeRunnerError)
	}

	for {
		select {
		case exitCode := <-runner.exitCh:
			log.Printf("[INFO] (cli) subprocess exited")

			if exitCode == exitCodeOK {
				os.Exit(exitCodeOK)
			}

			err := fmt.Errorf("unexpected exit from subprocess (%d)", exitCode)
			handleError(err, exitCode)
		}
	}
}

// handleError outputs the given error's Error() and returns the given exit status.
func handleError(err error, status int) {
	log.Printf("[ERR] %s", err.Error())
	os.Exit(status)
}
