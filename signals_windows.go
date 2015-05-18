// +build windows plan9

package main

import (
	"os"
	"syscall"
)

var Signals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGQUIT,
}

var SignalLookup = map[string]os.Signal {
	"SIGINT": syscall.SIGINT,
	"SIGTERM": syscall.SIGTERM,
	"SIGQUIT": syscall.SIGQUIT,
}
