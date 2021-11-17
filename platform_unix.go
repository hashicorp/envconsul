//go:build linux || darwin || freebsd || openbsd || solaris || netbsd
// +build linux darwin freebsd openbsd solaris netbsd

package main

import (
	"syscall"
)

// RuntimeSig is set to SIGURG, a signal used by the runtime on *nix systems to
// manage pre-emptive scheduling.
const RuntimeSig = syscall.SIGURG
