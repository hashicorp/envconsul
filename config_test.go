package main

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
)

func testConfig(contents string, t *testing.T) *Config {
	f, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Write([]byte(contents))
	if err != nil {
		t.Fatal(err)
	}

	config, err := ParseConfig(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestMerge_emptyConfig(t *testing.T) {
	config := DefaultConfig()
	config.Merge(&Config{})

	expected := DefaultConfig()
	if !reflect.DeepEqual(config, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config, expected)
	}
}

func TestMerge_topLevel(t *testing.T) {
	config1 := testConfig(`
		consul = "consul-1"
		token = "token-1"
		max_stale = "1s"
		retry = "1s"
		wait = "1s"
		log_level = "log_level-1"
	`, t)
	config2 := testConfig(`
		consul = "consul-2"
		token = "token-2"
		max_stale = "2s"
		retry = "2s"
		wait = "2s"
		log_level = "log_level-2"
	`, t)
	config1.Merge(config2)

	if !reflect.DeepEqual(config1, config2) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config1, config2)
	}
}

func TestMerge_auth(t *testing.T) {
	config := testConfig(`
		auth {
			enabled = true
			username = "1"
			password = "1"
		}
	`, t)
	config.Merge(testConfig(`
		auth {
			password = "2"
		}
	`, t))

	expected := &AuthConfig{
		Enabled:  true,
		Username: "1",
		Password: "2",
	}

	if !reflect.DeepEqual(config.Auth, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Auth, expected)
	}
}

func TestMerge_SSL(t *testing.T) {
	config := testConfig(`
		ssl {
			enabled = true
			verify = true
			cert = "1.pem"
			ca_cert = "ca-1.pem"
		}
	`, t)
	config.Merge(testConfig(`
		ssl {
			enabled = false
		}
	`, t))

	expected := &SSLConfig{
		Enabled: false,
		Verify:  true,
		Cert:    "1.pem",
		CaCert:  "ca-1.pem",
	}

	if !reflect.DeepEqual(config.SSL, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.SSL, expected)
	}
}

func TestMerge_syslog(t *testing.T) {
	config := testConfig(`
		syslog {
			enabled = true
			facility = "1"
		}
	`, t)
	config.Merge(testConfig(`
		syslog {
			facility = "2"
		}
	`, t))

	expected := &SyslogConfig{
		Enabled:  true,
		Facility: "2",
	}

	if !reflect.DeepEqual(config.Syslog, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Syslog, expected)
	}
}

func TestMerge_prefixes(t *testing.T) {
	global, err := dep.ParseStoreKeyPrefix("global/time")
	if err != nil {
		t.Fatal(err)
	}

	redis, err := dep.ParseStoreKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	config := testConfig(`
		prefixes = ["global/time"]
	`, t)
	config.Merge(testConfig(`
		prefixes = ["config/redis"]
	`, t))

	expected := []*dep.StoreKeyPrefix{global, redis}
	if !reflect.DeepEqual(config.Prefixes, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Prefixes, expected)
	}
}

func TestMerge_wait(t *testing.T) {
	config := testConfig(`
		wait = "1s:1s"
	`, t)
	config.Merge(testConfig(`
		wait = "2s:2s"
	`, t))

	expected := &watch.Wait{
		Min: 2 * time.Second,
		Max: 2 * time.Second,
	}

	if !reflect.DeepEqual(config.Wait, expected) {
		t.Errorf("expected \n\n%#v\n\n to be \n\n%#v\n\n", config.Wait, expected)
	}
}

func TestParseConfig_readFileError(t *testing.T) {
	_, err := ParseConfig(path.Join(os.TempDir(), "config.json"))
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "no such file or directory"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to include %q", err.Error(), expected)
	}
}

func TestParseConfig_parseFileError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    invalid file in here
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "syntax error"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestParseConfig_correctValues(t *testing.T) {
	global, err := dep.ParseStoreKeyPrefix("config/global")
	if err != nil {
		t.Fatal(err)
	}

	redis, err := dep.ParseStoreKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	configFile := test.CreateTempfile([]byte(`
		consul = "nyc1.demo.consul.io"
		max_stale = "5s"
		token = "abcd1234"
		wait = "5s:10s"
		retry = "10s"
		log_level = "warn"

		prefixes = ["config/global", "config/redis"]

		auth {
			enabled = true
			username = "test"
			password = "test"
		}

		ssl {
			enabled = true
			verify = false
			cert = "c1.pem"
			ca_cert = "c2.pem"
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
		Path:     configFile.Name(),
		Consul:   "nyc1.demo.consul.io",
		MaxStale: time.Second * 5,
		Upcase:   false,
		Sanitize: false,
		Timeout:  5 * time.Second,
		Auth: &AuthConfig{
			Enabled:  true,
			Username: "test",
			Password: "test",
		},
		SSL: &SSLConfig{
			Enabled: true,
			Verify:  false,
			Cert:    "c1.pem",
			CaCert:  "c2.pem",
		},
		Syslog: &SyslogConfig{
			Enabled:  true,
			Facility: "LOCAL5",
		},
		Token: "abcd1234",
		Wait: &watch.Wait{
			Min: time.Second * 5,
			Max: time.Second * 10,
		},
		Retry:      10 * time.Second,
		LogLevel:   "warn",
		Prefixes:   []*dep.StoreKeyPrefix{global, redis},
		KillSignal: "SIGTERM",
		setKeys:    config.setKeys,
	}

	if !reflect.DeepEqual(config, expected) {
		t.Fatalf("expected \n%#v\n to be \n%#v\n", config, expected)
	}

	// if !reflect.DeepEqual(config, expected) {
	// 	t.Fatalf("expected \n%#v\n to be \n%#v\n", config, expected)
	// }
}

func TestParseConfig_mapstructureError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    consul = true
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "unconvertible type 'bool'"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestParseConfig_extraKeys(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		fake_key = "nope"
		another_fake_key = "never"
	`), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error")
	}

	expected := "invalid keys: another_fake_key, fake_key"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to be %q", err.Error(), expected)
	}
}

func TestParseConfig_parseMaxStaleError(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
    max_stale = "bacon pants"
  `), t)
	defer test.DeleteTempfile(configFile, t)

	_, err := ParseConfig(configFile.Name())
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
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

	expectedErr := "time: invalid duration"
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

	expectedErr := "time: invalid duration"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error %q to contain %q", err.Error(), expectedErr)
	}
}

func TestConfigFromPath_singleFile(t *testing.T) {
	configFile := test.CreateTempfile([]byte(`
		consul = "127.0.0.1"
	`), t)
	defer test.DeleteTempfile(configFile, t)

	config, err := ConfigFromPath(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "127.0.0.1"
	if config.Consul != expected {
		t.Errorf("expected %q to be %q", config.Consul, expected)
	}
}

func TestConfigFromPath_NonExistentDirectory(t *testing.T) {
	// Create a directory and then delete it
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(configDir); err != nil {
		t.Fatal(err)
	}

	_, err = ConfigFromPath(configDir)
	if err == nil {
		t.Fatalf("expected error, but nothing was returned")
	}

	expected := "missing file/folder"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestConfigFromPath_EmptyDirectory(t *testing.T) {
	// Create a directory with no files
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(configDir)

	_, err = ConfigFromPath(configDir)
	if err != nil {
		t.Fatalf("empty directories are allowed")
	}
}

func TestConfigFromPath_BadConfigs(t *testing.T) {
	configDir, err := ioutil.TempDir("", "bad-configs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(configDir)

	configPath := filepath.Join(configDir, "config")
	err = ioutil.WriteFile(configPath, []byte(`
		totally not a valid config
	`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(configPath)

	_, err = ConfigFromPath(configDir)
	if err == nil {
		t.Fatalf("expected error, but nothing was returned")
	}

	expected := "error decoding config at"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestConfigFromPath_configDir(t *testing.T) {
	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	configFile1, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	config1 := []byte(`
		consul = "127.0.0.1:8500"
	`)
	_, err = configFile1.Write(config1)
	if err != nil {
		t.Fatal(err)
	}
	configFile2, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	config2 := []byte(`
		prefixes = ["config/global"]
	`)
	_, err = configFile2.Write(config2)
	if err != nil {
		t.Fatal(err)
	}

	config, err := ConfigFromPath(configDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedConfig := Config{
		Consul:   "127.0.0.1:8500",
		Prefixes: []*dep.StoreKeyPrefix{{Prefix: "global/time"}},
	}
	if expectedConfig.Consul != config.Consul {
		t.Fatalf("Config files failed to combine. Expected Consul to be %s but got %s", expectedConfig.Consul, config.Consul)
	}
}

func TestAuthString_disabled(t *testing.T) {
	a := &AuthConfig{Enabled: false}
	expected := ""
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabledNoPassword(t *testing.T) {
	a := &AuthConfig{Enabled: true, Username: "username"}
	expected := "username"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}

func TestAuthString_enabled(t *testing.T) {
	a := &AuthConfig{Enabled: true, Username: "username", Password: "password"}
	expected := "username:password"
	if a.String() != expected {
		t.Errorf("expected %q to be %q", a.String(), expected)
	}
}
