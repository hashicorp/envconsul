package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/zvelo/envetcd"
	"github.com/zvelo/zvelo-services/util"
)

const (
	version = "0.1.1"
)

var (
	app    = cli.NewApp()
	config struct {
		EnvEtcd  *envetcd.Config
		WriteEnv string
		Output   string
		CleanEnv bool
	}
)

func init() {
	hostname, _ := os.Hostname()
	app.Name = "envetcd"
	app.Author = "Joshua Rubin"
	app.Email = "jrubin@zvelo.com"
	app.Version = version
	app.Usage = "set environment variables from etcd"
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:   "peers, C",
			EnvVar: "ENVETCD_PEERS",
			Value:  &cli.StringSlice{"127.0.0.1:4001"},
			Usage:  "a comma-delimited list of machine addresses in the cluster (default: \"127.0.0.1:4001\")",
		},
		cli.StringFlag{
			Name:   "ca-file",
			EnvVar: "ENVETCD_CA_FILE",
			Usage:  "certificate authority file",
		},
		cli.StringFlag{
			Name:   "cert-file",
			EnvVar: "ENVETCD_CERT_FILE",
			Usage:  "tls client certificate file",
		},
		cli.StringFlag{
			Name:   "key-file",
			EnvVar: "ENVETCD_KEY_FILE",
			Usage:  "tls client key file",
		},
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
			Name:   "no-sync",
			EnvVar: "ENVETCD_NO_SYNC",
			Usage:  "don't synchronize cluster information before sending request",
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
	}
	app.Action = run
}

func main() {
	app.Run(os.Args)
}
