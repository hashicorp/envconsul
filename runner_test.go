package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/dependency"
)

func TestNewRunner_noPrefix(t *testing.T) {
	_, err := NewRunner("", nil, nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "missing prefix"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestNewRunner_noConfig(t *testing.T) {
	_, err := NewRunner("foo/bar", nil, nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "missing config"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestNewRunner_noCommand(t *testing.T) {
	_, err := NewRunner("foo/bar", &Config{}, nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "missing command"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestNewRunner_parseKeyPrefixError(t *testing.T) {
	_, err := NewRunner("!foo", &Config{}, []string{"env"})
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "invalid key prefix dependency format"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q to include %q", err.Error(), expected)
	}
}

func TestNewRunner_parsesRunner(t *testing.T) {
	config, command := &Config{}, []string{"env"}
	prefix, err := dependency.ParseStoreKeyPrefix("foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	runner, err := NewRunner("foo/bar", config, command)
	if err != nil {
		t.Fatal(err)
	}

	expected := &Runner{
		Prefix:    prefix,
		Command:   command,
		config:    config,
		outStream: os.Stdout,
		errStream: os.Stderr,
	}

	if !reflect.DeepEqual(runner, expected) {
		t.Errorf("expected \n%#v\n to include \n%#v\n", runner, expected)
	}
}

func TestRunner_dependencies(t *testing.T) {
	prefix, err := dependency.ParseStoreKeyPrefix("foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	runner, err := NewRunner("foo/bar", &Config{}, []string{"env"})
	if err != nil {
		t.Fatal(err)
	}

	expected := []dependency.Dependency{prefix}
	if !reflect.DeepEqual(runner.Dependencies(), expected) {
		t.Errorf("expected \n%#v\n to include \n%#v\n", runner, expected)
	}
}

func TestRunner_receiveSetsData(t *testing.T) {
	runner, err := NewRunner("foo/bar", &Config{}, []string{"env"})
	if err != nil {
		t.Fatal(err)
	}

	pair := []*dependency.KeyPair{&dependency.KeyPair{Path: "foo/bar"}}
	runner.Receive(pair)

	if !reflect.DeepEqual(runner.data, pair) {
		t.Errorf("expected \n%#v\n to include \n%#v\n", runner.data, pair)
	}
}

func TestRunner_waitWaits(t *testing.T) {
	runner, err := NewRunner("foo/bar", &Config{}, []string{"read"})
	if err != nil {
		t.Fatal(err)
	}

	go runner.Wait()

	select {
	case <-runner.ExitCh:
		t.Fatal("expected non-exit")
	case <-time.After(100 * time.Nanosecond):
	}
}

func TestRunner_runSanitize(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner, err := NewRunner("foo/bar", &Config{Sanitize: true}, []string{"env"})
	if err != nil {
		t.Fatal(err)
	}

	runner.outStream, runner.errStream = outStream, errStream

	pair := []*dependency.KeyPair{
		&dependency.KeyPair{
			Path:  "foo/bar",
			Key:   "b*a*r",
			Value: "baz",
		},
	}

	runner.Receive(pair)
	runner.Run()
	runner.Wait()

	expected := "b_a_r=baz"
	if !strings.Contains(outStream.String(), expected) {
		t.Fatalf("expected %q to include %q", outStream.String(), expected)
	}
}

func TestRunner_runUpcase(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner, err := NewRunner("foo/bar", &Config{Upcase: true}, []string{"env"})
	if err != nil {
		t.Fatal(err)
	}

	runner.outStream, runner.errStream = outStream, errStream

	pair := []*dependency.KeyPair{
		&dependency.KeyPair{
			Path:  "foo/bar",
			Key:   "bar",
			Value: "baz",
		},
	}

	runner.Receive(pair)
	runner.Run()
	runner.Wait()

	expected := "BAR=baz"
	if !strings.Contains(outStream.String(), expected) {
		t.Fatalf("expected %q to include %q", outStream.String(), expected)
	}
}

func TestRunner_runExitCh(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner, err := NewRunner("foo/bar", &Config{}, []string{"env"})
	if err != nil {
		t.Fatal(err)
	}

	runner.outStream, runner.errStream = outStream, errStream

	pair := []*dependency.KeyPair{
		&dependency.KeyPair{
			Path:  "foo/bar",
			Key:   "bar",
			Value: "baz",
		},
	}

	runner.Receive(pair)
	runner.Run()
	go runner.Wait()

	select {
	case <-runner.ExitCh:
		return
	case <-time.After(1 * time.Second):
		t.Fatal("expected process to exit on ExitCh")
	}
}
