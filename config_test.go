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
    consul = "demo.consul.io"
    token = "abcd1234"
    wait = "5s:10s"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	config, err := ParseConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := &Config{
		Path:   configFile.Name(),
		Consul: "demo.consul.io",
		Token:  "abcd1234",
		Wait: &watch.Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		WaitRaw: "5s:10s",
	}
	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v", expected, config)
	}
}

// Test that ParseWait errors are propagated up
func TestParseConfig_parseWaitError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    wait = "not_valid:duration"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "invalid duration not_valid"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}
