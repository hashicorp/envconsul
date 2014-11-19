package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	api "github.com/armon/consul-api"
	"github.com/hashicorp/consul-template/util"
	"github.com/hashicorp/logutils"
)

// Exit codes are int valuse that represent an exit code for a particular error.
// Sub-systems may check this unique error to determine the cause of an error
// without parsing the output or help text.
const (
	ExitCodeOK int = 0

	// Errors start at 10
	ExitCodeError = 10 + iota
	ExitCodeParseFlagsError
	ExitCodeParseWaitError
	ExitCodeParseConfigError
	ExitCodeRunnerError
	ExitCodeConsulAPIError
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
	cli.initLogger()

	// TODO: remove in v0.4.0 (deprecated)
	var address, datacenter string
	var errExit, terminate, reload bool

	var version, once bool
	var config = new(Config)

	// Parse the flags and options
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprintf(cli.errStream, usage, Name)
	}
	flags.StringVar(&config.Consul, "consul", "",
		"address of the Consul instance")
	flags.StringVar(&config.Token, "token", "",
		"a consul API token")
	flags.StringVar(&config.WaitRaw, "wait", "",
		"the minimum(:maximum) to wait before updating the environment")
	flags.StringVar(&config.Path, "config", "",
		"the path to a config file on disk")
	flags.DurationVar(&config.Timeout, "timeout", 0,
		"the time to wait for a process to restart")
	flags.BoolVar(&config.Sanitize, "sanitize", true,
		"remove bad characters from values")
	flags.BoolVar(&config.Upcase, "upcase", true,
		"convert all environment keys to uppercase")
	flags.BoolVar(&once, "once", false,
		"do not run as a daemon")
	flags.BoolVar(&version, "version", false, "display the version")

	// TODO: remove in v0.4.0 (deprecated)
	flags.StringVar(&datacenter, "dc", "",
		"DEPRECATED")
	flags.StringVar(&address, "addr", "",
		"DEPRECATED")
	flags.BoolVar(&errExit, "errexit", false,
		"DEPRECAETD")
	flags.BoolVar(&terminate, "terminate", false,
		"DEPRECAETD")
	flags.BoolVar(&reload, "reload", false,
		"DEPRECATED")

	// If there was a parser error, stop
	if err := flags.Parse(args[1:]); err != nil {
		return cli.handleError(err, ExitCodeParseFlagsError)
	}

	// TODO: remove in v0.4.0 (deprecated)
	if address != "" {
		fmt.Fprintf(cli.errStream,
			"DEPRECATED: the -addr flag is deprecated, please use -consul instead\n")
		config.Consul = address
	}

	if datacenter != "" {
		fmt.Fprintf(cli.errStream,
			"DEPRECATED: the -dc flag is deprecated, please use the @dc syntax instead\n")
	}

	if errExit {
		fmt.Fprintf(cli.errStream, "DEPRECATED: the -errexit flag is deprecated\n")
	}

	if terminate {
		fmt.Fprintf(cli.errStream,
			"DEPRECATED: the -terminate flag is deprecated, use -once instead\n")
	}

	if reload {
		fmt.Fprintf(cli.errStream,
			"DEPRECATED: the -reload flag is deprecated, use -once instead\n")
	}

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if version {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s v%s\n", Name, Version)
		return ExitCodeOK
	}

	// Parse the raw wait value into a Wait object
	if config.WaitRaw != "" {
		log.Printf("[DEBUG] (cli) detected -wait, parsing")
		wait, err := util.ParseWait(config.WaitRaw)
		if err != nil {
			return cli.handleError(err, ExitCodeParseWaitError)
		}
		config.Wait = wait
	}

	// Merge a path config with the command line options. Command line options
	// take precedence over config file options for easy overriding.
	if config.Path != "" {
		log.Printf("[DEBUG] (cli) detected -config, merging")
		fileConfig, err := ParseConfig(config.Path)
		if err != nil {
			return cli.handleError(err, ExitCodeParseConfigError)
		}

		fileConfig.Merge(config)
		config = fileConfig
	}

	args = flags.Args()
	if len(args) < 2 {
		err := fmt.Errorf("cli: missing required arguments prefix and command")
		return cli.handleError(err, ExitCodeParseFlagsError)
	}

	prefix, command := args[0], args[1:]

	log.Printf("[DEBUG] (cli) creating Runner")
	runner, err := NewRunner(prefix, config, command)
	if err != nil {
		return cli.handleError(err, ExitCodeRunnerError)
	}

	log.Printf("[DEBUG] (cli) creating Consul API client")
	consulConfig := api.DefaultConfig()
	if config.Consul != "" {
		consulConfig.Address = config.Consul
	}
	if config.Token != "" {
		consulConfig.Token = config.Token
	}
	client, err := api.NewClient(consulConfig)
	if err != nil {
		return cli.handleError(err, ExitCodeConsulAPIError)
	}
	if _, err := client.Agent().NodeName(); err != nil {
		return cli.handleError(err, ExitCodeConsulAPIError)
	}

	log.Printf("[DEBUG] (cli) creating Watcher")
	watcher, err := util.NewWatcher(client, runner.Dependencies())
	if err != nil {
		return cli.handleError(err, ExitCodeWatcherError)
	}

	go watcher.Watch(once)

	var minTimer, maxTimer <-chan time.Time

	for {
		log.Printf("[DEBUG] (cli) looping for data")

		select {
		case data := <-watcher.DataCh:
			log.Printf("[INFO] (cli) received %s from Watcher", data.Display())

			// Tell the Runner about the data
			runner.Receive(data.Data)

			// If we are waiting for quiescence, setup the timers
			if config.Wait != nil {
				log.Printf("[DEBUG] (cli) detected quiescence, starting timers")

				// Reset the min timer
				minTimer = time.After(config.Wait.Min)

				// Set the max timer if it does not already exist
				if maxTimer == nil {
					maxTimer = time.After(config.Wait.Max)
				}
			} else {
				log.Printf("[INFO] (cli) invoking Runner")
				if err := runner.Run(); err != nil {
					return cli.handleError(err, ExitCodeRunnerError)
				}
			}
		case <-minTimer:
			log.Printf("[DEBUG] (cli) quiescence minTimer fired, invoking Runner")

			minTimer, maxTimer = nil, nil

			if err := runner.Run(); err != nil {
				return cli.handleError(err, ExitCodeRunnerError)
			}
		case <-maxTimer:
			log.Printf("[DEBUG] (cli) quiescence maxTimer fired, invoking Runner")

			minTimer, maxTimer = nil, nil

			if err := runner.Run(); err != nil {
				return cli.handleError(err, ExitCodeRunnerError)
			}
		case err := <-watcher.ErrCh:
			log.Printf("[INFO] (cli) watcher got error")
			return cli.handleError(err, ExitCodeError)
		case <-watcher.FinishCh:
			log.Printf("[INFO] (cli) received finished signal")
			return runner.Wait()
		case exitCode := <-runner.ExitCh:
			log.Printf("[INFO] (cli) subprocess exited")

			if exitCode == ExitCodeOK {
				return ExitCodeOK
			} else {
				err := fmt.Errorf("unexpected exit from subprocess (%d)", exitCode)
				return cli.handleError(err, exitCode)
			}
		}
	}
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (cli *CLI) handleError(err error, status int) int {
	log.Printf("[ERR] %s", err.Error())
	return status
}

// initLogger gets the log level from the environment, falling back to DEBUG if
// nothing was given.
func (cli *CLI) initLogger() {
	minLevel := strings.ToUpper(strings.TrimSpace(os.Getenv("ENV_CONSUL_LOG")))
	if minLevel == "" {
		minLevel = "WARN"
	}

	levelFilter := &logutils.LevelFilter{
		Levels: []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERR"},
		Writer: cli.errStream,
	}

	levelFilter.SetMinLevel(logutils.LogLevel(minLevel))

	log.SetOutput(levelFilter)
}

const usage = `
Usage: %s [options]

  Watches values from Consul's K/V store and sets environment variables when
  Consul values are changed.

Options:

  -consul=<address>    Sets the address of the Consul instance
  -token=<token>       Sets the Consul API token
  -config=<path>       Sets the path to a configuration file on disk
  -wait=<duration>     Sets the 'minumum(:maximum)' amount of time to wait
                       before writing the environment (and triggering a command)
  -timeout=<time>      Sets the duration to wait for SIGTERM during a reload

  -sanitize            Replace invalid characters in keys to underscores
  -upcase              Convert all environment variable keys to uppercase

  -once                Do not poll for changes
  -version             Print the version of this daemon
`
