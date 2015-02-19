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
