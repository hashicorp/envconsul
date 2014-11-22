package main

import (
	"github.com/codegangsta/cli"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TODO(jrubin) test recursive keys
func TestEtcd(t *testing.T) {
	Convey("When getting keys from etcd", t, func() {
		set := flagSet(appTest.Name, appTest.Flags)
		ctx := cli.NewContext(appTest, set, set)

		etcdConf := newEtcdConfig(ctx)
		etcdClient, err := getClient(etcdConf)

		Convey("newEtcdConfig should return an etcd config", func() {
			So(etcdConf.Key.Prefix, ShouldEqual, "/config")
			So(etcdConf.Key.Hostname, ShouldEqual, "env")
			So(etcdConf.Sync, ShouldBeTrue)
			So(etcdConf.Key.System, ShouldEqual, "nsq")
			So(etcdConf.Key.Service, ShouldEqual, "redis")
			So(etcdConf.Peers, ShouldContain, "http://127.0.0.1:4001")

			Convey("getClient should return an etcd client based on a given config", func() {
				So(err, ShouldBeNil)
				So(etcdClient, ShouldNotBeEmpty)
				So(etcdClient.CheckRetry, ShouldBeNil)

			})
		})
	})
}
