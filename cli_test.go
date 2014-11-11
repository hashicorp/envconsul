package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRun_printsErrors(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("envetcd -bacon delicious", " ")

	status := cli.Run(args)
	if status == ExitCodeOK {
		t.Fatal("expected not OK exit code")
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

func TestRun__versionFlag(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("envetcd -version", " ")

	status := cli.Run(args)
	if status != ExitCodeOK {
		t.Errorf("expected %s to eq %s", status, ExitCodeOK)
	}

	expected := fmt.Sprintf("envetcd v%s", Version)
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("expected %q to eq %q", errStream.String(), expected)
	}
}

func TestRun_parseError(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("envetcd -bacon delicious", " ")

	status := cli.Run(args)
	if status != ExitCodeParseFlagsError {
		t.Errorf("expected %s to eq %s", status, ExitCodeParseFlagsError)
	}

	expected := "flag provided but not defined: -bacon"
	if !strings.Contains(errStream.String(), expected) {
		t.Fatalf("expected %q to contain %q", errStream.String(), expected)
	}
}

func TestRun_waitFlagError(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}
	args := strings.Split("envetcd -wait=watermelon:bacon", " ")

	status := cli.Run(args)
	if status != ExitCodeParseWaitError {
		t.Errorf("expected %s to eq %s", status, ExitCodeParseWaitError)
	}

	expected := "time: invalid duration watermelon"
	if !strings.Contains(errStream.String(), expected) {
		t.Fatalf("expected %q to contain %q", errStream.String(), expected)
	}
}

func TestRun_onceFlag(t *testing.T) {
	outStream, errStream := new(bytes.Buffer), new(bytes.Buffer)
	cli := &CLI{outStream: outStream, errStream: errStream}

	command := "envetcd -etcd demo.consul.io -once global/time sh -c ':'"
	args := strings.Split(command, " ")

	ch := make(chan int, 1)

	go func() {
		ch <- cli.Run(args)
	}()

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
