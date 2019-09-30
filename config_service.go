package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul-template/config"
)

type ServiceConfig struct {
	Query         *string `mapstructure:"query"`
	FormatId      *string `mapstructure:"format_id"`
	FormatName    *string `mapstructure:"format_name"`
	FormatAddress *string `mapstructure:"format_address"`
	FormatTag     *string `mapstructure:"format_tag"`
	FormatPort    *string `mapstructure:"format_port"`
}

func ParseServiceConfig(s string) (*ServiceConfig, error) {
	return &ServiceConfig{
		Query: config.String(s),
	}, nil
}

func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		FormatId:      config.String(""),
		FormatName:    config.String(""),
		FormatAddress: config.String(""),
		FormatTag:     config.String(""),
		FormatPort:    config.String(""),
	}
}

func (s *ServiceConfig) Copy() *ServiceConfig {
	if s == nil {
		return nil
	}
	return &ServiceConfig{
		Query:         s.Query,
		FormatId:      s.FormatId,
		FormatName:    s.FormatName,
		FormatAddress: s.FormatAddress,
		FormatTag:     s.FormatTag,
		FormatPort:    s.FormatPort,
	}
}

func (s *ServiceConfig) Merge(o *ServiceConfig) *ServiceConfig {
	if s == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return s.Copy()
	}

	r := s.Copy()

	if o.Query != nil {
		r.Query = o.Query
	}

	if o.FormatId != nil {
		r.FormatId = o.FormatId
	}

	if o.FormatName != nil {
		r.FormatName = o.FormatName
	}

	if o.FormatAddress != nil {
		r.FormatAddress = o.FormatAddress
	}

	if o.FormatTag != nil {
		r.FormatTag = o.FormatTag
	}

	if o.FormatPort != nil {
		r.FormatPort = o.FormatPort
	}

	return r
}

func (s *ServiceConfig) Finalize() {
	if s.Query == nil {
		s.Query = config.String("")
	}

	if s.FormatId == nil {
		s.FormatId = config.String("")
	}

	if s.FormatName == nil {
		s.FormatName = config.String("")
	}

	if s.FormatAddress == nil {
		s.FormatAddress = config.String("")
	}

	if s.FormatTag == nil {
		s.FormatTag = config.String("")
	}

	if s.FormatPort == nil {
		s.FormatPort = config.String("")
	}
}

func (s *ServiceConfig) GoString() string {
	if s == nil {
		return "(*ServiceConfig)(nil)"
	}

	return fmt.Sprintf("&ServiceConfig{"+
		"Query:%s, "+
		"FormatId:%s, "+
		"FormatName:%s, "+
		"FormatAddress:%s, "+
		"FormatTag:%s, "+
		"FormatPort:%s"+
		"}",
		config.StringGoString(s.Query),
		config.StringGoString(s.FormatId),
		config.StringGoString(s.FormatName),
		config.StringGoString(s.FormatAddress),
		config.StringGoString(s.FormatTag),
		config.StringGoString(s.FormatPort),
	)
}

type ServiceConfigs []*ServiceConfig

func DefaultServiceConfigs() *ServiceConfigs {
	return &ServiceConfigs{}
}

func (s *ServiceConfigs) LastSeviceConfig() *ServiceConfig {
	if s == nil {
		return nil
	}

	return []*ServiceConfig(*s)[len([]*ServiceConfig(*s))-1]
}

func (s *ServiceConfigs) Copy() *ServiceConfigs {
	if s == nil {
		return nil
	}

	o := make(ServiceConfigs, len(*s))
	for i, t := range *s {
		o[i] = t.Copy()
	}
	return &o
}

func (s *ServiceConfigs) Merge(o *ServiceConfigs) *ServiceConfigs {
	if s == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return s.Copy()
	}

	r := s.Copy()

	*r = append(*r, *o...)

	return r
}

func (s *ServiceConfigs) Finalize() {
	if s == nil {
		*s = *DefaultServiceConfigs()
	}

	for _, t := range *s {
		t.Finalize()
	}
}

func (s *ServiceConfigs) GoString() string {
	if s == nil {
		return "(*ServiceConfigs)(nil)"
	}

	ss := make([]string, len(*s))
	for i, t := range *s {
		ss[i] = t.GoString()
	}

	return "{" + strings.Join(ss, ", ") + "}"
}
