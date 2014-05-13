package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	var errExit bool
	var reload bool
	var consulAddr string
	var consulDC string
	flag.Usage = usage
	flag.BoolVar(
		&errExit, "errexit", false,
		"exit if there is an error watching config keys")
	flag.BoolVar(
		&reload, "reload", false,
		"if set, restarts the process when config changes")
	flag.StringVar(
		&consulAddr, "addr", "127.0.0.1:8500",
		"consul HTTP API address with port")
	flag.StringVar(
		&consulDC, "dc", "",
		"consul datacenter, uses local if blank")
	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		return 1
	}

	args := flag.Args()
	config := WatchConfig{
		ConsulAddr: consulAddr,
		ConsulDC:   consulDC,
		Cmd:        args[1:],
		ErrExit:    errExit,
		Prefix:     args[0],
		Reload:     reload,
	}
	result, err := watchAndExec(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 111
	}

	return result
}

func usage() {
	cmd := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, strings.TrimSpace(helpText)+"\n\n", cmd)
	flag.PrintDefaults()
}

const helpText = `
Usage: %s [options] prefix child...

  Sets environmental variables for the child process by reading
  K/V from Consul's K/V store with the given prefix.

Options:
`
