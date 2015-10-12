package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul-template/test"
	"github.com/hashicorp/consul-template/watch"
	"github.com/hashicorp/consul/testutil"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/vault"
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
		prefix {
			path = "foo/bar"
		}
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

func TestRun_consul(t *testing.T) {
	t.Parallel()

	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	consul.SetKV("foo/bar/bar", []byte("baz"))

	config := testConfig(fmt.Sprintf(`
		consul = "%s"
		prefix {
			path = "foo/bar"
		}
	`, consul.HTTPAddr), t)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.ExitCh:
		expected := "bar=baz"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}
	}
}

func TestRun_vault(t *testing.T) {
	t.Parallel()

	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := http.TestServer(t, core)
	defer ln.Close()

	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/foo",
		Data: map[string]interface{}{
			"zip":  "zap",
			"ding": "dong",
		},
		ClientToken: token,
	}
	if _, err := core.HandleRequest(req); err != nil {
		t.Fatal(err)
	}

	vaultconfig := vaultapi.DefaultConfig()
	vaultconfig.Address = addr
	client, err := vaultapi.NewClient(vaultconfig)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(token)

	// Create a new token - the core token is a root token and is therefore
	// not renewable
	secret, err := client.Auth().Token().Create(&vaultapi.TokenCreateRequest{
		Lease: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}

	config := testConfig(fmt.Sprintf(`
		vault {
			address = "%s"
			token   = "%s"
			ssl {
				enabled = false
			}
		}

		secret {
			path = "secret/foo"
		}
	`, addr, secret.Auth.ClientToken), t)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.ExitCh:
	case <-time.After(250 * time.Millisecond):
	}

	expected := "secret_foo_zip=zap"
	if !strings.Contains(outStream.String(), expected) {
		t.Errorf("expected %q to include %q", outStream.String(), expected)
	}

	expected = "secret_foo_ding=dong"
	if !strings.Contains(outStream.String(), expected) {
		t.Errorf("expected %q to include %q", outStream.String(), expected)
	}
}

func TestRun_vaultPrecedenceOverConsul(t *testing.T) {
	t.Parallel()

	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	consul.SetKV("secret/foo/secret_foo_zip", []byte("baz"))

	vaultCore, _, token := vault.TestCoreUnsealed(t)
	ln, vaultAddr := http.TestServer(t, vaultCore)
	defer ln.Close()

	req := &logical.Request{
		Operation: logical.WriteOperation,
		Path:      "secret/foo",
		Data: map[string]interface{}{
			"zip":  "zap",
			"ding": "dong",
		},
		ClientToken: token,
	}
	if _, err := vaultCore.HandleRequest(req); err != nil {
		t.Fatal(err)
	}

	vaultconfig := vaultapi.DefaultConfig()
	vaultconfig.Address = vaultAddr
	client, err := vaultapi.NewClient(vaultconfig)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(token)

	// Create a new token - the core token is a root token and is therefore
	// not renewable
	secret, err := client.Auth().Token().Create(&vaultapi.TokenCreateRequest{
		Lease: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}

	config := testConfig(fmt.Sprintf(`
		consul = "%s"

		vault {
			address = "%s"
			token   = "%s"
			ssl {
				enabled = false
			}
		}

		secret {
			path = "secret/foo"
		}

		prefix {
			path = "secret/foo"
		}
	`, consul.HTTPAddr, vaultAddr, secret.Auth.ClientToken), t)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.ExitCh:
	case <-time.After(250 * time.Millisecond):
	}

	expected := "secret_foo_zip=zap"
	if !strings.Contains(outStream.String(), expected) {
		t.Errorf("expected %q to include %q", outStream.String(), expected)
	}

	expected = "secret_foo_ding=dong"
	if !strings.Contains(outStream.String(), expected) {
		t.Errorf("expected %q to include %q", outStream.String(), expected)
	}
}

func TestRun_sanitize(t *testing.T) {
	t.Parallel()

	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	consul.SetKV("foo/bar/b*a*r", []byte("baz"))

	config := testConfig(fmt.Sprintf(`
		consul = "%s"
		sanitize = true
		prefix {
			path = "foo/bar"
		}
	`, consul.HTTPAddr), t)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.ExitCh:
		expected := "b_a_r=baz"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}
	}
}

func TestRun_upcase(t *testing.T) {
	t.Parallel()

	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	consul.SetKV("foo/bar/bar", []byte("baz"))

	config := testConfig(fmt.Sprintf(`
		consul = "%s"
		upcase = true
		prefix {
			path = "foo/bar"
		}
	`, consul.HTTPAddr), t)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.ExitCh:
		expected := "BAR=baz"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}
	}
}

func TestRun_pristine(t *testing.T) {
	t.Parallel()

	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	consul.SetKV("foo/bar/bar", []byte("baz"))

	config := testConfig(fmt.Sprintf(`
		consul = "%s"
		pristine = true
		prefix {
			path = "foo/bar"
		}
	`, consul.HTTPAddr), t)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.ExitCh:
		expected := "bar=baz"
		if !strings.Contains(outStream.String(), expected) {
			t.Fatalf("expected %q to include %q", outStream.String(), expected)
		}

		notExpected := "HOME="
		if strings.Contains(outStream.String(), notExpected) {
			t.Fatalf("did not expect %q to include %q", outStream.String(), notExpected)
		}
	}
}

func TestRun_merges(t *testing.T) {
	t.Parallel()

	consul := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer consul.Stop()

	consul.SetKV("config/global/address", []byte("1.2.3.4"))
	consul.SetKV("config/global/port", []byte("5598"))
	consul.SetKV("config/redis/port", []byte("8000"))

	config := testConfig(fmt.Sprintf(`
		consul = "%s"
		upcase = true

		prefix {
			path = "config/global"
		}

		prefix {
			path = "config/redis"
		}
	`, consul.HTTPAddr), t)

	runner, err := NewRunner(config, []string{"env"}, true)
	if err != nil {
		t.Fatal(err)
	}

	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	runner.outStream, runner.errStream = outStream, errStream

	go runner.Start()
	defer runner.Stop()

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case <-runner.ExitCh:
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
		prefix {
			path = "foo/bar"
		}
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
		prefix {
			path = "foo/bar"
		}
	`, t)

	f := test.CreateTempfile(nil, t)
	defer os.Remove(f.Name())
	os.Remove(f.Name())

	readFile := func(path string, ch chan string) {
		for {
			contents, err := ioutil.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					time.Sleep(50 * time.Millisecond)
					continue
				} else {
					t.Fatal(err)
					return
				}
			}

			ch <- string(contents)
			return
		}
	}

	runner, err := NewRunner(config, []string{"sh", "-c", "echo $BAR > " + f.Name()}, true)
	if err != nil {
		t.Fatal(err)
	}

	runner.outStream, runner.errStream = ioutil.Discard, ioutil.Discard

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

	contentCh := make(chan string)
	go readFile(f.Name(), contentCh)

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case content := <-contentCh:
		expected := "one\n"
		if content != expected {
			t.Fatalf("expected %q to be %q", content, expected)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected file to be rendered by now")
	}

	// Delete the file - otherwise the next read could have a false-positive since
	// the file already exists
	os.Remove(f.Name())

	pair = []*dep.KeyPair{
		&dep.KeyPair{
			Path:  "foo/bar",
			Key:   "BAR",
			Value: "two",
		},
	}
	runner.watcher.DataCh <- &watch.View{Dependency: prefix, Data: pair}

	contentCh = make(chan string)
	go readFile(f.Name(), contentCh)

	select {
	case err := <-runner.ErrCh:
		t.Fatal(err)
	case content := <-contentCh:
		expected := "two\n"
		if content != expected {
			t.Fatalf("expected %q to be %q", content, expected)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected file to be rendered by now")
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
