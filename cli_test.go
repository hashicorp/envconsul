package main

import (
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

func TestCLI(t *testing.T) {
	Convey("Command should execute as expected", t, func() {

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
	})
}
