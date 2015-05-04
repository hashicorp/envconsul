package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/zvelo/envetcd"
	"github.com/zvelo/zvelo-services/util"
)

const (
	name    = "envetcd"
	version = "0.2.1"
)

type configT struct {
	EnvEtcd  *envetcd.Config
	WriteEnv string
	Output   string
	CleanEnv bool
}

var (
	app    = cli.NewApp()
	config configT
)

func init() {
	hostname, _ := os.Hostname()
	app.Name = name
	app.Version = version
	app.Usage = "set environment variables from etcd"
	app.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "jrubin@zvelo.com"},
	}
	app.Flags = append(util.EtcdFlags, []cli.Flag{
		cli.StringFlag{
			Name:   "hostname",
			EnvVar: "HOSTNAME",
			Value:  hostname,
			Usage:  "computer hostname for host specific configuration",
		},
		cli.StringFlag{
			Name:   "system",
			EnvVar: "ENVETCD_SYSTEM",
			Usage:  "system name for system specific configuration",
		},
		cli.StringFlag{
			Name:   "service, s",
			EnvVar: "ENVETCD_SERVICE",
			Usage:  "service name for service specific configuration",
		},
		cli.StringFlag{
			Name:   "prefix, p",
			EnvVar: "ENVETCD_PREFIX",
			Value:  "/config",
			Usage:  "etcd prefix for all keys",
		},
		util.LogLevelFlag,
		cli.StringFlag{
			Name:   "write-env, w",
			EnvVar: "ENVETCD_WRITE_ENV",
			Usage:  "don't run a command, just write the environment to a 'sourcable' file",
		},
		cli.StringFlag{
			Name:   "output, o",
			EnvVar: "ENVETCD_OUTPUT",
			Usage:  "write stdout from the command to this file",
		},
		cli.BoolFlag{
			Name:   "clean-env, c",
			EnvVar: "ENVETCD_CLEAN_ENV",
			Usage:  "don't inherit any environment variables other than those pulled from etcd",
		},
		cli.BoolFlag{
			Name:   "no-sanitize",
			EnvVar: "ENVETCD_NO_SANITIZE",
			Usage:  "don't remove bad characters from environment keys",
		},
		cli.BoolFlag{
			Name:   "no-upcase",
			EnvVar: "ENVETCD_NO_UPCASE",
			Usage:  "don't convert all environment keys to uppercase",
		},
		cli.BoolFlag{
			Name:   "use-default-gateway, d",
			EnvVar: "ENVETCD_USE_DEFAULT_GATEWAY",
			Usage:  "expose the default gateway as $ENVETCD_DEFAULT_GATEWAY",
		},
	}...)
	app.Before = setup
	app.Action = run
}

func main() {
	app.Run(os.Args)
}
