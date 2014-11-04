package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/hashicorp/consul-template/util"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
)

// Config is used to configure Consul ENV
type Config struct {
	// Path is the path to this configuration file on disk. This value is not
	// read from disk by rather dynamically populated by the code so the Config
	// has a reference to the path to the file on disk that created it.
	Path string `mapstructure:"-"`

	// Consul is the location of the Consul instance to query (may be an IP
	// address or FQDN) with port.
	Consul string `mapstructure:"consul"`

	// Sanitize converts any "bad" characters in key values to underscores
	Sanitize bool `mapstructure:"sanitize"`

	Timeout    time.Duration `mapstructure:"-"`
	TimeoutRaw string        `mapstructure:"timeout" json:""`

	// Token is the Consul API token.
	Token string `mapstructure:"token"`

	// Upcase converts environment variables to uppercase
	Upcase bool `mapstructure:"upcase"`

	// Wait
	Wait    *util.Wait `mapstructure:"-"`
	WaitRaw string     `mapstructure:"wait" json:""`
}

// Merge merges the values in config into this config object. Values in the
// config object overwrite the values in c.
func (c *Config) Merge(config *Config) {
	if config.Consul != "" {
		c.Consul = config.Consul
	}

	if config.Sanitize {
		c.Sanitize = config.Sanitize
	}

	if config.Timeout != 0 {
		c.Timeout = config.Timeout
	}

	if config.Token != "" {
		c.Token = config.Token
	}

	if config.Upcase {
		c.Upcase = config.Upcase
	}

	if config.Wait != nil {
		c.Wait = &util.Wait{
			Min: config.Wait.Min,
			Max: config.Wait.Max,
		}
		c.WaitRaw = config.WaitRaw
	}
}

// GoString returns the detailed format of this object
func (c *Config) GoString() string {
	return fmt.Sprintf("*%#v", *c)
}

// ParseConfig reads the configuration file at the given path and returns a new
// Config struct with the data populated.
func ParseConfig(path string) (*Config, error) {
	// Read the contents of the file
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the file (could be HCL or JSON)
	var parsed interface{}
	if err := hcl.Decode(&parsed, string(contents)); err != nil {
		return nil, err
	}

	// Create a new, empty config
	config := &Config{}

	// Use mapstructure to populate the basic config fields
	if err := mapstructure.Decode(parsed, config); err != nil {
		return nil, err
	}

	// Store a reference to the path where this config was read from
	config.Path = path

	// Parse the Wait component
	if raw := config.WaitRaw; raw != "" {
		wait, err := util.ParseWait(raw)

		if err == nil {
			config.Wait = wait
		} else {
			return nil, fmt.Errorf("Wait invalid: %v", err)
		}
	}

	return config, nil
}
