package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul-template/config"
)

// PristineExcept is *TODO*
type PristineExceptConfig struct {
	Name *string `mapstructure:"name"`
}

func ParsePristineExceptConfig(s string) (*PristineExceptConfig, error) {
	s = strings.ToUpper(s)
	return &PristineExceptConfig{
		Name: config.String(s),
	}, nil
}

func DefaultPristineExceptConfig() *PristineExceptConfig {
	return &PristineExceptConfig{}
}

func (c *PristineExceptConfig) Copy() *PristineExceptConfig {
	if c == nil {
		return nil
	}

	var o PristineExceptConfig

	o.Name = c.Name

	return &o
}

func (c *PristineExceptConfig) Merge(o *PristineExceptConfig) *PristineExceptConfig {
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

	if o.Name != nil {
		r.Name = o.Name
	}

	return r
}

func (c *PristineExceptConfig) Finalize() {
	if c.Name == nil {
		c.Name = config.String("")
	}
}

func (c *PristineExceptConfig) GoString() string {
	if c == nil {
		return "(*PristineExceptConfig)(nil)"
	}

	return fmt.Sprintf("&PristineExceptConfig{"+
		"Name:%s, "+
		"}",
		config.StringGoString(c.Name),
	)
}

type PristineExceptConfigs []*PristineExceptConfig

func DefaultPristineExceptConfigs() *PristineExceptConfigs {
	return &PristineExceptConfigs{}
}

func (c *PristineExceptConfigs) Copy() *PristineExceptConfigs {
	if c == nil {
		return nil
	}

	o := make(PristineExceptConfigs, len(*c))
	for i, t := range *c {
		o[i] = t.Copy()
	}
	return &o
}

func (c *PristineExceptConfigs) Merge(o *PristineExceptConfigs) *PristineExceptConfigs {
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

func (c *PristineExceptConfigs) Finalize() {
	if c == nil {
		*c = *DefaultPristineExceptConfigs()
	}

	for _, t := range *c {
		t.Finalize()
	}
}

func (c *PristineExceptConfigs) GoString() string {
	if c == nil {
		return "(*PristineExceptConfigs)(nil)"
	}

	s := make([]string, len(*c))
	for i, t := range *c {
		s[i] = t.GoString()
	}

	return "{" + strings.Join(s, ", ") + "}"
}
