package main

import (
	"testing"
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
	"crypto/rand"
	"encoding/base64"

	"github.com/armon/consul-api"
)

func createRandomString() string {
	random_bytes := make([]byte, 24)
	rand.Read(random_bytes)
	return base64.StdEncoding.EncodeToString(random_bytes)
}

func makeConsulClient(t *testing.T) *consulapi.Client {
	conf := consulapi.DefaultConfig()
	client, err := consulapi.NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return client
}

func TestAddFile(t* testing.T) {
	client := makeConsulClient(t)
	kv := client.KV()

	key := "gotest/envinput"

	// Delete all keys in the "gotest" KV space
	if _, err := kv.DeleteTree("gotest", nil); err != nil {
		t.Fatalf("err: %v", err)
	}

	var cmd = "printenv"
	if (runtime.GOOS == "windows") {
		cmd = "env"
	}

	// Create a pipe to capture os.Stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	// Create a func for copying the stdout buffer
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// Run the fsconsul listener in the background
	go func() {

		config := WatchConfig{
			ConsulAddr: consulapi.DefaultConfig().Address,
			ConsulDC:   "dc1",
			Prefix:     "gotest",
			Reload:		true,
			Cmd:		[]string{cmd},
			Upcase:		true,
		}

		_, err := watchAndExec(&config)
		if err != nil {
			t.Fatalf("Failed to run watchAndExec: %v")
		}

	}()

	// Put a test KV
	value := []byte(createRandomString())
	p := &consulapi.KVPair{Key: key, Flags: 42, Value: value}
	if _, err := kv.Put(p, nil); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Give ourselves a little bit of time for the watcher to read the file
	time.Sleep(500 * time.Millisecond)

	// Close the pipe and gather stdout to a string
	w.Close()
	out := <-outC
	os.Stdout = old
	fmt.Printf("Got output\n%s", out)

	if (!bytes.Contains([]byte(out), value)) {
		t.Fatal("Unmatched values")
	}
}
