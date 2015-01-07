package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCLI(t *testing.T) {
	Convey("cli should work", t, func() {
		Convey("massagePeers should work", func() {
			config.Peers = []string{"127.0.0.1:4001"}
			So(massagePeers(), ShouldBeNil)
			So(len(config.Peers), ShouldEqual, 1)
			So(config.Peers[0], ShouldEqual, "http://127.0.0.1:4001")

			config.Peers = []string{":"}
			err := massagePeers()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "parse :: missing protocol scheme")
		})

		Convey("start should execute and not panic", func() {
			So(func() { start("echo", "-n") }, ShouldNotPanic)
		})
	})
}
