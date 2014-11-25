package main

import (
	"flag"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func capture(args string) (string, error) {
	stdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	defer func() {
		w.Close()
		os.Stdout = stdout
	}()

	os.Stdout = w
	err = app.Run(strings.Split(args, " "))
	if err != nil {
		return "", err
	}

	w.Close()
	output, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func flagSet(name string, flags []cli.Flag) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	for _, f := range flags {
		f.Apply(set)
	}
	return set
}

func TestCLI(t *testing.T) {
	Convey("Command should execute as expected", t, func() {
		set := flagSet(appTest.Name, appTest.Flags)
		ctx := cli.NewContext(appTest, set, set)

		Convey("Invalid flags should cause an error", func() {
			_, err := capture("envetcd -shmaltz delicious")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "flag provided but not defined: -shmaltz")
		})

		Convey("Version should be printed", func() {
			output, err := capture("envetcd --version")
			So(err, ShouldBeNil)
			So(output, ShouldEqual, "envetcd version "+app.Version)
		})
		Convey("Initlogger should be printed", func() {
			initLogger(ctx)
			//os.Args = []string{"./envetcd", "no-upcase=true", "no-sync=true", "-o", "ooooo", "--system", "nsq", "-c", "env"}
			//fmt.Println(appTest.Run(os.Args))
		})

	})
}

var (
	appTest = cli.NewApp()
)

//Set up a new test app with some predetermined values
func init() {
	os.Setenv("ENVETCD_CLEAN_ENV", "true")
	os.Setenv("ENVETCD_NO_SANITIZE", "true")
	os.Setenv("ENVETCD_NO_UPCASE", "true")
	appTest.Name = "testApp"
	appTest.Author = "Karl Dominguez"
	appTest.Email = "kdominguez@zvelo.com"
	appTest.Version = "0.0.4"
	appTest.Usage = "get environment variables from etcd"
	appTest.Flags = []cli.Flag{
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
			Value:  "env",
			Usage:  "computer hostname for host specific configuration",
		},
		cli.StringFlag{
			Name:   "system",
			EnvVar: "ENVETCD_SYSTEM",
			Value:  "nsq",
			Usage:  "system name for system specific configuration",
		},
		cli.StringFlag{
			Name:   "service",
			EnvVar: "ENVETCD_SERVICE",
			Value:  "redis",
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
			Value:  "DEBUG",
			Usage:  "set log level (DEBUG, INFO, WARN, ERR)",
		},
		cli.StringFlag{
			Name:   "output, o",
			Value:  "testOut.txt",
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
	appTest.Action = run
}
