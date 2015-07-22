package main

import (
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/codegangsta/cli"
	"github.com/zvelo/envetcd"
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
	exitCodeEnvEtcdError
)

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func run(c *cli.Context) {
	args := c.Args()
	if len(config.WriteEnv) > 0 && len(args) > 0 {
		log.Warnln("command not executed when --write-env is used")
	} else if len(config.WriteEnv) == 0 && len(args) < 1 {
		err := fmt.Errorf("cli: missing command")
		cli.ShowAppHelp(c)
		log.Printf("[ERR] %s", err.Error())
		os.Exit(exitCodeParseFlagsError)
	}

	exitCode, err := start(args[0:]...)
	if err != nil {
		log.Errorf("%s", err.Error())
	}

	os.Exit(exitCode)
}

func writeEnvFile() (int, error) {
	f, err := os.Create(config.WriteEnv)
	if err != nil {
		return exitCodeError, nil
	}
	defer f.Close()

	keyPairs, err := envetcd.GetKeyPairs(config.EnvEtcd)
	if err != nil {
		return exitCodeEnvEtcdError, err
	}

	for key, value := range keyPairs {
		value = strings.Replace(value, "\"", "\\\"", -1)
		fmt.Fprintf(f, "%s=\"%s\"\n", key, value)
	}

	return exitCodeOK, nil
}

func start(command ...string) (int, error) {
	log.Debugf("(cli) getting data from etcd")

	if len(config.WriteEnv) > 0 {
		return writeEnvFile()
	}

	log.Debugf("(cli) creating Runner")
	runner, err := newRunner(command...)
	if err != nil {
		return exitCodeParseFlagsError, err
	}

	runner.data, err = envetcd.GetKeyPairs(config.EnvEtcd)
	if err != nil {
		return exitCodeEnvEtcdError, err
	}

	log.Infof("(cli) invoking Runner")
	if err := runner.run(); err != nil {
		return exitCodeRunnerError, err
	}

	for {
		select {
		case exitCode := <-runner.exitCh:
			log.Infof("(cli) subprocess exited")

			if exitCode == exitCodeOK {
				return exitCodeOK, nil
			}

			err := fmt.Errorf("unexpected exit from subprocess (%d)", exitCode)
			return exitCode, err
		}
	}
}
