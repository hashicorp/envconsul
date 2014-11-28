package main

import (
	"github.com/codegangsta/cli"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewRunner(t *testing.T) {
	Convey("Given a command to execute", t, func() {
		set := flagSet(appTest.Name, appTest.Flags)
		ctx := cli.NewContext(appTest, set, set)

		Convey("NewRunner should return a runner object based on a given context", func() {
			runVal, err := newRunner(ctx, []string{"env"})
			So(err, ShouldBeNil)

			Convey("run executes the command", func() {
				err = runVal.run()
				So(err, ShouldBeNil)
			})
		})
	})
}
