package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul-template/config"
)

// KeyFormat wraps configuration for a particular key in secrets set.
//
// Name is the name of the key
// Format is per-key format override, just like PrefixConfig.Format
type KeyFormat struct {
	Format *string `mapstructure:"format"`
	Name   *string `mapstructure:"name"`
}

func (f *KeyFormat) Copy() *KeyFormat {
	if f == nil {
		return nil
	}

	r := KeyFormat{}
	r.Name = f.Name
	r.Format = f.Format

	return &r
}

// KeyFormats holds a list of per-key override configurations.
type KeyFormats []*KeyFormat

func (f *KeyFormats) Copy() *KeyFormats {
	if f == nil {
		return nil
	}

	r := KeyFormats{}
	for _, v := range *f {
		r = append(r, v.Copy())
	}

	return &r
}

// PrefixConfig is a wrapper around some common options for Consul and Vault
// prefixes.
type PrefixConfig struct {
	Format   *string     `mapstructure:"format"`
	NoPrefix *bool       `mapstructure:"no_prefix"`
	Path     *string     `mapstructure:"path"`
	Keys     *KeyFormats `mapstructure:"key"`
}

func ParsePrefixConfig(s string) (*PrefixConfig, error) {
	s = strings.TrimPrefix(s, "/")
	return &PrefixConfig{
		Path: config.String(s),
	}, nil
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

	if c.Keys != nil {
		o.Keys = c.Keys.Copy()
	}

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

	if o.Keys != nil {
		r.Keys = o.Keys.Copy()
	}

	return r
}

func (c *PrefixConfig) Finalize() {
	if c.Format == nil {
		c.Format = config.String("")
	}

	if c.NoPrefix == nil {
		// Do not set a default value to allow differing defaults for Vault and Consul.
		// Vault secrets include prefix by default while Consul keys exclude it.
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

	// entries without a path are invalid and ignored
	confs := *c
	i := 0
	for _, t := range confs {
		if config.StringVal(t.Path) == "" {
			confs = append(confs[:i], confs[i+1:]...)
			continue
		}
		t.Finalize()
		i++
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
