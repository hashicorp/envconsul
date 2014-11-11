package main

import (
	"os"
)

const (
	name    = "envetcd"
	version = "0.1.0"
)

func main() {
	c := &cli{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(c.Run(os.Args))
}
