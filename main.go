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
	app.Version = "0.0.3"
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
			Usage:  "computer hostname for host specific configuration",
		},
		cli.StringFlag{
			Name:   "system",
			EnvVar: "ENVETCD_SYSTEM",
			Usage:  "system name for system specific configuration",
		},
		cli.StringFlag{
			Name:   "service",
			EnvVar: "ENVETCD_SERVICE",
			Usage:  "service name for service specific configuration",
		},
		cli.StringFlag{
			Name:   "prefix",
			EnvVar: "ENVETCD_PREFIX",
			Value:  "/config",
			Usage:  "etcd prefix for all keys",
		},
		cli.StringFlag{
			Name:   "log-level, l",
			EnvVar: "ENVETCD_LOG_LEVEL",
			Value:  "WARN",
			Usage:  "set log level (DEBUG, INFO, WARN, ERR)",
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
