package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/zvelo/envetcd"
	"github.com/zvelo/zvelo-services/util"
)

func init() {
	config.EnvEtcd = &envetcd.Config{
		Etcd: &util.EtcdConfig{
			Peers: []string{"127.0.0.1:4001"},
		},
	}
}

func TestCLI(t *testing.T) {
	Convey("cli should work", t, func() {
		Convey("start should execute and not panic", func() {
			So(func() { start("echo", "-n") }, ShouldNotPanic)
		})
	})
}
