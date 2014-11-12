package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TODO(jrubin) test recursive keys

func TestEtcd(t *testing.T) {
	Convey("Given some integer with a starting value", t, func() {
		x := 1

		Convey("When the integer is incremented", func() {
			x++

			Convey("The value should be greater by one", func() {
				So(x, ShouldEqual, 2)
			})
		})
	})
}
