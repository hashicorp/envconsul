package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/coreos/go-etcd/etcd"
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
	exitCodeEtcdError
)

func massagePeers() error {
	for i, ep := range config.Peers {
		u, err := url.Parse(ep)
		if err != nil {
			return err
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		config.Peers[i] = u.String()
	}
	return nil
}

func genConfig(c *cli.Context) {
	config.Peers = c.GlobalStringSlice("peers")
	config.Hostname = c.GlobalString("hostname")
	config.System = c.GlobalString("system")
	config.Service = c.GlobalString("service")
	config.Prefix = c.GlobalString("prefix")
	config.Output = c.String("output")
	config.WriteEnv = c.GlobalString("write-env")
	config.Sync = !c.GlobalBool("no-sync")
	config.CleanEnv = c.GlobalBool("clean-env")
	config.Sanitize = !c.GlobalBool("no-sanitize")
	config.Upcase = !c.GlobalBool("no-upcase")
	config.TLS.CAFile = c.GlobalString("ca-file")
	config.TLS.CertFile = c.GlobalString("cert-file")
	config.TLS.KeyFile = c.GlobalString("key-file")

	if err := massagePeers(); err != nil {
		log.Fatal(err)
	}
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func run(c *cli.Context) {
	util.InitLogger(c.GlobalString("log-level"))
	genConfig(c)

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

func writeEnvFile(client *etcd.Client) (int, error) {
	f, err := os.Create(config.WriteEnv)
	if err != nil {
		return exitCodeError, nil
	}
	defer f.Close()

	for key, value := range getKeyPairs(client) {
		if config.Sanitize {
			key = invalidRegexp.ReplaceAllString(key, "_")
		}

		if config.Upcase {
			key = strings.ToUpper(key)
		}

		value = strings.Replace(value, "\"", "\\\"", -1)

		fmt.Fprintf(f, "%s=\"%s\"\n", key, value)
	}

	return exitCodeOK, nil
}

func start(command ...string) (int, error) {
	log.Printf("[DEBUG] (cli) creating etcd API client")
	client, err := getClient()
	if err != nil {
		return exitCodeEtcdError, err
	}

	log.Printf("[DEBUG] (cli) getting data from etcd")

	if len(config.WriteEnv) > 0 {
		return writeEnvFile(client)
	}

	log.Printf("[DEBUG] (cli) creating Runner")
	runner, err := newRunner(command...)
	if err != nil {
		return exitCodeParseFlagsError, err
	}

	runner.data = getKeyPairs(client)

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
