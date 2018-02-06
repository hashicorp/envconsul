package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul-template/config"
)

// PrefixConfig is a wrapper around some common options for Consul and Vault
// prefixes.
type PrefixConfig struct {
	Format   *string `mapstructure:"format"`
	NoPrefix *bool   `mapstructure:"no_prefix"`
	Path     *string `mapstructure:"path"`
}

func ParsePrefixConfig(s string) (*PrefixConfig, error) {
	switch parts := strings.Split(s, "@"); len(parts) {
	case 1:
		path := strings.TrimPrefix(parts[0], "/")

		return &PrefixConfig{
			Path: config.String(path),
		}, nil
	case 2:
		path := strings.TrimPrefix(parts[0], "/")
		format := parts[1]

		return &PrefixConfig{
			Path: config.String(path),
			Format: config.String(format),
		}, nil
	default:
		return nil, fmt.Errorf("Wrong number of delimiters found when parsing prefix (%d, expected 1 at most)", len(parts))
	}
}

func DefaultPrefixConfig() *PrefixConfig {
	return &PrefixConfig{}
}

func (c *PrefixConfig) Copy() *PrefixConfig {
	if c == nil {
		return nil
	}

	var o PrefixConfig

	o.Format = c.Format

	o.NoPrefix = c.NoPrefix

	o.Path = c.Path

	return &o
}

func (c *PrefixConfig) Merge(o *PrefixConfig) *PrefixConfig {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.Format != nil {
		r.Format = o.Format
	}

	if o.NoPrefix != nil {
		r.NoPrefix = o.NoPrefix
	}

	if o.Path != nil {
		r.Path = o.Path
	}

	return r
}

func (c *PrefixConfig) Finalize() {
	if c.Format == nil {
		c.Format = config.String("")
	}

	if c.NoPrefix == nil {
		c.NoPrefix = config.Bool(false)
	}

	if c.Path == nil {
		c.Path = config.String("")
	}
}

func (c *PrefixConfig) GoString() string {
	if c == nil {
		return "(*PrefixConfig)(nil)"
	}

	return fmt.Sprintf("&PrefixConfig{"+
		"Format:%s, "+
		"NoPrefix:%s, "+
		"Path:%s"+
		"}",
		config.StringGoString(c.Format),
		config.BoolGoString(c.NoPrefix),
		config.StringGoString(c.Path),
	)
}

type PrefixConfigs []*PrefixConfig

func DefaultPrefixConfigs() *PrefixConfigs {
	return &PrefixConfigs{}
}

func (c *PrefixConfigs) Copy() *PrefixConfigs {
	if c == nil {
		return nil
	}

	o := make(PrefixConfigs, len(*c))
	for i, t := range *c {
		o[i] = t.Copy()
	}
	return &o
}

func (c *PrefixConfigs) Merge(o *PrefixConfigs) *PrefixConfigs {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	*r = append(*r, *o...)

	return r
}

func (c *PrefixConfigs) Finalize() {
	if c == nil {
		*c = *DefaultPrefixConfigs()
	}

	for _, t := range *c {
		t.Finalize()
	}
}

func (c *PrefixConfigs) GoString() string {
	if c == nil {
		return "(*PrefixConfigs)(nil)"
	}

	s := make([]string, len(*c))
	for i, t := range *c {
		s[i] = t.GoString()
	}

	return "{" + strings.Join(s, ", ") + "}"
}
