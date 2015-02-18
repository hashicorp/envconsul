// +build linux darwin freebsd openbsd solaris netbsd

package main

import (
	"os"
	"syscall"
)

var Signals = []os.Signal{
	// POSIX.1-1990 standard
	syscall.SIGHUP,  // Hangup detected on controlling terminalor death of controlling process
	syscall.SIGINT,  // Interrupt from keyboard
	syscall.SIGQUIT, // Quit from keyboard
	syscall.SIGILL,  // Illegal Instruction
	syscall.SIGABRT, // Abort signal from abort(3)
	syscall.SIGFPE,  // Floating point exception
	// syscall.SIGKILL, // Kill signal
	syscall.SIGSEGV, // Invalid memory reference
	syscall.SIGPIPE, // Broken pipe: write to pipe with no readers
	syscall.SIGALRM, // Timer signal from alarm(2)
	syscall.SIGTERM, // Termination signal
	syscall.SIGUSR1, // User-defined signal 1
	syscall.SIGUSR2, // User-defined signal 2
	syscall.SIGCHLD, // Child stopped or terminated
	syscall.SIGCONT, // Continue if stopped
	// syscall.SIGSTOP, // Stop process
	syscall.SIGTSTP, // Stop typed at tty
	syscall.SIGTTIN, // tty input for background process
	syscall.SIGTTOU, // tty output for background process

	// Described in  SUSv2 and POSIX.1-2001
	syscall.SIGBUS,    // Bus error (bad memory access)
	syscall.SIGPROF,   // Profiling timer expired
	syscall.SIGSYS,    // Bad argument to routine (SVr4)
	syscall.SIGTRAP,   // Trace/breakpoint trap
	syscall.SIGURG,    // Urgent condition on socket (4.2BSD)
	syscall.SIGVTALRM, // Virtual alarm clock (4.2BSD)
	syscall.SIGXCPU,   // CPU time limit exceeded (4.2BSD)
	syscall.SIGXFSZ,   // File size limit exceeded (4.2BSD)
}
