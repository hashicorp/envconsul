package main

import (
	"flag"
	"fmt"
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
	appTest     = cli.NewApp()
	werckerPeer = werckerAdd()
)

func werckerAdd() string {

	etcdhost := os.Getenv("ZVELO_ETCD_HOST")
	if etcdhost == "" {
		etcdhost = "127.0.0.1"
	}
	etcdport := os.Getenv("WERCKER_ETCD_PORT")
	if etcdport == "" {
		etcdport = "4001"
	}
	return fmt.Sprintf("%s:%s", etcdhost, etcdport)
}

//Set up a new test app with some predetermined values
func init() {

}
