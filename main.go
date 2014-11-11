package main

import (
	"os"

	"github.com/codegangsta/cli"
)

var (
	app = cli.NewApp()
)

func init() {
	app.Name = "envetcd"
	app.Author = "Joshua Rubin"
	app.Email = "jrubin@zvelo.com"
	app.Version = "0.1.0"
	app.Usage = "set environment variables from etcd"
}

func main() {
	app.Run(os.Args)
}
