package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	dep "github.com/hashicorp/consul-template/dependency"
)

func TestNewRunner(t *testing.T) {
	config := DefaultConfig()
	command := []string{"env"}
	runner, err := NewRunner(config, command, true)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(runner.config, config) {
		t.Errorf("expected %#v to be %#v", runner.config, config)
	}

	if !reflect.DeepEqual(runner.command, command) {
		t.Errorf("expected %#v to be %#v", runner.command, command)
	}

	if runner.once != true {
		t.Error("expected once to be true")
	}

	if runner.client == nil {
		t.Error("expected client to exist")
	}

	if runner.watcher == nil {
		t.Error("expected watcher to exist")
	}

	if runner.data == nil {
		t.Error("expected data to exist")
	}

	if runner.outStream == nil {
		t.Errorf("expected outStream to exist")
	}

	if runner.errStream == nil {
		t.Error("expected errStream to exist")
	}

	if runner.ErrCh == nil {
		t.Error("expected ErrCh to exist")
	}

	if runner.DoneCh == nil {
		t.Error("expected DoneCh to exist")
	}

	if runner.ExitCh == nil {
		t.Error("expected ExitCh to exit")
	}
}

func TestReceive_receivesData(t *testing.T) {
	prefix, err := dep.ParseStoreKeyPrefix("foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultConfig()
	config.Prefixes = append(config.Prefixes, prefix)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	data := []*dep.KeyPair{&dep.KeyPair{Path: "foo/bar"}}
	runner.Receive(prefix, data)

	if !reflect.DeepEqual(runner.data[prefix.HashCode()], data) {
		t.Errorf("expected %#v to be %#v", runner.data[prefix.HashCode()], data)
	}
}

func TestRun_sanitize(t *testing.T) {
	prefix, err := dep.ParseStoreKeyPrefix("foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultConfig()
	config.Sanitize = true
	config.Prefixes = append(config.Prefixes, prefix)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	pair := []*dep.KeyPair{
		&dep.KeyPair{
			Path:  "foo/bar",
			Key:   "b*a*r",
			Value: "baz",
		},
	}

	runner.Receive(prefix, pair)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-runner.ExitCh:
		expected := "b_a_r=baz"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}
	}
}

func TestRun_upcase(t *testing.T) {
	prefix, err := dep.ParseStoreKeyPrefix("foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultConfig()
	config.Upcase = true
	config.Prefixes = append(config.Prefixes, prefix)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	pair := []*dep.KeyPair{
		&dep.KeyPair{
			Path:  "foo/bar",
			Key:   "bar",
			Value: "baz",
		},
	}

	runner.Receive(prefix, pair)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-runner.ExitCh:
		expected := "BAR=baz"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}
	}
}

func TestRun_exitCh(t *testing.T) {
	prefix, err := dep.ParseStoreKeyPrefix("foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultConfig()
	config.Prefixes = append(config.Prefixes, prefix)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	pair := []*dep.KeyPair{
		&dep.KeyPair{
			Path:  "foo/bar",
			Key:   "bar",
			Value: "baz",
		},
	}

	runner.Receive(prefix, pair)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-runner.ExitCh:
		// Ok
	}
}

func TestRun_merges(t *testing.T) {
	globalPrefix, err := dep.ParseStoreKeyPrefix("config/global")
	if err != nil {
		t.Fatal(err)
	}

	redisPrefix, err := dep.ParseStoreKeyPrefix("config/redis")
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultConfig()
	config.Upcase = true
	config.Prefixes = append(config.Prefixes, globalPrefix)
	config.Prefixes = append(config.Prefixes, redisPrefix)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	globalData := []*dep.KeyPair{
		&dep.KeyPair{
			Path:  "config/global",
			Key:   "address",
			Value: "1.2.3.4",
		},
		&dep.KeyPair{
			Path:  "config/global",
			Key:   "port",
			Value: "5598",
		},
	}
	runner.Receive(globalPrefix, globalData)

	redisData := []*dep.KeyPair{
		&dep.KeyPair{
			Path:  "config/redis",
			Key:   "port",
			Value: "8000",
		},
	}
	runner.Receive(redisPrefix, redisData)

	if err := runner.Run(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-runner.ExitCh:
		expected := "ADDRESS=1.2.3.4\nPORT=8000"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}
	}
}
