package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
)

// Config is used to configure Consul ENV
type Config struct {
	// Path is the path to this configuration file on disk. This value is not
	// read from disk by rather dynamically populated by the code so the Config
	// has a reference to the path to the file on disk that created it.
	Path string `json:"path" mapstructure:"-"`

	// Consul is the location of the Consul instance to query (may be an IP
	// address or FQDN) with port.
	Consul string `json:"consul" mapstructure:"consul"`

	// Token is the Consul API token.
	Token string `json:"-" mapstructure:"token"`

	// PrefixesOld is the list of key prefix dependencies. This is deprecated and
	// is only used for backwards compatability purposes. Please use "prefix" and
	// the Prefixes struct key instead.
	PrefixesOld []string `json:"prefixes" mapstructure:"prefixes"`

	// Prefixes is the list of all prefix dependencies (consul)
	// in merge order.
	Prefixes []*ConfigPrefix `json:"prefix" mapstructure:"prefix"`

	// Secrets is the list of all secret dependencies (vault)
	Secrets []*ConfigPrefix `json:"secret" mapstructure:"secret"`

	// Auth is the HTTP basic authentication for communicating with Consul.
	Auth *AuthConfig `json:"auth" mapstructure:"auth"`

	// Vault is the configuration for connecting to a vault server.
	Vault *VaultConfig `json:"vault" mapstructure:"vault"`

	// SSL indicates we should use a secure connection while talking to
	// Consul. This requires Consul to be configured to serve HTTPS.
	SSL *SSLConfig `json:"ssl" mapstructure:"ssl"`

	// Syslog is the configuration for syslog.
	Syslog *SyslogConfig `json:"syslog" mapstructure:"syslog"`

	// MaxStale is the maximum amount of time for staleness from Consul as given
	// by LastContact. If supplied, Consul Template will query all servers instead
	// of just the leader.
	MaxStale time.Duration `json:"max_stale" mapstructure:"max_stale"`

	// Retry is the duration of time to wait between Consul failures.
	Retry time.Duration `json:"retry" mapstructure:"retry"`

	// Sanitize converts any "bad" characters in key values to underscores
	Sanitize bool `json:"sanitize" mapstructure:"sanitize"`

	// Splay is the maximum time in seconds to wait before restarting the program,
	// from which a random value is chosen. This is designed to prevent the
	// "thundering herd" problem.
	Splay time.Duration `json:"splay" mapstructure:"splay"`

	// Upcase converts environment variables to uppercase
	Upcase bool `json:"upcase" mapstructure:"upcase"`

	// Seperator change the default separator
	Separator string `json:"separator" mapstructure:"separator`

	// Timeout is the amount of time to wait for the child process to restart
	// before killing and restarting.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// Wait is the quiescence timers.
	Wait *watch.Wait `json:"wait" mapstructure:"wait"`

	// KillSignal is the signal to send to the child process on kill
	KillSignal string `json:"kill_signal" mapstructure:"kill_signal"`

	// LogLevel is the level with which to log for this config.
	LogLevel string `mapstructure:"log_level"`

	// Pristine indicates that we want a clean environment only
	// composed of consul config variables, not inheriting from exising
	// environment
	Pristine bool `json:"pristine" mapstructure:"pristine"`

	// setKeys is the list of config keys that were set by the user.
	setKeys map[string]struct{}
}

// Merge merges the values in config into this config object. Values in the
// config object overwrite the values in c.
func (c *Config) Merge(config *Config) {
	if config.WasSet("path") {
		c.Path = config.Path
	}

	if config.WasSet("consul") {
		c.Consul = config.Consul
	}

	if config.WasSet("token") {
		c.Token = config.Token
	}

	if len(config.Prefixes) > 0 {
		if c.Prefixes == nil {
			c.Prefixes = make([]*ConfigPrefix, 0, 1)
		}
		for _, prefix := range config.Prefixes {
			c.Prefixes = append(c.Prefixes, prefix)
		}
	}

	if len(config.Secrets) > 0 {
		if c.Secrets == nil {
			c.Secrets = make([]*ConfigPrefix, 0, 1)
		}
		for _, prefix := range config.Secrets {
			c.Secrets = append(c.Secrets, prefix)
		}
	}

	if config.WasSet("auth") {
		if c.Auth == nil {
			c.Auth = &AuthConfig{}
		}
		if config.WasSet("auth.username") {
			c.Auth.Username = config.Auth.Username
			c.Auth.Enabled = true
		}
		if config.WasSet("auth.password") {
			c.Auth.Password = config.Auth.Password
			c.Auth.Enabled = true
		}
		if config.WasSet("auth.enabled") {
			c.Auth.Enabled = config.Auth.Enabled
		}
	}

	if config.WasSet("vault") {
		if c.Vault == nil {
			c.Vault = &VaultConfig{}
		}
		if config.WasSet("vault.address") {
			c.Vault.Address = config.Vault.Address
		}
		if config.WasSet("vault.token") {
			c.Vault.Token = config.Vault.Token
		}
		if config.WasSet("vault.renew") {
			c.Vault.Renew = config.Vault.Renew
		}
		if config.WasSet("vault.ssl") {
			if c.Vault.SSL == nil {
				c.Vault.SSL = &SSLConfig{}
			}
			if config.WasSet("vault.ssl.verify") {
				c.Vault.SSL.Verify = config.Vault.SSL.Verify
				c.Vault.SSL.Enabled = true
			}
			if config.WasSet("vault.ssl.cert") {
				c.Vault.SSL.Cert = config.Vault.SSL.Cert
				c.Vault.SSL.Enabled = true
			}
			if config.WasSet("vault.ssl.ca_cert") {
				c.Vault.SSL.CaCert = config.Vault.SSL.CaCert
				c.Vault.SSL.Enabled = true
			}
			if config.WasSet("vault.ssl.enabled") {
				c.Vault.SSL.Enabled = config.Vault.SSL.Enabled
			}
		}
	}

	if config.WasSet("ssl") {
		if c.SSL == nil {
			c.SSL = &SSLConfig{}
		}
		if config.WasSet("ssl.verify") {
			c.SSL.Verify = config.SSL.Verify
			c.SSL.Enabled = true
		}
		if config.WasSet("ssl.cert") {
			c.SSL.Cert = config.SSL.Cert
			c.SSL.Enabled = true
		}
		if config.WasSet("ssl.ca_cert") {
			c.SSL.CaCert = config.SSL.CaCert
			c.SSL.Enabled = true
		}
		if config.WasSet("ssl.enabled") {
			c.SSL.Enabled = config.SSL.Enabled
		}
	}

	if config.WasSet("syslog") {
		if c.Syslog == nil {
			c.Syslog = &SyslogConfig{}
		}
		if config.WasSet("syslog.facility") {
			c.Syslog.Facility = config.Syslog.Facility
			c.Syslog.Enabled = true
		}
		if config.WasSet("syslog.enabled") {
			c.Syslog.Enabled = config.Syslog.Enabled
		}
	}

	if config.WasSet("max_stale") {
		c.MaxStale = config.MaxStale
	}

	if config.WasSet("retry") {
		c.Retry = config.Retry
	}

	if config.WasSet("sanitize") {
		c.Sanitize = config.Sanitize
	}

    if config.WasSet("separator") {
    	c.Separator = config.Separator
    }

	if config.WasSet("splay") {
		c.Splay = config.Splay
	}

	if config.WasSet("upcase") {
		c.Upcase = config.Upcase
	}

	if config.WasSet("timeout") {
		c.Timeout = config.Timeout
	}

	if config.WasSet("wait") {
		c.Wait = &watch.Wait{
			Min: config.Wait.Min,
			Max: config.Wait.Max,
		}
	}

	if config.WasSet("log_level") {
		c.LogLevel = config.LogLevel
	}

	if config.WasSet("kill_signal") {
		c.KillSignal = config.KillSignal
	}

	if config.WasSet("pristine") {
		c.Pristine = config.Pristine
	}

	if c.setKeys == nil {
		c.setKeys = make(map[string]struct{})
	}
	for k, _ := range config.setKeys {
		if _, ok := c.setKeys[k]; !ok {
			c.setKeys[k] = struct{}{}
		}
	}
}

// WasSet determines if the given key was set in the config (as opposed to just
// having the default value).
func (c *Config) WasSet(key string) bool {
	if _, ok := c.setKeys[key]; ok {
		return true
	}
	return false
}

// set is a helper function for marking a key as set.
func (c *Config) set(key string) {
	if _, ok := c.setKeys[key]; !ok {
		c.setKeys[key] = struct{}{}
	}
}

// ParseConfig reads the configuration file at the given path and returns a new
// Config struct with the data populated.
func ParseConfig(path string) (*Config, error) {
	var errs *multierror.Error

	// Read the contents of the file
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config at %q: %s", path, err)
	}

	// Parse the file (could be HCL or JSON)
	var shadow interface{}
	if err := hcl.Decode(&shadow, string(contents)); err != nil {
		return nil, fmt.Errorf("error decoding config at %q: %s", path, err)
	}

	// Convert to a map and flatten the keys we want to flatten
	parsed, ok := shadow.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error converting config at %q", path)
	}
	flattenKeys(parsed, []string{"auth", "ssl", "syslog", "vault"})

	// Create a new, empty config
	config := new(Config)

	// Use mapstructure to populate the basic config fields
	metadata := new(mapstructure.Metadata)
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			watch.StringToWaitDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.StringToTimeDurationHookFunc(),
		),
		ErrorUnused: true,
		Metadata:    metadata,
		Result:      config,
	})
	if err != nil {
		errs = multierror.Append(errs, err)
		return nil, errs.ErrorOrNil()
	}
	if err := decoder.Decode(parsed); err != nil {
		errs = multierror.Append(errs, err)
		return nil, errs.ErrorOrNil()
	}

	// Store a reference to the path where this config was read from
	config.Path = path

	// Handle deprecations
	if len(config.PrefixesOld) > 0 {
		log.Printf(`[WARN] Specifying the key "prefixes" in the configuration is `+
			`no longer supported. Please specify each prefix individually using `+
			`the key "prefix" (config at %s)`, path)
		prefixes := make([]*ConfigPrefix, 0, len(config.PrefixesOld))
		for _, prefix := range config.PrefixesOld {
			prefixes = append(prefixes, &ConfigPrefix{
				Path: prefix,
			})
		}
		config.Prefixes = append(prefixes, config.Prefixes...)
		config.PrefixesOld = nil
	}

	// Update the list of set keys
	if config.setKeys == nil {
		config.setKeys = make(map[string]struct{})
	}
	for _, key := range metadata.Keys {
		if _, ok := config.setKeys[key]; !ok {
			config.setKeys[key] = struct{}{}
		}
	}
	config.setKeys["path"] = struct{}{}

	d := DefaultConfig()
	d.Merge(config)
	config = d

	return config, errs.ErrorOrNil()
}

// ConfigFromPath iterates and merges all configuration files in a given
// directory, returning the resulting config.
func ConfigFromPath(path string) (*Config, error) {
	// Ensure the given filepath exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config: missing file/folder: %s", path)
	}

	// Check if a file was given or a path to a directory
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("config: error stating file: %s", err)
	}

	// Recursively parse directories, single load files
	if stat.Mode().IsDir() {
		// Ensure the given filepath has at least one config file
		_, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("config: error listing directory: %s", err)
		}

		// Create a blank config to merge off of
		config := DefaultConfig()

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

			log.Printf("[DEBUG] (config) merging with %q", path)

			// Parse and merge the config
			newConfig, err := ParseConfig(path)
			if err != nil {
				return err
			}
			config.Merge(newConfig)

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("config: walk error: %s", err)
		}

		return config, nil
	} else if stat.Mode().IsRegular() {
		return ParseConfig(path)
	}

	return nil, fmt.Errorf("config: unknown filetype: %q", stat.Mode().String())
}

// DefaultConfig returns the default configuration struct.
func DefaultConfig() *Config {
	logLevel := os.Getenv("ENV_CONSUL_LOG")
	if logLevel == "" {
		logLevel = os.Getenv("ENVCONSUL_LOG")
		if logLevel == "" {
			logLevel = "WARN"
		}
	}

	config := &Config{
		Auth: &AuthConfig{
			Enabled: false,
		},
		Vault: &VaultConfig{
			Renew: true,
			SSL: &SSLConfig{
				Enabled: true,
				Verify:  true,
			},
		},
		SSL: &SSLConfig{
			Enabled: false,
			Verify:  true,
		},
		Syslog: &SyslogConfig{
			Enabled:  false,
			Facility: "LOCAL0",
		},
		Separator:  "_",
		Sanitize:   false,
		Upcase:     false,
		Timeout:    5 * time.Second,
		Retry:      5 * time.Second,
		MaxStale:   1 * time.Second,
		Wait:       &watch.Wait{},
		LogLevel:   logLevel,
		KillSignal: "SIGTERM",
		Pristine:   false,
		setKeys:    make(map[string]struct{}),
	}

	if v := os.Getenv("CONSUL_HTTP_ADDR"); v != "" {
		config.Consul = v
	}

	if v := os.Getenv("CONSUL_TOKEN"); v != "" {
		config.Token = v
	}

	if v := os.Getenv("VAULT_ADDR"); v != "" {
		config.Vault.Address = v
	}

	if v := os.Getenv("VAULT_CAPATH"); v != "" {
		config.Vault.SSL.Cert = v
	}

	if v := os.Getenv("VAULT_CACERT"); v != "" {
		config.Vault.SSL.CaCert = v
	}

	if v := os.Getenv("VAULT_SKIP_VERIFY"); v != "" {
		config.Vault.SSL.Verify = false
	}

	return config
}

// ConfigPrefix is a wrapper around some common options for Consul and Vault
// prefixes.
type ConfigPrefix struct {
	Path   string `json:"path" mapstructure:"path"`
	Format string `json:"format" mapstructure:"format"`
}

// AuthConfig is the HTTP basic authentication data.
type AuthConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
}

// String is the string representation of this authentication. If authentication
// is not enabled, this returns the empty string. The username and password will
// be separated by a colon.
func (a *AuthConfig) String() string {
	if !a.Enabled {
		return ""
	}

	if a.Password != "" {
		return fmt.Sprintf("%s:%s", a.Username, a.Password)
	}

	return a.Username
}

// VaultConfig is the configuration for connecting to a vault server.
type VaultConfig struct {
	Address string `json:"address,omitempty" mapstructure:"address"`
	Token   string `json:"-" mapstructure:"token"`
	Renew   bool   `json:"renew" mapstructure:"renew"`

	// SSL indicates we should use a secure connection while talking to Vault.
	SSL *SSLConfig `json:"ssl" mapstructure:"ssl"`
}

// SSLConfig is the configuration for SSL.
type SSLConfig struct {
	Enabled bool   `json:"enabled" mapstructure:"enabled"`
	Verify  bool   `json:"verify" mapstructure:"verify"`
	Cert    string `json:"cert,omitempty" mapstructure:"cert"`
	CaCert  string `json:"ca_cert,omitempty" mapstructure:"ca_cert"`
}

// SyslogConfig is the configuration for syslog.
type SyslogConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	Facility string `json:"facility" mapstructure:"facility"`
}

// flattenKeys is a function that takes a map[string]interface{} and recursively
// flattens any keys that are a []map[string]interface{} where the key is in the
// given list of keys.
func flattenKeys(m map[string]interface{}, keys []string) {
	keyMap := make(map[string]struct{})
	for _, key := range keys {
		keyMap[key] = struct{}{}
	}

	var flatten func(map[string]interface{})
	flatten = func(m map[string]interface{}) {
		for k, v := range m {
			if _, ok := keyMap[k]; !ok {
				continue
			}

			switch typed := v.(type) {
			case []map[string]interface{}:
				if len(typed) > 0 {
					last := typed[len(typed)-1]
					flatten(last)
					m[k] = last
				} else {
					m[k] = nil
				}
			case map[string]interface{}:
				flatten(typed)
				m[k] = typed
			default:
				m[k] = v
			}
		}
	}

	flatten(m)
}
