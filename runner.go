package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/watch"
	consulapi "github.com/hashicorp/consul/api"
	vaultapi "github.com/hashicorp/vault/api"
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

	// once indicates the runner should get data exactly one time and then stop.
	once bool

	// minTimer and maxTimer are used for quiescence.
	minTimer, maxTimer <-chan time.Time

	// outStream and errStream are the io.Writer streams where the runner will
	// write information.
	outStream, errStream io.Writer

	// watcher is the watcher this runner is using.
	watcher *watch.Watcher

	// dependencies is the list of dependencies for this runner.
	dependencies []dep.Dependency

	// configPrefixMap is a map of a dependency's hashcode back to the config
	// prefix that created it.
	configPrefixMap map[string]*ConfigPrefix

	// data is the latest representation of the data from Consul.
	data map[string]interface{}

	// env is the last compiled environment.
	env map[string]string

	// command is the string of the command to run. cmd is the last known instance
	// of the running command.
	command []string
	cmd     *exec.Cmd

	// killSignal is the signal to send to kill the process.
	killSignal os.Signal
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

	// Add each dependency to the watcher
	for _, d := range r.dependencies {
		r.watcher.Add(d)
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
			if r.once {
				r.ErrCh <- err
				return
			}
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
	r.Lock()
	defer r.Unlock()

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
	r.data[d.HashCode()] = data
}

// Signal sends a signal to the child process, if it exists. Any errors that
// occur are returned.
func (r *Runner) Signal(sig os.Signal) error {
	r.Lock()
	defer r.Unlock()

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
	r.Lock()
	defer r.Unlock()

	log.Printf("[INFO] (runner) running")

	env := make(map[string]string)

	// Iterate over each dependency and pull out its data. If any dependencies do
	// not have data yet, this function will immediately return because we cannot
	// safely continue until all dependencies have received data at least once.
	//
	// We iterate over the list of config prefixes so that order is maintained,
	// since order in a map is not deterministic.
	for _, d := range r.dependencies {
		data, ok := r.data[d.HashCode()]
		if !ok {
			log.Printf("[INFO] (runner) missing data for %s", d.Display())
			return nil, nil
		}

		switch typed := d.(type) {
		case *dep.StoreKeyPrefix:
			r.appendPrefixes(env, typed, data)
		case *dep.VaultSecret:
			r.appendSecrets(env, typed, data)
		default:
			return nil, fmt.Errorf("unknown dependency type %T", typed)
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
	var cmdEnv []string

	if r.config.Pristine {
		cmdEnv = make([]string, 0)
	} else {
		processEnv := os.Environ()
		cmdEnv = make([]string, len(processEnv), len(r.env)+len(processEnv))
		copy(cmdEnv, processEnv)
	}
	for k, v := range r.env {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}

	// Create the command
	log.Printf("[INFO] (runner) running command %s %s", r.command[0], strings.Join(r.command[1:], " "))
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

func applyTemplate(contents, key string) (string, error) {
	funcs := template.FuncMap{
		"key": func() (string, error) {
			return key, nil
		},
		"stripped_key": func() (string, error) {
			i := strings.Index(key, "_")
			return key[i+1:], nil
		},
	}

	tmpl, err := template.New("filter").Funcs(funcs).Parse(contents)
	if err != nil {
		return "", nil
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, nil); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r *Runner) appendPrefixes(
	env map[string]string, d *dep.StoreKeyPrefix, data interface{}) error {
	var err error

	typed, ok := data.([]*dep.KeyPair)
	if !ok {
		return fmt.Errorf("error converting to keypair %s", d.Display())
	}

	// Get the ConfigPrefix so we can get configuration from it.
	cp := r.configPrefixMap[d.HashCode()]

	// For each pair, update the environment hash. Subsequent runs could
	// overwrite an existing key.
	for _, pair := range typed {
		key, value := pair.Key, string(pair.Value)

		// It is not possible to have an environment variable that is blank, but
		// it is possible to have an environment variable _value_ that is blank.
		if strings.TrimSpace(key) == "" {
			continue
		}

		// If the user specified a custom format, apply that here.
		if cp.Format != "" {
			key, err = applyTemplate(cp.Format, key)
			if err != nil {
				return err
			}
		}

		if r.config.Sanitize {
			key = InvalidRegexp.ReplaceAllString(key, "_")
		}

		if r.config.Separator != "_" {
				key = regexp.MustCompile(`[_/]`).ReplaceAllString(key, r.config.Separator)
		}

		if r.config.Upcase {
			key = strings.ToUpper(key)
		}

		if current, ok := env[key]; ok {
			log.Printf("[DEBUG] (runner) overwriting %s=%q (was %q) from %s",
				key, value, current, d.Display())
			env[key] = value
		} else {
			log.Printf("[DEBUG] (runner) setting %s=%q from %s",
				key, value, d.Display())
			env[key] = value
		}
	}

	return nil
}

func (r *Runner) appendSecrets(
	env map[string]string, d *dep.VaultSecret, data interface{}) error {
	var err error

	typed, ok := data.(*dep.Secret)
	if !ok {
		return fmt.Errorf("error converting to secret %s", d.Display())
	}

	// Get the ConfigPrefix so we can get configuration from it.
	cp := r.configPrefixMap[d.HashCode()]

	for key, value := range typed.Data {
		// Ignore any keys that are empty (not sure if this is even possible in
		// Vault, but I play defense).
		if strings.TrimSpace(key) == "" {
			continue
		}

        path := InvalidRegexp.ReplaceAllString(d.Path, "_")
		
		// Prefix the key value with the path value.
		if( key == "value") {
			key = path
		} else {
			key = fmt.Sprintf("%s_%s", path, key)
		}

		// If the user specified a custom format, apply that here.
		if cp.Format != "" {
			key, err = applyTemplate(cp.Format, key)
			if err != nil {
				return err
			}
		}

		if r.config.Sanitize {
			key = InvalidRegexp.ReplaceAllString(key, "_")
		}

		if r.config.Separator != "_" {
			key = regexp.MustCompile(`[_/]`).ReplaceAllString(key, r.config.Separator)
		}

		if r.config.Upcase {
			key = strings.ToUpper(key)
		}

		if current, ok := env[key]; ok {
			log.Printf("[DEBUG] (runner) overwriting %s=%q (was %q) from %s",
				key, value, current, d.Display())
			env[key] = value.(string)
		} else {
			log.Printf("[DEBUG] (runner) setting %s=%q from %s",
				key, value, d.Display())
			env[key] = value.(string)
		}
	}

	return nil
}

// init creates the Runner's underlying data structures and returns an error if
// any problems occur.
func (r *Runner) init() error {
	// Ensure we have defaults
	config := DefaultConfig()
	config.Merge(r.config)
	r.config = config

	// Print the final config for debugging
	result, err := json.MarshalIndent(r.config, "", "  ")
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] (runner) final config (tokens suppressed):\n\n%s\n\n",
		result)

	// Setup the kill signal
	signal, ok := SignalLookup[r.config.KillSignal]
	if !ok {
		valid := make([]string, 0, len(SignalLookup))
		for k, _ := range SignalLookup {
			valid = append(valid, k)
		}
		sort.Strings(valid)
		return fmt.Errorf("runner: unknown signal %q - valid signals are %q",
			r.config.KillSignal, valid)
	}
	r.killSignal = signal

	// Create the clientset
	clients, err := newClientSet(r.config)
	if err != nil {
		return fmt.Errorf("runner: %s", err)
	}

	// Create the watcher
	watcher, err := newWatcher(r.config, clients, r.once)
	if err != nil {
		return fmt.Errorf("runner: %s", err)
	}
	r.watcher = watcher

	r.data = make(map[string]interface{})
	r.configPrefixMap = make(map[string]*ConfigPrefix)

	r.outStream = os.Stdout
	r.errStream = os.Stderr

	r.ErrCh = make(chan error)
	r.DoneCh = make(chan struct{})
	r.ExitCh = make(chan int, 1)

	// Parse and add consul dependencies
	for _, p := range r.config.Prefixes {
		d, err := dep.ParseStoreKeyPrefix(p.Path)
		if err != nil {
			return err
		}
		r.dependencies = append(r.dependencies, d)
		r.configPrefixMap[d.HashCode()] = p
	}

	// Parse and add vault dependencies - it is important that this come after
	// consul, because consul should never be permitted to overwrite values from
	// vault; that would expose a security hole since access to consul is
	// typically less controlled than access to vault.
	for _, s := range r.config.Secrets {
		log.Printf("looking at vault %s", s.Path)
		d, err := dep.ParseVaultSecret(s.Path)
		if err != nil {
			return err
		}
		r.dependencies = append(r.dependencies, d)
		r.configPrefixMap[d.HashCode()] = s
	}

	return nil
}

// Restart the current process in the Runner by sending a SIGTERM. It is
// assumed that the process is set on the Runner! It is the caller's
// responsibility to lock the runner.
func (r *Runner) killProcess() {
	// Kill the process
	exited := false

	// If a splay value was given, sleep for a random amount of time up to the
	// splay.
	if r.config.Splay > 0 {
		nanoseconds := r.config.Splay.Nanoseconds()
		offset := rand.Int63n(nanoseconds)

		log.Printf("[INFO] (runner) waiting %.2fs for random splay",
			time.Duration(offset).Seconds())

		select {
		case <-time.After(time.Duration(offset)):
		case <-r.ExitCh:
		}
	}

	if err := r.cmd.Process.Signal(r.killSignal); err == nil {
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

// newClientSet creates a new client set from the given config.
func newClientSet(config *Config) (*dep.ClientSet, error) {
	clients := dep.NewClientSet()

	consul, err := newConsulClient(config)
	if err != nil {
		return nil, err
	}
	if err := clients.Add(consul); err != nil {
		return nil, err
	}

	vault, err := newVaultClient(config)
	if err != nil {
		return nil, err
	}
	if err := clients.Add(vault); err != nil {
		return nil, err
	}

	return clients, nil
}

// newConsulClient creates a new API client from the given config and
func newConsulClient(config *Config) (*consulapi.Client, error) {
	log.Printf("[INFO] (runner) creating consul/api client")

	consulConfig := consulapi.DefaultConfig()

	if config.Consul != "" {
		log.Printf("[DEBUG] (runner) setting consul address to %s", config.Consul)
		consulConfig.Address = config.Consul
	}

	if config.Token != "" {
		log.Printf("[DEBUG] (runner) setting consul token")
		consulConfig.Token = config.Token
	}

	if config.SSL.Enabled {
		log.Printf("[DEBUG] (runner) enabling consul SSL")
		consulConfig.Scheme = "https"

		tlsConfig := &tls.Config{}

		if config.SSL.Cert != "" {
			cert, err := tls.LoadX509KeyPair(config.SSL.Cert, config.SSL.Cert)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		if config.SSL.CaCert != "" {
			cacert, err := ioutil.ReadFile(config.SSL.CaCert)
			if err != nil {
				return nil, err
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(cacert)

			tlsConfig.RootCAs = caCertPool
		}
		tlsConfig.BuildNameToCertificate()

		if !config.SSL.Verify {
			log.Printf("[WARN] (runner) disabling consul SSL verification")
			tlsConfig.InsecureSkipVerify = true
		}
		consulConfig.HttpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	if config.Auth.Enabled {
		log.Printf("[DEBUG] (runner) setting basic auth")
		consulConfig.HttpAuth = &consulapi.HttpBasicAuth{
			Username: config.Auth.Username,
			Password: config.Auth.Password,
		}
	}

	client, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// newVaultClient creates a new client for connecting to vault.
func newVaultClient(config *Config) (*vaultapi.Client, error) {
	log.Printf("[INFO] (runner) creating vault/api client")

	vaultConfig := vaultapi.DefaultConfig()

	if config.Vault.Address != "" {
		log.Printf("[DEBUG] (runner) setting vault address to %s", config.Vault.Address)
		vaultConfig.Address = config.Vault.Address
	}

	if config.Vault.SSL.Enabled {
		log.Printf("[DEBUG] (runner) enabling vault SSL")
		tlsConfig := &tls.Config{}

		if config.Vault.SSL.Cert != "" {
			cert, err := tls.LoadX509KeyPair(config.Vault.SSL.Cert, config.Vault.SSL.Cert)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		if config.Vault.SSL.CaCert != "" {
			cacert, err := ioutil.ReadFile(config.Vault.SSL.CaCert)
			if err != nil {
				return nil, err
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(cacert)

			tlsConfig.RootCAs = caCertPool
		}
		tlsConfig.BuildNameToCertificate()

		if !config.Vault.SSL.Verify {
			log.Printf("[WARN] (runner) disabling vault SSL verification")
			tlsConfig.InsecureSkipVerify = true
		}

		vaultConfig.HttpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	client, err := vaultapi.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}

	if config.Vault.Token != "" {
		log.Printf("[DEBUG] (runner) setting vault token")
		client.SetToken(config.Vault.Token)
	}

	return client, nil
}

// newWatcher creates a new watcher.
func newWatcher(config *Config, clients *dep.ClientSet, once bool) (*watch.Watcher, error) {
	log.Printf("[INFO] (runner) creating Watcher")

	watcher, err := watch.NewWatcher(&watch.WatcherConfig{
		Clients:  clients,
		Once:     once,
		MaxStale: config.MaxStale,
		RetryFunc: func(current time.Duration) time.Duration {
			return config.Retry
		},
		RenewVault: config.Vault.Token != "" && config.Vault.Renew,
	})
	if err != nil {
		return nil, err
	}

	return watcher, err
}
