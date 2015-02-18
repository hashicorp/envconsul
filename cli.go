package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/consul-template/logging"
	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/consul/api"
)

// Exit codes are int valuse that represent an exit code for a particular error.
// Sub-systems may check this unique error to determine the cause of an error
// without parsing the output or help text.
const (
	ExitCodeOK int = 0

	// Errors start at 10
	ExitCodeError = 10 + iota
	ExitCodeLoggingError
	ExitCodeParseFlagsError
	ExitCodeParseConfigError
	ExitCodeRunnerError
	ExitCodeConsulAPIError
	ExitCodeWatcherError
)

/// ------------------------- ///

// CLI is the main entry point for envconsul.
type CLI struct {
	sync.Mutex

	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer

	// stopCh is an internal channel used to trigger a shutdown of the CLI.
	stopCh  chan struct{}
	stopped bool
}

func NewCLI(out, err io.Writer) *CLI {
	return &CLI{
		outStream: out,
		errStream: err,
		stopCh:    make(chan struct{}),
	}
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func (cli *CLI) Run(args []string) int {
	// Parse the flags and args
	config, parsedArgs, once, version, err := cli.parseFlags(args[1:])
	if err != nil {
		return cli.handleError(err, ExitCodeParseFlagsError)
	}

	// Setup the logging
	if err := logging.Setup(&logging.Config{
		Name:           Name,
		Level:          config.LogLevel,
		Syslog:         config.Syslog.Enabled,
		SyslogFacility: config.Syslog.Facility,
		Writer:         cli.errStream,
	}); err != nil {
		return cli.handleError(err, ExitCodeLoggingError)
	}

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if version {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s v%s\n", Name, Version)
		return ExitCodeOK
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

	if len(parsedArgs) < 2 {
		err := fmt.Errorf("cli: missing required arguments prefix and command")
		return cli.handleError(err, ExitCodeParseFlagsError)
	}

	prefix, command := parsedArgs[0], parsedArgs[1:]

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

	log.Printf("[DEBUG] (cli) creating Watcher")
	watcher, err := watch.NewWatcher(&watch.WatcherConfig{
		Client:   client,
		Once:     once,
		MaxStale: config.MaxStale,
		RetryFunc: func(current time.Duration) time.Duration {
			return config.Retry
		},
	})
	if err != nil {
		return cli.handleError(err, ExitCodeWatcherError)
	}

	for _, dep := range runner.Dependencies() {
		if _, err := watcher.Add(dep); err != nil {
			return cli.handleError(err, ExitCodeWatcherError)
		}
	}

	var minTimer, maxTimer <-chan time.Time

	for {
		log.Printf("[DEBUG] (cli) looping for data")

		select {
		case data := <-watcher.DataCh:
			log.Printf("[INFO] (cli) received %s from Watcher", data.Dependency.Display())

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
		case <-cli.stopCh:
			return ExitCodeOK
		}
	}
}

// stop is used internally to shutdown a running CLI
func (cli *CLI) stop() {
	cli.Lock()
	defer cli.Unlock()

	if cli.stopped {
		return
	}

	close(cli.stopCh)
	cli.stopped = true
}

// parseFlags is a helper function for parsing command line flags using Go's
// Flag library. This is extracted into a helper to keep the main function
// small, but it also makes writing tests for parsing command line arguments
// much easier and cleaner.
func (cli *CLI) parseFlags(args []string) (*Config, []string, bool, bool, error) {
	var once, version bool
	var config = DefaultConfig()

	// Parse the flags and options
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)
	flags.Usage = func() {
		fmt.Fprintf(cli.errStream, usage, Name)
	}
	flags.StringVar(&config.Consul, "consul", config.Consul, "")
	flags.StringVar(&config.Token, "token", config.Token, "")
	flags.Var((*authVar)(config.Auth), "auth", "")
	flags.BoolVar(&config.SSL.Enabled, "ssl", config.SSL.Enabled, "")
	flags.BoolVar(&config.SSL.Verify, "ssl-verify", config.SSL.Verify, "")
	flags.DurationVar(&config.MaxStale, "max-stale", config.MaxStale, "")
	flags.BoolVar(&config.Syslog.Enabled, "syslog", config.Syslog.Enabled, "")
	flags.StringVar(&config.Syslog.Facility, "syslog-facility", config.Syslog.Facility, "")
	flags.Var((*watch.WaitVar)(config.Wait), "wait", "")
	flags.DurationVar(&config.Retry, "retry", config.Retry, "")
	flags.BoolVar(&config.Sanitize, "sanitize", config.Sanitize, "")
	flags.BoolVar(&config.Upcase, "upcase", config.Upcase, "")
	flags.StringVar(&config.Path, "config", config.Path, "")
	flags.StringVar(&config.LogLevel, "log-level", config.LogLevel, "")
	flags.BoolVar(&once, "once", false, "")
	flags.BoolVar(&version, "version", false, "")

	// If there was a parser error, stop
	if err := flags.Parse(args); err != nil {
		return nil, nil, false, false, err
	}

	return config, flags.Args(), once, version, nil
}

// handleError outputs the given error's Error() to the errStream and returns
// the given exit status.
func (cli *CLI) handleError(err error, status int) int {
	log.Printf("[ERR] %s", err.Error())
	return status
}

const usage = `
Usage: %s [options]

  Watches values from Consul's K/V store and sets environment variables when
  Consul values are changed.

Options:

  -auth=<user[:pass]>      Set the basic authentication username (and password)
  -consul=<address>        Sets the address of the Consul instance
  -max-stale=<duration>    Set the maximum staleness and allow stale queries to
                           Consul which will distribute work among all servers
                           instead of just the leader
  -ssl                     Use SSL when connecting to Consul
  -ssl-verify              Verify certificates when connecting via SSL
  -token=<token>           Sets the Consul API token

  -syslog                  Send the output to syslog instead of standard error
                           and standard out. The syslog facility defaults to
                           LOCAL0 and can be changed using a configuration file
  -syslog-facility=<f>     Set the facility where syslog should log. If this
                           attribute is supplied, the -syslog flag must also be
                           supplied.

  -wait=<duration>         Sets the 'minumum(:maximum)' amount of time to wait
                           before writing a triggering a restart
  -retry=<duration>        The amount of time to wait if Consul returns an
                           error when communicating with the API

  -sanitize                Replace invalid characters in keys to underscores
  -upcase                  Convert all environment variable keys to uppercase


  -config=<path>           Sets the path to a configuration file on disk

  -log-level=<level>       Set the logging level - valid values are "debug",
                           "info", "warn" (default), and "err"

  -once                    Do not run the process as a daemon
  -version                 Print the version of this daemon
`
