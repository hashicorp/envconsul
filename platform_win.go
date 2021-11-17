//go:build windows
// +build windows

package main

import (
	"os"
)

// RuntimeSig is set to nil on windows as it doesn't support the signal (SIGURG)
var RuntimeSig = os.Signal(nil)
