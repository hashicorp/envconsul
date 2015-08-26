package main

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/watch"
)

func TestParseFlags_consul(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-consul", "12.34.56.78",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "12.34.56.78"
	if config.Consul != expected {
		t.Errorf("expected %q to be %q", config.Consul, expected)
	}
	if !config.WasSet("consul") {
		t.Errorf("expected consul to be set")
	}
}

func TestParseFlags_token(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-token", "abcd1234",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "abcd1234"
	if config.Token != expected {
		t.Errorf("expected %q to be %q", config.Token, expected)
	}
	if !config.WasSet("token") {
		t.Errorf("expected token to be set")
	}
}

func TestParseFlags_prefix(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-prefix", "global",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected, err := dep.ParseStoreKeyPrefix("global")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config.Prefixes[0], expected) {
		t.Errorf("expected %#v to be %#v", config.Prefixes[0], expected)
	}
}

func TestParseFlags_authUsername(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-auth", "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Auth.Enabled != true {
		t.Errorf("expected auth to be enabled")
	}
	if !config.WasSet("auth.enabled") {
		t.Errorf("expected auth.enabled to be set")
	}

	expected := "test"
	if config.Auth.Username != expected {
		t.Errorf("expected %v to be %v", config.Auth.Username, expected)
	}
	if !config.WasSet("auth.username") {
		t.Errorf("expected auth.username to be set")
	}
}

func TestParseFlags_authUsernamePassword(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-auth", "test:test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if config.Auth.Enabled != true {
		t.Errorf("expected auth to be enabled")
	}
	if !config.WasSet("auth.enabled") {
		t.Errorf("expected auth.enabled to be set")
	}

	expected := "test"
	if config.Auth.Username != expected {
		t.Errorf("expected %v to be %v", config.Auth.Username, expected)
	}
	if !config.WasSet("auth.username") {
		t.Errorf("expected auth.username to be set")
	}
	if config.Auth.Password != expected {
		t.Errorf("expected %v to be %v", config.Auth.Password, expected)
	}
	if !config.WasSet("auth.password") {
		t.Errorf("expected auth.password to be set")
	}
}

func TestParseFlags_SSL(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.SSL.Enabled != expected {
		t.Errorf("expected %v to be %v", config.SSL.Enabled, expected)
	}
	if !config.WasSet("ssl.enabled") {
		t.Errorf("expected ssl.enabled to be set")
	}
}

func TestParseFlags_noSSL(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl=false",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := false
	if config.SSL.Enabled != expected {
		t.Errorf("expected %v to be %v", config.SSL.Enabled, expected)
	}
	if !config.WasSet("ssl.enabled") {
		t.Errorf("expected ssl.enabled to be set")
	}
}

func TestParseFlags_SSLVerify(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-verify",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.SSL.Verify != expected {
		t.Errorf("expected %v to be %v", config.SSL.Verify, expected)
	}
	if !config.WasSet("ssl.verify") {
		t.Errorf("expected ssl.verify to be set")
	}
}

func TestParseFlags_noSSLVerify(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-verify=false",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := false
	if config.SSL.Verify != expected {
		t.Errorf("expected %v to be %v", config.SSL.Verify, expected)
	}
	if !config.WasSet("ssl.verify") {
		t.Errorf("expected ssl.verify to be set")
	}
}

func TestParseFlags_SSLCert(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-cert", "/path/to/c1.pem",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/c1.pem"
	if config.SSL.Cert != expected {
		t.Errorf("expected %v to be %v", config.SSL.Cert, expected)
	}
	if !config.WasSet("ssl.cert") {
		t.Errorf("expected ssl.cert to be set")
	}
}

func TestParseFlags_SSLCaCert(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-ssl-ca-cert", "/path/to/c2.pem",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/c2.pem"
	if config.SSL.CaCert != expected {
		t.Errorf("expected %v to be %v", config.SSL.CaCert, expected)
	}
	if !config.WasSet("ssl.ca_cert") {
		t.Errorf("expected ssl.ca_cert to be set")
	}
}

func TestParseFlags_maxStale(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-max-stale", "10h",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := 10 * time.Hour
	if config.MaxStale != expected {
		t.Errorf("expected %q to be %q", config.MaxStale, expected)
	}
}

func TestParseFlags_syslog(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-syslog",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Syslog.Enabled != expected {
		t.Errorf("expected %v to be %v", config.Syslog.Enabled, expected)
	}
	if !config.WasSet("syslog.enabled") {
		t.Errorf("expected syslog.enabled to be set")
	}
}

func TestParseFlags_syslogFacility(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-syslog-facility", "LOCAL5",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "LOCAL5"
	if config.Syslog.Facility != expected {
		t.Errorf("expected %v to be %v", config.Syslog.Facility, expected)
	}
	if !config.WasSet("syslog.facility") {
		t.Errorf("expected syslog.facility to be set")
	}
}

func TestParseFlags_wait(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-wait", "10h:11h",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := &watch.Wait{
		Min: 10 * time.Hour,
		Max: 11 * time.Hour,
	}
	if !reflect.DeepEqual(config.Wait, expected) {
		t.Errorf("expected %v to be %v", config.Wait, expected)
	}
	if !config.WasSet("wait") {
		t.Errorf("expected wait to be set")
	}
}

func TestParseFlags_waitError(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, _, err := cli.parseFlags([]string{
		"-wait", "watermelon:bacon",
	})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "invalid value"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestParseFlags_retry(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-retry", "10h",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := 10 * time.Hour
	if config.Retry != expected {
		t.Errorf("expected %v to be %v", config.Retry, expected)
	}
	if !config.WasSet("retry") {
		t.Errorf("expected retry to be set")
	}
}

func TestParseFlags_sanitize(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-sanitize",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Sanitize != expected {
		t.Errorf("expected %v to be %v", config.Sanitize, expected)
	}
	if !config.WasSet("sanitize") {
		t.Errorf("expected sanitize to be set")
	}
}

func TestParseFlags_upcase(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-upcase",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Upcase != expected {
		t.Errorf("expected %v to be %v", config.Upcase, expected)
	}
	if !config.WasSet("upcase") {
		t.Errorf("expected upcase to be set")
	}
}

func TestParseFlags_config(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-config", "/path/to/file",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "/path/to/file"
	if config.Path != expected {
		t.Errorf("expected %v to be %v", config.Path, expected)
	}
	if !config.WasSet("path") {
		t.Errorf("expected path to be set")
	}
}

func TestParseFlags_kill_signal(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-kill-signal", "SIGHUP",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "SIGHUP"
	if config.KillSignal != expected {
		t.Errorf("expected %v to be %v", config.KillSignal, expected)
	}
	if !config.WasSet("kill_signal") {
		t.Errorf("expected kill_signal to be set")
	}
}

func TestParseFlags_logLevel(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-log-level", "debug",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := "debug"
	if config.LogLevel != expected {
		t.Errorf("expected %v to be %v", config.LogLevel, expected)
	}
	if !config.WasSet("log_level") {
		t.Errorf("expected log_level to be set")
	}
}

func TestParseFlags_once(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, once, _, err := cli.parseFlags([]string{
		"-once",
	})
	if err != nil {
		t.Fatal(err)
	}

	if once != true {
		t.Errorf("expected once to be true")
	}
}

func TestParseFlags_version(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, version, err := cli.parseFlags([]string{
		"-version",
	})
	if err != nil {
		t.Fatal(err)
	}

	if version != true {
		t.Errorf("expected version to be true")
	}
}

func TestParseFlags_pristine(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-pristine",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := true
	if config.Pristine != expected {
		t.Errorf("expected %v to be %v", config.Pristine, expected)
	}
	if !config.WasSet("pristine") {
		t.Errorf("expected pristine to be set")
	}
}

func TestParseFlags_v(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, version, err := cli.parseFlags([]string{
		"-v",
	})
	if err != nil {
		t.Fatal(err)
	}

	if version != true {
		t.Errorf("expected version to be true")
	}
}

func TestParseFlags_errors(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, _, _, _, err := cli.parseFlags([]string{
		"-totally", "-not", "-valid",
	})

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}
}

func TestRun_errors(t *testing.T) {
	buf := new(bytes.Buffer)

	// Returns the right exit code if no command is given
	cli := NewCLI(ioutil.Discard, buf)
	if code := cli.Run([]string{"envconsul"}); code != ExitCodeUsageError {
		t.Fatalf("expected %d, got: %d", ExitCodeUsageError, code)
	}

	// Output reflects the returned error
	out := buf.String()
	if !strings.Contains(out, ErrMissingCommand.Error()) {
		t.Fatalf("expected to find %q, got: %q", ErrMissingCommand.Error(), out)
	}
}
