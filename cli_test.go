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

// capture takes a command and captures the output from
// running that command.
func capture(args string) (string, error) {

	// save original Stdout reference
	stdout := os.Stdout

	// create a connected pair of files
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	// close the writer and remap stdout on end
	defer func() {
		w.Close()
		r.Close()
		os.Stdout = stdout
	}()

	// Stdout is now the connected file writer
	os.Stdout = w

	// Run the passed string as a command, stdout is readable through r.
	err = app.Run(strings.Split(args, " "))
	if err != nil {
		return "", err
	}

	// Read the output from piped file r.
	w.Close()
	output, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	// return trimmed output.
	return strings.TrimSpace(string(output)), nil
}

// Create a new flag set with the given name and cli flags.
func flagSet(name string, flags []cli.Flag) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	for _, f := range flags {
		f.Apply(set)
	}
	return set
}

// TestCLI creates a new flag set from appTest's flags
func TestCLI(t *testing.T) {
	Convey("Command should execute as expected", t, func() {
		set := flagSet(appTest.Name, appTest.Flags)
		ctx := cli.NewContext(appTest, set, set) // same flags for specific and global flags

		// Try running with invalid flags
		Convey("Invalid flags should cause an error", func() {
			_, err := capture("envetcd -shmaltz delicious")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "flag provided but not defined: -shmaltz")
		})

		// Check that version is returned
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
