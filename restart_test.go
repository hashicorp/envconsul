package main

import (
	"strconv"
	"testing"
	"time"
)

func TestRestart(t *testing.T) {
	future := time.Now().Add(time.Second)

	config := WatchConfig{
		ConsulAddr: "127.0.0.1:8500",
		ConsulDC:   "",
		Cmd:        []string{"go", "run", "crash.go", strconv.FormatInt(future.Unix(), 10)},
		ErrExit:    true,
		Prefix:     "",
		Reload:     false,
		Restart:    true,
		Terminate:  false,
		Timeout:    0,
		Sanitize:   true,
		Upcase:     false,
	}
	result, err := watchAndExec(&config)
	if err != nil {
		t.Error(err)
	}

	if result != 0 {
		t.Error("result is ", result)
	}

	if !time.Now().Before(future) {
		t.Error("exit too quickly")
	}
}
