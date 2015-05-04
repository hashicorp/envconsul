package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/zvelo/envetcd"
	"github.com/zvelo/zvelo-services/util"
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

func setup(c *cli.Context) error {
	util.InitLogger(c.GlobalString("log-level"))

	config = configT{
		EnvEtcd: &envetcd.Config{
			Etcd:              util.NewEtcdConfig(c),
			Hostname:          c.GlobalString("hostname"),
			System:            c.GlobalString("system"),
			Service:           c.GlobalString("service"),
			Prefix:            c.GlobalString("prefix"),
			Sanitize:          !c.GlobalBool("no-sanitize"),
			Upcase:            !c.GlobalBool("no-upcase"),
			UseDefaultGateway: c.GlobalBool("use-default-gateway"),
		},
		Output:   c.String("output"),
		WriteEnv: c.GlobalString("write-env"),
		CleanEnv: c.GlobalBool("clean-env"),
	}

	return nil
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func run(c *cli.Context) {
	args := c.Args()
	if len(config.WriteEnv) > 0 && len(args) > 0 {
		log.Println("[WARN] command not executed when --write-env is used")
	} else if len(config.WriteEnv) == 0 && len(args) < 1 {
		err := fmt.Errorf("cli: missing command")
		cli.ShowAppHelp(c)
		log.Printf("[ERR] %s", err.Error())
		os.Exit(exitCodeParseFlagsError)
	}

	exitCode, err := start(args[0:]...)
	if err != nil {
		log.Printf("[ERR] %s", err.Error())
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
	log.Printf("[DEBUG] (cli) getting data from etcd")

	if len(config.WriteEnv) > 0 {
		return writeEnvFile()
	}

	log.Printf("[DEBUG] (cli) creating Runner")
	runner, err := newRunner(command...)
	if err != nil {
		return exitCodeParseFlagsError, err
	}

	runner.data, err = envetcd.GetKeyPairs(config.EnvEtcd)
	if err != nil {
		return exitCodeEnvEtcdError, err
	}

	log.Printf("[INFO] (cli) invoking Runner")
	if err := runner.run(); err != nil {
		return exitCodeRunnerError, err
	}

	for {
		select {
		case exitCode := <-runner.exitCh:
			log.Printf("[INFO] (cli) subprocess exited")

			if exitCode == exitCodeOK {
				return exitCodeOK, nil
			}

			err := fmt.Errorf("unexpected exit from subprocess (%d)", exitCode)
			return exitCode, err
		}
	}
}
