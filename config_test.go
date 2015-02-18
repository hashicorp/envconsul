package main

import (
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
)

// Test that an empty config does nothing
func TestMerge_emptyConfig(t *testing.T) {
	consul := "consul.io:8500"
	config := &Config{Consul: consul}
	config.Merge(&Config{})

	if config.Consul != consul {
		t.Fatalf("expected %q to equal %q", config.Consul, consul)
	}
}

// Test that simple values are merged
func TestMerge_simpleConfig(t *testing.T) {
	config, newConsul := &Config{Consul: "consul.io:8500"}, "packer.io:7300"
	config.Merge(&Config{Consul: newConsul})

	if config.Consul != newConsul {
		t.Fatalf("expected %q to equal %q", config.Consul, newConsul)
	}
}

// Test that the flags for HTTPS are properly merged
func TestMerge_HttpsOptions(t *testing.T) {
	config := &Config{
		SSL: &SSL{
			Enabled: false,
			Verify:  false,
		},
	}
	otherConfig := &Config{
		SSL: &SSL{
			Enabled: true,
			Verify:  true,
		},
	}
	config.Merge(otherConfig)

	if config.SSL.Enabled != true {
		t.Errorf("expected enabled to be true")
	}

	if config.SSL.Verify != true {
		t.Errorf("expected SSL verify to be true")
	}

	config = &Config{
		SSL: &SSL{
			Enabled: true,
			Verify:  true,
		},
	}
	otherConfig = &Config{
		SSL: &SSL{
			Enabled: false,
			Verify:  false,
		},
	}
	config.Merge(otherConfig)

	if config.SSL.Enabled != false {
		t.Errorf("expected enabled to be false")
	}

	if config.SSL.Verify != false {
		t.Errorf("expected SSL verify to be false")
	}
}

func TestMerge_AuthOptions(t *testing.T) {
	config := &Config{
		Auth: &Auth{Username: "user", Password: "pass"},
	}
	otherConfig := &Config{
		Auth: &Auth{Username: "newUser", Password: ""},
	}
	config.Merge(otherConfig)

	if config.Auth.Username != "newUser" {
		t.Errorf("expected %q to be %q", config.Auth.Username, "newUser")
	}
}

func TestMerge_SyslogOptions(t *testing.T) {
	config := &Config{
		Syslog: &Syslog{Enabled: false, Facility: "LOCAL0"},
	}
	otherConfig := &Config{
		Syslog: &Syslog{Enabled: true, Facility: "LOCAL1"},
	}
	config.Merge(otherConfig)

	if config.Syslog.Enabled != true {
		t.Errorf("expected %t to be %t", config.Syslog.Enabled, true)
	}

	if config.Syslog.Facility != "LOCAL1" {
		t.Errorf("expected %q to be %q", config.Syslog.Facility, "LOCAL1")
	}
}

// Test that file read errors are propagated up
func TestParseConfig_readFileError(t *testing.T) {
	_, err := ParseConfig(path.Join(os.TempDir(), "config.json"))
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "no such file or directory"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that parser errors are propagated up
func TestParseConfig_parseFileError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    invalid file in here
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "syntax error"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that mapstructure errors are propagated up
func TestParseConfig_mapstructureError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = true
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "nconvertible type 'bool'"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

// Test that the config is parsed correctly
func TestParseConfig_correctValues(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = "nyc1.demo.consul.io"
    max_stale = "5s"
    token = "abcd1234"
    timeout = "5m"
    wait = "5s:10s"
    retry = "10s"
    log_level = "warn"

    auth {
    	enabled = true
    	username = "test"
    	password = "test"
    }

    ssl {
    	enabled = true
    	verify = false
    }

    syslog {
    	enabled = true
    	facility = "LOCAL5"
    }
  `), t)
	defer test.DeleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := &Config{
		Path:        configFile.Name(),
		Consul:      "nyc1.demo.consul.io",
		Token:       "abcd1234",
		MaxStale:    time.Second * 5,
		MaxStaleRaw: "5s",
		Auth: &Auth{
			Enabled:  true,
			Username: "test",
			Password: "test",
		},
		AuthRaw: []*Auth{
			&Auth{
				Enabled:  true,
				Username: "test",
				Password: "test",
			},
		},
		SSL: &SSL{
			Enabled: true,
			Verify:  false,
		},
		SSLRaw: []*SSL{
			&SSL{
				Enabled: true,
				Verify:  false,
			},
		},
		Syslog: &Syslog{
			Enabled:  true,
			Facility: "LOCAL5",
		},
		SyslogRaw: []*Syslog{
			&Syslog{
				Enabled:  true,
				Facility: "LOCAL5",
			},
		},
		Timeout:    5 * time.Minute,
		TimeoutRaw: "5m",
		Wait: &watch.Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		WaitRaw:  "5s:10s",
		Retry:    10 * time.Second,
		RetryRaw: "10s",
		LogLevel: "warn",
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v", expected, config)
	}
}

func TestParseConfig_parseRetryError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    retry = "bacon pants"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "retry invalid"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfig_parseWaitError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    wait = "not_valid:duration"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "wait invalid"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}
