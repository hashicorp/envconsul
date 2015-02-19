package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/consul/api"
)

// Regexp for invalid characters in keys
var InvalidRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)

type Runner struct {
	sync.RWMutex

	// // Prefix is the KeyPrefixDependency associated with this Runner.
	// Prefix *dependency.StoreKeyPrefix

	// ErrCh and DoneCh are channels where errors and finish notifications occur.
	ErrCh  chan error
	DoneCh chan struct{}

	// ExitCh is a channel for parent processes to read exit status values from
	// the child processes.
	ExitCh chan int

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *Config

	// client is the consul/api client.
	client *api.Client

	// once indicates the runner should get data exactly one time and then stop.
	once bool

	// minTimer and maxTimer are used for quiescence.
	minTimer, maxTimer <-chan time.Time

	// outStream and errStream are the io.Writer streams where the runner will
	// write information.
	outStream, errStream io.Writer

	// watcher is the watcher this runner is using.
	watcher *watch.Watcher

	// data is the latest representation of the data from Consul.
	data map[string][]*dep.KeyPair

	// env is the last compiled environment.
	env map[string]string

	// command is the string of the command to run. cmd is the last known instance
	// of the running command.
	command []string
	cmd     *exec.Cmd
}

// NewRunner accepts a config, command, and boolean value for once mode.
func NewRunner(config *Config, command []string, once bool) (*Runner, error) {
	log.Printf("[INFO] (runner) creating new runner (command: %v, once: %v)", command, once)

	runner := &Runner{
		config:  config,
		command: command,
		once:    once,
	}

	if err := runner.init(); err != nil {
		return nil, err
	}

	return runner, nil
}

// Start creates a new runner and begins watching dependencies and quiescence
// timers. This is the main event loop and will block until finished.
func (r *Runner) Start() {
	log.Printf("[INFO] (runner) starting")

	// Add the dependencies to the watcher
	for _, prefix := range r.config.Prefixes {
		r.watcher.Add(prefix)
	}

	var err error
	var exitCh <-chan int

	for {
		select {
		case data := <-r.watcher.DataCh:
			r.Receive(data.Dependency, data.Data)

			// Drain all views that have data
		OUTER:
			for {
				select {
				case data = <-r.watcher.DataCh:
					r.Receive(data.Dependency, data.Data)
				default:
					break OUTER
				}
			}

			// If we are waiting for quiescence, setup the timers
			if r.config.Wait.Min != 0 && r.config.Wait.Max != 0 {
				log.Printf("[INFO] (runner) quiescence timers starting")
				r.minTimer = time.After(r.config.Wait.Min)
				if r.maxTimer == nil {
					r.maxTimer = time.After(r.config.Wait.Max)
				}
				continue
			}
		case <-r.minTimer:
			log.Printf("[INFO] (runner) quiescence minTimer fired")
			r.minTimer, r.maxTimer = nil, nil
		case <-r.maxTimer:
			log.Printf("[INFO] (runner) quiescence maxTimer fired")
			r.minTimer, r.maxTimer = nil, nil
		case err := <-r.watcher.ErrCh:
			// Intentionally do not send the error back up to the runner. Eventually,
			// once Consul API implements errwrap and multierror, we can check the
			// "type" of error and conditionally alert back.
			//
			// if err.Contains(Something) {
			//   errCh <- err
			// }
			log.Printf("[ERR] (runner) watcher reported error: %s", err)
		case <-r.watcher.FinishCh:
			log.Printf("[INFO] (runner) watcher reported finish")
			return
		case code := <-exitCh:
			r.ExitCh <- code
		case <-r.DoneCh:
			log.Printf("[INFO] (runner) received finish")
			return
		}

		// If we got this far, that means we got new data or one of the timers
		// fired, so attempt to re-process the environment.
		exitCh, err = r.Run()
		if err != nil {
			r.ErrCh <- err
			return
		}
	}
}

// Stop halts the execution of this runner and its subprocesses.
func (r *Runner) Stop() {
	log.Printf("[INFO] (runner) stopping")
	r.watcher.Stop()

	// Stop the process if it is running
	if r.cmd != nil {
		log.Printf("[DEBUG] (runner) killing child process")
		r.killProcess()
	}

	close(r.DoneCh)
}

// Receive accepts data from Consul and maps that data to the prefix.
func (r *Runner) Receive(d dep.Dependency, data interface{}) {
	r.Lock()
	defer r.Unlock()
	r.data[d.HashCode()] = data.([]*dep.KeyPair)
}

// Signal sends a signal to the child process, if it exists. Any errors that
// occur are returned.
func (r *Runner) Signal(sig os.Signal) error {
	if r.cmd == nil || r.cmd.Process == nil {
		log.Printf("[WARN] (runner) attempted to send %s to subprocess, "+
			"but it does not exist ", sig.String())
		return nil
	}

	return r.cmd.Process.Signal(sig)
}

// Run executes and manages the child process with the correct environment. The
// current enviornment is also copied into the child process environment.
func (r *Runner) Run() (<-chan int, error) {
	log.Printf("[INFO] (runner) running")

	env := make(map[string]string)

	// Iterate over each dependency and pull out its data. If any dependencies do
	// not have data yet, this function will immediately return because we cannot
	// safely continue until all dependencies have received data at least once.
	//
	// We iterate over the list of config prefixes so that order is maintained,
	// since order in a map is not deterministic.
	for _, dep := range r.config.Prefixes {
		data, ok := r.data[dep.HashCode()]
		if !ok {
			log.Printf("[INFO] (runner) missing data for %s", dep.Display())
			return nil, nil
		}

		// For each pair, update the environment hash. Subsequent runs could
		// overwrite an existing key.
		for _, pair := range data {
			key, value := pair.Key, string(pair.Value)

			if r.config.Sanitize {
				key = InvalidRegexp.ReplaceAllString(key, "_")
			}

			if r.config.Upcase {
				key = strings.ToUpper(key)
			}

			if current, ok := env[key]; ok {
				log.Printf("[DEBUG] (runner) overwriting %s=%q (was %q)", key, value, current)
				env[key] = value
			} else {
				log.Printf("[DEBUG] (runner) setting %s=%q", key, value)
				env[key] = value
			}
		}
	}

	// Print the final environment
	log.Printf("[DEBUG] Environment:")
	for k, v := range env {
		log.Printf("[DEBUG]   %s=%q", k, v)
	}

	// If the resulting map is the same, do not do anything
	if reflect.DeepEqual(r.env, env) {
		log.Printf("[INFO] (runner) environment was the same")
		return nil, nil
	}

	// Update the environment
	r.env = env

	// Restart the current process if it exists
	if r.cmd != nil && r.cmd.Process != nil {
		r.killProcess()
	}

	// Create a new environment
	processEnv := os.Environ()
	cmdEnv := make([]string, len(processEnv), len(r.env)+len(processEnv))
	copy(cmdEnv, processEnv)
	for k, v := range r.env {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(r.command[0], r.command[1:]...)
	cmd.Stdout = r.outStream
	cmd.Stderr = r.errStream
	cmd.Env = cmdEnv
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	r.cmd = cmd

	// Create a new exitCh so that previously invoked commands
	// (if any) don't cause us to exit, and start a goroutine
	// to wait for that process to end.
	exitCh := make(chan int, 1)
	go func() {
		err := cmd.Wait()
		if err == nil {
			exitCh <- ExitCodeOK
			return
		}

		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitCh <- status.ExitStatus()
				return
			}
		}

		exitCh <- ExitCodeError
	}()

	return exitCh, nil
}

// init creates the Runner's underlying data structures and returns an error if
// any problems occur.
func (r *Runner) init() error {
	// Merge multiple configs if given
	if r.config.Path != "" {
		err := buildConfig(r.config, r.config.Path)
		if err != nil {
			return fmt.Errorf("runner: %s", err)
		}
	}

	// Add default values for the config
	r.config.Merge(DefaultConfig())

	// Create the client
	client, err := newAPIClient(r.config)
	if err != nil {
		return fmt.Errorf("runner: %s", err)
	}
	r.client = client

	// Create the watcher
	watcher, err := newWatcher(r.config, client, r.once)
	if err != nil {
		return fmt.Errorf("runner: %s", err)
	}
	r.watcher = watcher

	r.data = make(map[string][]*dep.KeyPair)

	r.outStream = os.Stdout
	r.errStream = os.Stderr

	r.ErrCh = make(chan error)
	r.DoneCh = make(chan struct{})
	r.ExitCh = make(chan int, 1)

	return nil
}

// Restart the current process in the Runner by sending a SIGTERM. It is
// assumed that the process is set on the Runner!
func (r *Runner) killProcess() {
	// Kill the process
	exited := false

	if err := r.cmd.Process.Signal(syscall.SIGTERM); err == nil {
		// Wait a few seconds for it to exit
		killCh := make(chan struct{})
		go func() {
			defer close(killCh)
			r.cmd.Process.Wait()
		}()

		select {
		case <-killCh:
			exited = true
		case <-time.After(r.config.Timeout):
		}
	}

	// If we still haven't exited from a SIGKILL
	if !exited {
		r.cmd.Process.Kill()
	}

	r.cmd = nil
}

// newAPIClient creates a new API client from the given config and
func newAPIClient(config *Config) (*api.Client, error) {
	log.Printf("[INFO] (runner) creating consul/api client")

	consulConfig := api.DefaultConfig()

	if config.Consul != "" {
		log.Printf("[DEBUG] (runner) setting address to %s", config.Consul)
		consulConfig.Address = config.Consul
	}

	if config.Token != "" {
		log.Printf("[DEBUG] (runner) setting token to %s", config.Token)
		consulConfig.Token = config.Token
	}

	if config.SSL.Enabled {
		log.Printf("[DEBUG] (runner) enabling SSL")
		consulConfig.Scheme = "https"
	}

	if !config.SSL.Verify {
		log.Printf("[WARN] (runner) disabling SSL verification")
		consulConfig.HttpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	if config.Auth != nil {
		log.Printf("[DEBUG] (runner) setting basic auth")
		consulConfig.HttpAuth = &api.HttpBasicAuth{
			Username: config.Auth.Username,
			Password: config.Auth.Password,
		}
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// newWatcher creates a new watcher.
func newWatcher(config *Config, client *api.Client, once bool) (*watch.Watcher, error) {
	log.Printf("[INFO] (runner) creating Watcher")

	watcher, err := watch.NewWatcher(&watch.WatcherConfig{
		Client:   client,
		Once:     once,
		MaxStale: config.MaxStale,
		RetryFunc: func(current time.Duration) time.Duration {
			return config.Retry
		},
	})
	if err != nil {
		return nil, err
	}

	return watcher, err
}

// buildConfig iterates and merges all configuration files in a given directory.
// The config parameter will be modified and merged with subsequent configs
// found in the directory.
func buildConfig(config *Config, path string) error {
	log.Printf("[DEBUG] merging with config at %s", path)

	// Ensure the given filepath exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config: missing file/folder: %s", path)
	}

	// Check if a file was given or a path to a directory
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("config: error stating file: %s", err)
	}

	// Recursively parse directories, single load files
	if stat.Mode().IsDir() {
		// Ensure the given filepath has at least one config file
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return fmt.Errorf("config: error listing directory: %s", err)
		}
		if len(files) == 0 {
			return fmt.Errorf("config: must contain at least one configuration file")
		}

		// Potential bug: Walk does not follow symlinks!
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			// If WalkFunc had an error, just return it
			if err != nil {
				return err
			}

			// Do nothing for directories
			if info.IsDir() {
				return nil
			}

			// Parse and merge the config
			newConfig, err := ParseConfig(path)
			if err != nil {
				return err
			}
			config.Merge(newConfig)

			return nil
		})

		if err != nil {
			return fmt.Errorf("config: walk error: %s", err)
		}
	} else if stat.Mode().IsRegular() {
		newConfig, err := ParseConfig(path)
		if err != nil {
			return err
		}
		config.Merge(newConfig)
	} else {
		return fmt.Errorf("config: unknown filetype: %s", stat.Mode().String())
	}

	return nil
}
