package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewRunner(t *testing.T) {
	Convey("Given a command to execute", t, func() {
		Convey("NewRunner should return a runner object based on a given context", func() {
			runVal, err := newRunner("echo", "-n")
			So(err, ShouldBeNil)

			Convey("run executes the command", func() {
				err = runVal.run()
				So(err, ShouldBeNil)
			})
		})
	})
}
