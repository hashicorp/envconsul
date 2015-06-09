package main

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
)

func TestNewRunner(t *testing.T) {
	config := testConfig("", t)
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

	config := testConfig(`
		prefixes = ["foo/bar"]
	`, t)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}
	runner.outStream, runner.errStream = ioutil.Discard, ioutil.Discard

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

	config := testConfig(`
		sanitize = true
		prefixes = ["foo/bar"]
	`, t)

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

	exitCh, err := runner.Run()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-exitCh:
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

	config := testConfig(`
		upcase = true
		prefixes = ["foo/bar"]
	`, t)

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

	exitCh, err := runner.Run()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-exitCh:
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

	config := testConfig(`
		prefixes = ["foo/bar"]
	`, t)

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

	exitCh, err := runner.Run()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-exitCh:
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

	config := testConfig(`
		upcase = true
		prefixes = ["config/global", "config/redis"]
	`, t)

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

	exitCh, err := runner.Run()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-exitCh:
		expected := "ADDRESS=1.2.3.4"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}

		expected = "PORT=8000"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}
	}
}

func TestStart_noRunMissingData(t *testing.T) {
	config := testConfig(`
		prefixes = ["foo/bar"]
	`, t)

	runner, err := NewRunner(config, []string{"sh", "-c", "echo $BAR"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	// Kind of hacky, but wait for the runner to return an error, indicating we
	// are all setup.
	select {
	case <-runner.watcher.ErrCh:
	}

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-time.After(50 * time.Millisecond):
		expected := ""
		if outStream.String() != expected {
			t.Fatalf("expected %q to be %q", outStream.String(), expected)
		}
	}
}

func TestStart_runsCommandOnChange(t *testing.T) {
	prefix, err := dep.ParseStoreKeyPrefix("foo/bar")
	if err != nil {
		t.Fatal(err)
	}

	config := testConfig(`
		prefixes = ["foo/bar"]
	`, t)

	runner, err := NewRunner(config, []string{"sh", "-c", "echo $BAR"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	// Kind of hacky, but wait for the runner to return an error, indicating we
	// are all setup.
	select {
	case <-runner.watcher.ErrCh:
	}

	pair := []*dep.KeyPair{
		&dep.KeyPair{
			Path:  "foo/bar",
			Key:   "BAR",
			Value: "one",
		},
	}
	runner.watcher.DataCh <- &watch.View{Dependency: prefix, Data: pair}

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-time.After(200 * time.Millisecond):
		expected := "one\n"
		if outStream.String() != expected {
			t.Fatalf("expected %q to be %q", outStream.String(), expected)
		}
	}

	outStream.Reset()
	errStream.Reset()

	pair = []*dep.KeyPair{
		&dep.KeyPair{
			Path:  "foo/bar",
			Key:   "BAR",
			Value: "two",
		},
	}
	runner.watcher.DataCh <- &watch.View{Dependency: prefix, Data: pair}

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-time.After(200 * time.Millisecond):
		expected := "two\n"
		if outStream.String() != expected {
			t.Fatalf("expected %q to be %q", outStream.String(), expected)
		}
	}
}

func TestSignal_sendsToChild(t *testing.T) {
	script := test.CreateTempfile([]byte(`
		trap 'exit 123' USR1
		while : ; do sleep 0.1; done
	`), t)
	defer test.DeleteTempfile(script, t)

	config := testConfig("", t)

	runner, err := NewRunner(config, []string{"bash", script.Name()}, false)
	if err != nil {
		t.Fatal(err)
	}
	runner.outStream, runner.errStream = ioutil.Discard, ioutil.Discard
	defer runner.Stop()

	exitCh, err := runner.Run()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-exitCh:
		t.Error("unexpected exit")
	case <-time.After(10 * time.Millisecond):
		// Continue
	}

	if err := runner.Signal(syscall.SIGUSR1); err != nil {
		t.Fatal(err)
	}

	select {
	case code := <-exitCh:
		if code != 123 {
			t.Errorf("bad exit code: %d", code)
		}
	}
}
