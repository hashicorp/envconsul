package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"time"

	"github.com/armon/consul-kv"
)

// Configuration for watches.
type WatchConfig struct {
	ConsulAddr string
	ConsulDC   string
	Prefix     string
	Cmd        []string
}

// Connects to Consul and watches a given K/V prefix and uses that to
// execute a child process.
func watchAndExec(config *WatchConfig) (int, error) {
	kvConfig := consulkv.DefaultConfig()
	kvConfig.Address = config.ConsulAddr
	kvConfig.Datacenter = config.ConsulDC

	client, err := consulkv.NewClient(kvConfig)
	if err != nil {
		return 0, err
	}

	// Start the watcher goroutine that watches for changes in the
	// K/V and notifies us on a channel.
	pairCh := make(chan consulkv.KVPairs)
	errCh := make(chan error, 1)
	quitCh := make(chan struct{})
	defer close(quitCh)
	go watch(client, config.Prefix, pairCh, errCh, quitCh)

	// This channel is what is sent to when a process exits that we
	// are running. We start it out as `nil` since we have no process.
	var exitCh chan int

	var env map[string]string
	var cmd *exec.Cmd
	for {
		var pairs consulkv.KVPairs

		// Wait for new pairs to come on our channel or an error
		// to occur.
		select {
		case exit := <-exitCh:
			return exit, nil
		case pairs = <-pairCh:
		case err := <-errCh:
			return 0, err
		}

		newEnv := make(map[string]string)
		for _, pair := range pairs {
			newEnv[pair.Key] = string(pair.Value)
		}

		// If the environmental variables didn't actually change,
		// then don't do anything.
		if reflect.DeepEqual(env, newEnv) {
			continue
		}

		// Configuration changed, reload the process.
		cmdEnv := make([]string, 0, len(newEnv))
		for k, v := range newEnv {
			cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
		}
		cmd = exec.Command(config.Cmd[0], config.Cmd[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = cmdEnv
		err := cmd.Start()
		if err != nil {
			return 111, err
		}

		// Create a new exitCh so that previously invoked commands
		// (if any) don't cause us to exit, and start a goroutine
		// to wait for that process to end.
		exitCh = make(chan int, 1)
		go func(cmd *exec.Cmd) {
			err := cmd.Wait()
			if err == nil {
				exitCh <- 0
				return
			}

			if exiterr, ok := err.(*exec.ExitError); ok {
				// The program has exited with an exit code != 0
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					exitCh <- status.ExitStatus()
					return
				}
			}

			exitCh <- 111
		}(cmd)
	}

	return 0, nil
}

func watch(
	client *consulkv.Client,
	prefix string,
	pairCh chan<- consulkv.KVPairs,
	errCh chan<- error,
	quitCh <-chan struct{}) {
	// Get the initial list of k/v pairs. We don't do a retryableList
	// here because we want a fast fail if the initial request fails.
	meta, pairs, err := client.List(prefix)
	if err != nil {
		errCh <- err
		return
	}

	// Send the initial list out right away
	pairCh <- pairs

	// Loop forever (or until quitCh is closed) and watch the keys
	// for changes.
	curIndex := meta.ModifyIndex
	for {
		select {
		case <-quitCh:
			return
		default:
		}

		meta, pairs, err = retryableList(
			func() (*consulkv.KVMeta, consulkv.KVPairs, error) {
				return client.WatchList(prefix, curIndex)
			})
		if err != nil {
			errCh <- err
			return
		}

		// If nothing actually changed (request just timed out), return
		if meta.ModifyIndex == curIndex {
			continue
		}

		pairCh <- pairs
		curIndex = meta.ModifyIndex
	}
}

// This function is able to call KV listing functions and retry them.
// We want to retry if there are errors because it is safe (GET request),
// and erroring early is MUCH more costly than retrying over time and
// delaying the configuration propagation.
func retryableList(f func() (*consulkv.KVMeta, consulkv.KVPairs, error)) (*consulkv.KVMeta, consulkv.KVPairs, error) {
	i := 0
	for {
		m, p, e := f()
		if e != nil {
			if i >= 3 {
				return nil, nil, e
			}

			i++

			// Reasonably arbitrary sleep to just try again... It is
			// a GET request so this is safe.
			time.Sleep(time.Duration(i*2) * time.Second)
		}

		return m, p, e
	}
}
