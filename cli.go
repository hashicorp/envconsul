package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul-template/config"
	"github.com/hashicorp/consul-template/logging"
	"github.com/hashicorp/consul-template/manager"
	"github.com/hashicorp/consul-template/signals"
	"github.com/hashicorp/envconsul/version"
)

// Exit codes are int values that represent an exit code for a particular error.
// Sub-systems may check this unique error to determine the cause of an error
// without parsing the output or help text.
const (
	ExitCodeOK int = 0

	ExitCodeError = 10 + iota
	ExitCodeInterrupt
	ExitCodeParseFlagsError
	ExitCodeRunnerError
	ExitCodeConfigError
)

var (
	// ErrMissingCommand is returned when no command is specified.
	ErrMissingCommand = fmt.Errorf("No command given")
)

// CLI is the main entry point for envconsul.
type CLI struct {
	sync.Mutex

	// outSteam and errStream are the standard out and standard error streams to
	// write messages from the CLI.
	outStream, errStream io.Writer

	// signalCh is the channel where the cli receives signals.
	signalCh chan os.Signal

	// stopCh is an internal channel used to trigger a shutdown of the CLI.
	stopCh  chan struct{}
	stopped bool
}

// NewCLI creates a new command line interface with the given streams.
func NewCLI(out, err io.Writer) *CLI {
	return &CLI{
		outStream: out,
		errStream: err,
		signalCh:  make(chan os.Signal, 1),
		stopCh:    make(chan struct{}),
	}
}

// Run accepts a slice of arguments and returns an int representing the exit
// status from the command.
func (cli *CLI) Run(args []string) int {
	// Parse the flags and args
	cfg, paths, once, isVersion, err := cli.ParseFlags(args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			fmt.Fprintf(cli.errStream, usage, version.Name)
			return 0
		}
		fmt.Fprintln(cli.errStream, err.Error())
		return ExitCodeParseFlagsError
	}

	// Save original config (defaults + parsed flags) for handling reloads
	cliConfig := cfg.Copy()

	// Load configuration paths, with CLI taking precendence
	cfg, err = loadConfigs(paths, cliConfig)
	if err != nil {
		return logError(err, ExitCodeConfigError)
	}

	cfg.Finalize()

	// Setup the config and logging
	cfg, err = cli.setup(cfg)
	if err != nil {
		return logError(err, ExitCodeConfigError)
	}

	// Print version information for debugging
	log.Printf("[INFO] %s", version.HumanVersion)

	// If the version was requested, return an "error" containing the version
	// information. This might sound weird, but most *nix applications actually
	// print their version on stderr anyway.
	if isVersion {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s\n", version.HumanVersion)
		return ExitCodeOK
	}

	// Return an error if no command was given
	if !config.StringPresent(cfg.Exec.Command) {
		return logError(ErrMissingCommand, ExitCodeConfigError)
	}

	// Initial runner
	runner, err := NewRunner(cfg, once)
	if err != nil {
		return logError(err, ExitCodeRunnerError)
	}
	go runner.Start()

	// Listen for signals
	signal.Notify(cli.signalCh)

	for {
		select {
		case err := <-runner.ErrCh:
			// Check if the runner's error returned a specific exit status, and return
			// that value. If no value was given, return a generic exit status.
			code := ExitCodeRunnerError
			if typed, ok := err.(manager.ErrExitable); ok {
				code = typed.ExitStatus()
			}
			return logError(err, code)
		case <-runner.DoneCh:
			return ExitCodeOK
		case code := <-runner.ExitCh:
			log.Printf("[INFO] (cli) subprocess exited")
			runner.Stop()

			if code == ExitCodeOK {
				return ExitCodeOK
			} else {
				err := fmt.Errorf("unexpected exit from subprocess (%d)", code)
				return logError(err, code)
			}
		case s := <-cli.signalCh:
			log.Printf("[DEBUG] (cli) receiving signal %q", s)

			switch s {
			case *cfg.ReloadSignal:
				fmt.Fprintf(cli.errStream, "Reloading configuration...\n")
				runner.Stop()

				// Re-parse any configuration files or paths
				cfg, err = loadConfigs(paths, cliConfig)
				if err != nil {
					return logError(err, ExitCodeConfigError)
				}
				cfg.Finalize()

				// Load the new configuration from disk
				cfg, err = cli.setup(cfg)
				if err != nil {
					return logError(err, ExitCodeConfigError)
				}

				runner, err = NewRunner(cfg, once)
				if err != nil {
					return logError(err, ExitCodeRunnerError)
				}
				go runner.Start()
			case *cfg.KillSignal:
				fmt.Fprintf(cli.errStream, "Cleaning up...\n")
				runner.Stop()
				return ExitCodeInterrupt
			case signals.SignalLookup["SIGCHLD"]:
				// The SIGCHLD signal is sent to the parent of a child process when it
				// exits, is interrupted, or resumes after being interrupted. We ignore
				// this signal because the child process is monitored on its own.
				//
				// Also, the reason we do a lookup instead of a direct syscall.SIGCHLD
				// is because that isn't defined on Windows.
			case RuntimeSig:
				// ignore these as the runtime uses them with the scheduler
			default:
				// Propogate the signal to the child process
				runner.Signal(s)
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

// ParseFlags is a helper function for parsing command line flags using Go's
// Flag library. This is extracted into a helper to keep the main function
// small, but it also makes writing tests for parsing command line arguments
// much easier and cleaner.
func (cli *CLI) ParseFlags(args []string) (*Config, []string, bool, bool, error) {
	var once, isVersion bool
	var no_prefix *bool
	var c = DefaultConfig()

	// configPaths stores the list of configuration paths on disk
	configPaths := make([]string, 0, 6)

	// Parse the flags and options
	flags := flag.NewFlagSet(version.Name, flag.ContinueOnError)
	flags.SetOutput(ioutil.Discard)
	flags.Usage = func() {}

	flags.Var((funcVar)(func(s string) error {
		configPaths = append(configPaths, s)
		return nil
	}), "config", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.Address = config.String(s)
		return nil
	}), "consul-addr", "")

	flags.Var((funcVar)(func(s string) error {
		a, err := config.ParseAuthConfig(s)
		if err != nil {
			return err
		}
		c.Consul.Auth = a
		return nil
	}), "consul-auth", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Consul.Retry.Enabled = config.Bool(b)
		return nil
	}), "consul-retry", "")

	flags.Var((funcIntVar)(func(i int) error {
		c.Consul.Retry.Attempts = config.Int(i)
		return nil
	}), "consul-retry-attempts", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Retry.Backoff = config.TimeDuration(d)
		return nil
	}), "consul-retry-backoff", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Retry.MaxBackoff = config.TimeDuration(d)
		return nil
	}), "consul-retry-max-backoff", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Consul.SSL.Enabled = config.Bool(b)
		return nil
	}), "consul-ssl", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.CaCert = config.String(s)
		return nil
	}), "consul-ssl-ca-cert", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.CaPath = config.String(s)
		return nil
	}), "consul-ssl-ca-path", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.Cert = config.String(s)
		return nil
	}), "consul-ssl-cert", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.Key = config.String(s)
		return nil
	}), "consul-ssl-key", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.ServerName = config.String(s)
		return nil
	}), "consul-ssl-server-name", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Consul.SSL.Verify = config.Bool(b)
		return nil
	}), "consul-ssl-verify", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.Token = config.String(s)
		return nil
	}), "consul-token", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Transport.DialKeepAlive = config.TimeDuration(d)
		return nil
	}), "consul-transport-dial-keep-alive", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Transport.DialTimeout = config.TimeDuration(d)
		return nil
	}), "consul-transport-dial-timeout", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Consul.Transport.DisableKeepAlives = config.Bool(b)
		return nil
	}), "consul-transport-disable-keep-alives", "")

	flags.Var((funcIntVar)(func(i int) error {
		c.Consul.Transport.MaxIdleConnsPerHost = config.Int(i)
		return nil
	}), "consul-transport-max-idle-conns-per-host", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Transport.TLSHandshakeTimeout = config.TimeDuration(d)
		return nil
	}), "consul-transport-tls-handshake-timeout", "")

	flags.Var((funcVar)(func(s string) error {
		c.Exec.Enabled = config.Bool(true)
		c.Exec.Command = config.String(s)
		return nil
	}), "exec", "")

	flags.Var((funcVar)(func(s string) error {
		sig, err := signals.Parse(s)
		if err != nil {
			return err
		}
		c.Exec.KillSignal = config.Signal(sig)
		return nil
	}), "exec-kill-signal", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Exec.KillTimeout = config.TimeDuration(d)
		return nil
	}), "exec-kill-timeout", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Exec.Splay = config.TimeDuration(d)
		return nil
	}), "exec-splay", "")

	flags.Var((funcVar)(func(s string) error {
		sig, err := signals.Parse(s)
		if err != nil {
			return err
		}
		c.KillSignal = config.Signal(sig)
		return nil
	}), "kill-signal", "")

	flags.Var((funcVar)(func(s string) error {
		c.LogLevel = config.String(s)
		return nil
	}), "log-level", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.MaxStale = config.TimeDuration(d)
		return nil
	}), "max-stale", "")

	// requires post processing (see below) as it depends on -prefix
	flags.Var((funcBoolVar)(func(b bool) error {
		no_prefix = config.Bool(b)
		return nil
	}), "no-prefix", "")

	flags.BoolVar(&once, "once", false, "")

	flags.Var((funcVar)(func(s string) error {
		c.PidFile = config.String(s)
		return nil
	}), "pid-file", "")

	flags.Var((funcVar)(func(s string) error {
		p, err := ParsePrefixConfig(s)
		if err != nil {
			return err
		}
		*c.Prefixes = append(*c.Prefixes, p)
		return nil
	}), "prefix", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Pristine = config.Bool(b)
		return nil
	}), "pristine", "")

	flags.Var((funcVar)(func(s string) error {
		sig, err := signals.Parse(s)
		if err != nil {
			return err
		}
		c.ReloadSignal = config.Signal(sig)
		return nil
	}), "reload-signal", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Sanitize = config.Bool(b)
		return nil
	}), "sanitize", "")

	flags.Var((funcVar)(func(s string) error {
		p, err := ParsePrefixConfig(s)
		if err != nil {
			return err
		}
		*c.Secrets = append(*c.Secrets, p)
		return nil
	}), "secret", "")

	flags.Var((funcVar)(func(s string) error {
		p, err := ParseServiceConfig(s)
		if err != nil {
			return err
		}
		*c.Services = append(*c.Services, p)
		return nil
	}), "service-query", "")

	flags.Var((funcVar)(func(s string) error {
		serviceConfig := c.Services.LastSeviceConfig()
		if serviceConfig == nil {
			return fmt.Errorf("format must be specified before query")
		}
		serviceConfig.FormatId = config.String(s)
		return nil
	}), "service-format-id", "")

	flags.Var((funcVar)(func(s string) error {
		serviceConfig := c.Services.LastSeviceConfig()
		if serviceConfig == nil {
			return fmt.Errorf("format must be specified before query")
		}
		serviceConfig.FormatName = config.String(s)
		return nil
	}), "service-format-name", "")

	flags.Var((funcVar)(func(s string) error {
		serviceConfig := c.Services.LastSeviceConfig()
		if serviceConfig == nil {
			return fmt.Errorf("format must be specified before query")
		}
		serviceConfig.FormatAddress = config.String(s)
		return nil
	}), "service-format-address", "")

	flags.Var((funcVar)(func(s string) error {
		serviceConfig := c.Services.LastSeviceConfig()
		if serviceConfig == nil {
			return fmt.Errorf("format must be specified before query")
		}
		serviceConfig.FormatTag = config.String(s)
		return nil
	}), "service-format-tag", "")

	flags.Var((funcVar)(func(s string) error {
		serviceConfig := c.Services.LastSeviceConfig()
		if serviceConfig == nil {
			return fmt.Errorf("format must be specified before query")
		}
		serviceConfig.FormatPort = config.String(s)
		return nil
	}), "service-format-port", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Syslog.Enabled = config.Bool(b)
		return nil
	}), "syslog", "")

	flags.Var((funcVar)(func(s string) error {
		c.Syslog.Facility = config.String(s)
		return nil
	}), "syslog-facility", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Upcase = config.Bool(b)
		return nil
	}), "upcase", "")

	flags.Var((funcVar)(func(s string) error {
		c.Vault.Address = config.String(s)
		return nil
	}), "vault-addr", "")

	flags.Var((funcVar)(func(s string) error {
		c.Vault.Namespace = config.String(s)
		return nil
	}), "vault-namespace", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Vault.RenewToken = config.Bool(b)
		return nil
	}), "vault-renew-token", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Vault.Retry.Enabled = config.Bool(b)
		return nil
	}), "vault-retry", "")

	flags.Var((funcIntVar)(func(i int) error {
		c.Vault.Retry.Attempts = config.Int(i)
		return nil
	}), "vault-retry-attempts", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Vault.Retry.Backoff = config.TimeDuration(d)
		return nil
	}), "vault-retry-backoff", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Vault.Retry.MaxBackoff = config.TimeDuration(d)
		return nil
	}), "vault-retry-max-backoff", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Vault.SSL.Enabled = config.Bool(b)
		return nil
	}), "vault-ssl", "")

	flags.Var((funcVar)(func(s string) error {
		c.Vault.SSL.CaCert = config.String(s)
		return nil
	}), "vault-ssl-ca-cert", "")

	flags.Var((funcVar)(func(s string) error {
		c.Vault.SSL.CaPath = config.String(s)
		return nil
	}), "vault-ssl-ca-path", "")

	flags.Var((funcVar)(func(s string) error {
		c.Vault.SSL.Cert = config.String(s)
		return nil
	}), "vault-ssl-cert", "")

	flags.Var((funcVar)(func(s string) error {
		c.Vault.SSL.Key = config.String(s)
		return nil
	}), "vault-ssl-key", "")

	flags.Var((funcVar)(func(s string) error {
		c.Vault.SSL.ServerName = config.String(s)
		return nil
	}), "vault-ssl-server-name", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Vault.SSL.Verify = config.Bool(b)
		return nil
	}), "vault-ssl-verify", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Vault.Transport.DialKeepAlive = config.TimeDuration(d)
		return nil
	}), "vault-transport-dial-keep-alive", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Vault.Transport.DialTimeout = config.TimeDuration(d)
		return nil
	}), "vault-transport-dial-timeout", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Vault.Transport.DisableKeepAlives = config.Bool(b)
		return nil
	}), "vault-transport-disable-keep-alives", "")

	flags.Var((funcIntVar)(func(i int) error {
		c.Vault.Transport.MaxIdleConnsPerHost = config.Int(i)
		return nil
	}), "vault-transport-max-idle-conns-per-host", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Vault.Transport.TLSHandshakeTimeout = config.TimeDuration(d)
		return nil
	}), "vault-transport-tls-handshake-timeout", "")

	flags.Var((funcVar)(func(s string) error {
		c.Vault.Token = config.String(s)
		return nil
	}), "vault-token", "")
	flags.Var((funcVar)(func(s string) error {
		c.Vault.VaultAgentTokenFile = config.String(s)
		return nil
	}), "vault-agent-token-file", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Vault.UnwrapToken = config.Bool(b)
		return nil
	}), "vault-unwrap-token", "")

	flags.Var((funcVar)(func(s string) error {
		w, err := config.ParseWaitConfig(s)
		if err != nil {
			return err
		}
		c.Wait = w
		return nil
	}), "wait", "")

	flags.BoolVar(&isVersion, "v", false, "")
	flags.BoolVar(&isVersion, "version", false, "")

	// Deprecations
	// TODO remove in 0.8.0
	flags.Var((funcVar)(func(s string) error {
		log.Printf("[WARN] -auth is now -consul-auth")
		a, err := config.ParseAuthConfig(s)
		if err != nil {
			return err
		}
		c.Consul.Auth = a
		return nil
	}), "auth", "")
	flags.Var((funcVar)(func(s string) error {
		log.Printf("[WARN] -consul is now -consul-addr")
		c.Consul.Address = config.String(s)
		return nil
	}), "consul", "")
	flags.Var((funcDurationVar)(func(d time.Duration) error {
		log.Printf("[WARN] -retry is now -consul-retry-* and -vault-retry-*")
		c.Consul.Retry.Backoff = config.TimeDuration(d)
		c.Consul.Retry.MaxBackoff = config.TimeDuration(d)
		c.Vault.Retry.Backoff = config.TimeDuration(d)
		c.Vault.Retry.MaxBackoff = config.TimeDuration(d)
		return nil
	}), "retry", "")
	flags.Var((funcDurationVar)(func(d time.Duration) error {
		log.Printf("[WARN] -splay is now -exec-splay")
		c.Exec.Splay = config.TimeDuration(d)
		return nil
	}), "splay", "")
	flags.Var((funcBoolVar)(func(b bool) error {
		log.Printf("[WARN] -ssl is now -consul-ssl-* and -vault-ssl-*")
		c.Consul.SSL.Enabled = config.Bool(b)
		c.Vault.SSL.Enabled = config.Bool(b)
		return nil
	}), "ssl", "")
	flags.Var((funcBoolVar)(func(b bool) error {
		log.Printf("[WARN] -ssl-verify is now -consul-ssl-verify and -vault-ssl-verify")
		c.Consul.SSL.Verify = config.Bool(b)
		c.Vault.SSL.Verify = config.Bool(b)
		return nil
	}), "ssl-verify", "")
	flags.Var((funcVar)(func(s string) error {
		log.Printf("[WARN] -ssl-ca-cert is now -consul-ssl-ca-cert and -vault-ssl-ca-cert")
		c.Consul.SSL.CaCert = config.String(s)
		c.Vault.SSL.CaCert = config.String(s)
		return nil
	}), "ssl-ca-cert", "")
	flags.Var((funcVar)(func(s string) error {
		log.Printf("[WARN] -ssl-cert is now -consul-ssl-cert and -vault-ssl-cert")
		c.Consul.SSL.Cert = config.String(s)
		c.Vault.SSL.Cert = config.String(s)
		return nil
	}), "ssl-cert", "")
	flags.Var((funcDurationVar)(func(d time.Duration) error {
		log.Printf("[WARN] -timeout is now -exec-timeout")
		c.Exec.Timeout = config.TimeDuration(d)
		return nil
	}), "timeout", "")
	flags.Var((funcVar)(func(s string) error {
		log.Printf("[WARN] -token is now -consul-token")
		c.Consul.Token = config.String(s)
		return nil
	}), "token", "")
	// End deprecations
	// TODO remove in 0.8.0

	// If there was a parser error, stop
	if err := flags.Parse(args); err != nil {
		return nil, nil, false, false, err
	}

	// Post-processing of no-prefix option
	if no_prefix != nil {
		for _, p := range *c.Prefixes {
			p.NoPrefix = no_prefix
		}
		for _, s := range *c.Secrets {
			s.NoPrefix = no_prefix
		}
	}

	// Convert any arguments given after to the command, but a command specified
	// via the flag takes precedence.
	if c.Exec.Command == nil {
		if command := strings.Join(flags.Args(), " "); command != "" {
			c.Exec.Enabled = config.Bool(true)
			c.Exec.Command = config.String(command)
		}
	}

	return c, configPaths, once, isVersion, nil
}

// loadConfigs loads the configuration from the list of paths. The optional
// configuration is the list of overrides to apply at the very end, taking
// precendence over any configurations that were loaded from the paths. If any
// errors occur when reading or parsing those sub-configs, it is returned.
func loadConfigs(paths []string, o *Config) (*Config, error) {
	finalC := DefaultConfig()

	for _, path := range paths {
		c, err := FromPath(path)
		if err != nil {
			return nil, err
		}

		finalC = finalC.Merge(c)
	}

	finalC = finalC.Merge(o)
	finalC.Finalize()
	return finalC, nil
}

// logError logs an error message and then returns the given status.
func logError(err error, status int) int {
	log.Printf("[ERR] (cli) %s", err)
	return status
}

func (cli *CLI) setup(conf *Config) (*Config, error) {
	if err := logging.Setup(&logging.Config{
		SyslogName:     version.Name,
		Level:          config.StringVal(conf.LogLevel),
		Syslog:         config.BoolVal(conf.Syslog.Enabled),
		SyslogFacility: config.StringVal(conf.Syslog.Facility),
		Writer:         cli.errStream,
	}); err != nil {
		return nil, err
	}

	return conf, nil
}

const usage = `Usage: %s [options] <command>

  Watches values from Consul's K/V store and Vault secrets to set environment
  variables when the values are changed. It spawns a child process populated
  with the environment variables.

Options:

  -config=<path>
      Sets the path to a configuration file or folder on disk. This can be
      specified multiple times to load multiple files or folders. If multiple
      values are given, they are merged left-to-right, and CLI arguments take
      the top-most precedence.

  -consul-addr=<address>
      Sets the address of the Consul instance

  -consul-auth=<username[:password]>
      Set the basic authentication username and password for communicating
      with Consul.

  -consul-retry
      Use retry logic when communication with Consul fails

  -consul-retry-attempts=<int>
      The number of attempts to use when retrying failed communications

  -consul-retry-backoff=<duration>
      The base amount to use for the backoff duration. This number will be
      increased exponentially for each retry attempt.

  -consul-retry-max-backoff=<duration>
      The maximum limit of the retry backoff duration. Default is one minute.
      0 means infinite. The backoff will increase exponentially until given value.

  -consul-ssl
      Use SSL when connecting to Consul

  -consul-ssl-ca-cert=<string>
      Validate server certificate against this CA certificate file list

  -consul-ssl-ca-path=<string>
      Sets the path to the CA to use for TLS verification

  -consul-ssl-cert=<string>
      SSL client certificate to send to server

  -consul-ssl-key=<string>
      SSL/TLS private key for use in client authentication key exchange

  -consul-ssl-server-name=<string>
      Sets the name of the server to use when validating TLS.

  -consul-ssl-verify
      Verify certificates when connecting via SSL

  -consul-token=<token>
      Sets the Consul API token

  -consul-transport-dial-keep-alive=<duration>
      Sets the amount of time to use for keep-alives

  -consul-transport-dial-timeout=<duration>
      Sets the amount of time to wait to establish a connection

  -consul-transport-disable-keep-alives
      Disables keep-alives (this will impact performance)

  -consul-transport-max-idle-conns-per-host=<int>
      Sets the maximum number of idle connections to permit per host

  -consul-transport-tls-handshake-timeout=<duration>
      Sets the handshake timeout

  -exec=<command>
      Enable exec mode to run as a supervisor-like process - the given command
      will receive all signals provided to the parent process and will receive a
      signal when templates change

  -exec-kill-signal=<signal>
      Signal to send when gracefully killing the process

  -exec-kill-timeout=<duration>
      Amount of time to wait before force-killing the child

  -exec-splay=<duration>
      Amount of time to wait before sending signals

  -kill-signal=<signal>
      Signal to listen to gracefully terminate the process

  -log-level=<level>
      Set the logging level - values are "debug", "info", "warn", and "err"

  -max-stale=<duration>
      Set the maximum staleness and allow stale queries to Consul which will
      distribute work among all servers instead of just the leader

  -no-prefix[=<bool>]
	  Tells Envconsul to not prefix the keys with their parent "folder".

  -once
      Do not run the process as a daemon

  -pid-file=<path>
      Path on disk to write the PID of the process

  -prefix=<prefix>
      Add a prefix to watch (to the right of configured prefixes), multiple
      prefixes are merged from left to right, with the right-most result taking
      precedence, including any values specified with -secret (secrets
      overrides prefixes)

  -pristine
      Only use values retrieved from prefixes and secrets, do not inherit the
      existing environment variables

  -reload-signal=<signal>
      Signal to listen to reload configuration

  -sanitize
      Replace invalid characters in keys to underscores

  -secret=<prefix>
      Add a secret path to watch in Vault (to the right of configured secrets),
      multiple prefixes are merged from left to right, with the right-most
      result taking precedence, including any values specified with -prefix
      (secrets overrides prefixes)

  -service-query=<service-name>
      A query to watch in Consul service parameters

  -service-format-id=<{{service}}/{{key}}>
      Format key environment for service id.

  -service-format-name=<{{service}}/{{key}}>
      Format key environment for service name.

  -service-format-address=<{{service}}/{{key}}>
      Format key environment for service address.

  -service-format-tag=<{{service}}/{{key}}>
      Format key environment for service tag.

  -service-format-port=<{{service}}/{{key}}>
      Format key environment for service port.

  -syslog
      Send the output to syslog instead of standard error and standard out. The
      syslog facility defaults to LOCAL0 and can be changed using a
      configuration file

  -syslog-facility=<facility>
      Set the facility where syslog should log - if this attribute is supplied,
      the -syslog flag must also be supplied

  -upcase
      Convert all environment variable keys to uppercase

  -vault-addr=<address>
      Sets the address of the Vault server

  -vault-namespace=<namespace>
      Sets the Vault namespace

  -vault-renew-token
	  Periodically renew the provided Vault API token - this defaults to "true"
	  and will renew the token at half of the lease duration (unless
	  vault-agent-token-file is set, then it defaults to false as it is
	  presumed the vault-agent will take care of renewing)

  -vault-retry
      Use retry logic when communication with Vault fails

  -vault-retry-attempts=<int>
      The number of attempts to use when retrying failed communications

  -vault-retry-backoff=<duration>
      The base amount to use for the backoff duration. This number will be
      increased exponentially for each retry attempt.

  -vault-retry-max-backoff=<duration>
      The maximum limit of the retry backoff duration. Default is one minute.
      0 means infinite. The backoff will increase exponentially until given value.

  -vault-ssl
      Specifies whether communications with Vault should be done via SSL

  -vault-ssl-ca-cert=<string>
      Sets the path to the CA certificate to use for TLS verification

  -vault-ssl-ca-path=<string>
      Sets the path to the CA to use for TLS verification

  -vault-ssl-cert=<string>
      Sets the path to the certificate to use for TLS verification

  -vault-ssl-key=<string>
      Sets the path to the key to use for TLS verification

  -vault-ssl-server-name=<string>
      Sets the name of the server to use when validating TLS.

  -vault-ssl-verify
      Enable SSL verification for communications with Vault.

  -vault-token=<token>
      Sets the Vault API token

  -vault-agent-token-file=<token-file>
      File to read Vault API token from.

  -vault-transport-dial-keep-alive=<duration>
      Sets the amount of time to use for keep-alives

  -vault-transport-dial-timeout=<duration>
      Sets the amount of time to wait to establish a connection

  -vault-transport-disable-keep-alives
      Disables keep-alives (this will impact performance)

  -vault-transport-max-idle-conns-per-host=<int>
      Sets the maximum number of idle connections to permit per host

  -vault-transport-tls-handshake-timeout=<duration>
      Sets the handshake timeout

  -vault-unwrap-token
      Unwrap the provided Vault API token (see Vault documentation for more
      information on this feature)

  -wait=<duration>
      Sets the 'min(:max)' amount of time to wait before writing a template (and
      triggering a command)

  -v, -version
      Print the version of this daemon
`
