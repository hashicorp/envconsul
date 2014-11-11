package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
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

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:  "run",
		Usage: "run a command with environment variables set from etcd",
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:   "peers, C",
				EnvVar: "ENVETCD_PEERS",
				Value:  &cli.StringSlice{"127.0.0.1:4001"},
				Usage:  "a comma-delimited list of machine addresses in the cluster (default: \"127.0.0.1:4001\")",
			},
			cli.StringFlag{
				Name:   "ca-file",
				EnvVar: "ENVETCD_CA_FILE",
				Usage:  "certificate authority file",
			},
			cli.StringFlag{
				Name:   "cert-file",
				EnvVar: "ENVETCD_CERT_FILE",
				Usage:  "tls client certificate file",
			},
			cli.StringFlag{
				Name:   "key-file",
				EnvVar: "ENVETCD_KEY_FILE",
				Usage:  "tls client key file",
			},
			cli.BoolTFlag{
				Name:   "sanitize",
				EnvVar: "ENVETCD_SANITIZE",
				Usage:  "remove bad characters from values",
			},
			cli.BoolFlag{
				Name:   "upcase",
				EnvVar: "ENVETCD_UPCASE",
				Usage:  "convert all environment keys to uppercase",
			},
		},
		Action: run,
	})
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func run(c *cli.Context) {
	args := c.Args()
	if len(args) < 2 {
		err := fmt.Errorf("cli: missing required arguments prefix and command")
		cli.ShowCommandHelp(c, "run")
		handleError(err, exitCodeParseFlagsError)
	}

	prefix, command := args[0], args[1:]

	runner, err := newRunner(prefix, c, command)
	if err != nil {
		handleError(err, exitCodeRunnerError)
	}

	client, err := getClient(c)
	if err != nil {
		handleError(err, exitCodeError)
	}

	watcher, err := util.NewWatcher(client, runner.Dependencies())
	if err != nil {
		handleError(err, exitCodeWatcherError)
	}

	go watcher.Watch()

	for {
		select {
		case data := <-watcher.DataCh:
			// Tell the Runner about the data
			runner.Receive(data.Data)

			if err := runner.Run(); err != nil {
				handleError(err, exitCodeRunnerError)
			}
		case err := <-watcher.ErrCh:
			log.Printf("[INFO] (cli) watcher got error")
			handleError(err, exitCodeError)
		case <-watcher.FinishCh:
			os.Exit(runner.Wait())
		case exitCode := <-runner.ExitCh:
			log.Printf("[INFO] (cli) subprocess exited")

			if exitCode == exitCodeOK {
				os.Exit(exitCodeOK)
			}

			err := fmt.Errorf("unexpected exit from subprocess (%d)", exitCode)
			handleError(err, exitCode)
		}
	}
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func handleError(err error, status int) {
	log.Printf("[ERR] %s", err.Error())
	os.Exit(status)
}
