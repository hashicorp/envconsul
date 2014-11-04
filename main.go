package main

import (
	"os"
)

const Name = "envconsul"
const Version = "0.3.0"

func main() {
	cli := &CLI{outStream: os.Stdout, errStream: os.Stderr}
	os.Exit(cli.Run(os.Args))
}
