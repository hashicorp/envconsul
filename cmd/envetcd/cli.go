package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

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

	const ext = ".tmpl"
	for _, tmpl := range config.Templates {
		if filepath.Ext(tmpl) != ext {
			log.Printf("[WARN] template file does not end with '.tmpl' (%s)", tmpl)
			continue
		}

		outFileName := tmpl[0 : len(tmpl)-len(ext)]
		outFile, err := os.Create(outFileName)
		if err != nil {
			log.Printf("[WARN] error creating file (%s): %s", outFileName, err)
			continue
		}

		tpl, err := template.ParseFiles(tmpl)
		if err != nil {
			log.Printf("[WARN] error parsing template (%s): %s", tmpl, err)
			continue
		}

		log.Printf("[INFO] updating template %s", tmpl)
		if err := tpl.Execute(outFile, runner.data); err != nil {
			log.Printf("[WARN] error writing template file (%s): %s", outFileName, err)
		}
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
