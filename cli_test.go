package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"syscall"
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

	expected := "test"
	if config.Auth.Username != expected {
		t.Errorf("expected %v to be %v", config.Auth.Username, expected)
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

	expected := "test"
	if config.Auth.Username != expected {
		t.Errorf("expected %v to be %v", config.Auth.Username, expected)
	}
	if config.Auth.Password != expected {
		t.Errorf("expected %v to be %v", config.Auth.Password, expected)
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
}

func TestParseFlags_prefixes(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-prefix", "config/global", "-prefix", "config/redis",
	})
	if err != nil {
		t.Fatal(err)
	}

	globalDep, err := dep.ParseStoreKeyPrefix("config/global")
	if err != nil {
		t.Fatal(err)
	}

	redisDep, err := dep.ParseStoreKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	expected := []*dep.StoreKeyPrefix{globalDep, redisDep}
	if !reflect.DeepEqual(config.Prefixes, expected) {
		t.Errorf("expected %v to be %v", config.Prefixes, expected)
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
		t.Errorf("expected %t to be %t", config.Sanitize, expected)
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
		t.Errorf("expected %t to be %t", config.Upcase, expected)
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
}

func TestParseFlags_args(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	_, args, _, _, err := cli.parseFlags([]string{
		"redis/config", "/etc/init.d/redis restart",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"redis/config", "/etc/init.d/redis restart"}
	if !reflect.DeepEqual(args, expected) {
		t.Errorf("expected %v to be %v", args, expected)
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

func TestRun_printsErrors(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := NewCLI(outStream, errStream)
	args := strings.Split("envconsul -bacon delicious", " ")

	status := cli.Run(args)
	defer cli.stop()

	if status == ExitCodeOK {
		t.Fatal("expected not OK exit code")
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

func TestRun_versionFlag(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := NewCLI(outStream, errStream)
	args := strings.Split("envconsul -version", " ")

	status := cli.Run(args)
	defer cli.stop()

	if status != ExitCodeOK {
		t.Errorf("expected %q to eq %q", status, ExitCodeOK)
	}

	expected := fmt.Sprintf("envconsul v%s", Version)
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

func TestRun_parseError(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := NewCLI(outStream, errStream)
	args := strings.Split("envconsul -bacon delicious", " ")

	status := cli.Run(args)
	defer cli.stop()

	if status != ExitCodeParseFlagsError {
		t.Errorf("expected %q to eq %q", status, ExitCodeParseFlagsError)
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Fatalf("expected %q to contain %q", errStream.String(), expected)
	}
}

func TestRun_onceFlag(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := NewCLI(outStream, errStream)

	command := "envconsul -consul demo.consul.io -once -prefix global/time env"
	args := strings.Split(command, " ")

	ch := make(chan int, 1)

	go func() {
		ch <- cli.Run(args)
	}()
	defer cli.stop()

	select {
	case status := <-ch:
		if status != ExitCodeOK {
			t.Errorf("expected %d to eq %d", status, ExitCodeOK)
			t.Errorf("stderr: %s", errStream.String())
		}
	case <-time.After(5 * time.Second):
		t.Errorf("expected data, but nothing was returned")
	}
}

func TestParseFlags_killsigFlag(t *testing.T) {
	cli := NewCLI(ioutil.Discard, ioutil.Discard)
	config, _, _, _, err := cli.parseFlags([]string{
		"-killsig", "SIGHUP",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := syscall.SIGHUP
	if config.KillSig != expected {
		t.Errorf("expected %q to be %q", config.KillSig, expected)
	}
}
