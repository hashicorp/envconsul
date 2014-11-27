package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/coreos/go-etcd/etcd"
	"log"
	"os/exec"
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

		cmd := exec.Command("env")
		out, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Container ENV: %s\n", out)

		etcdAddress := fmt.Sprintf("%s%s", "http://", werckerPeer)
		etcdPeer := etcd.NewClient([]string{etcdAddress})
		etcdPeer.Delete("/config", true)
		etcdPeer.SetDir("/config/system/systemtest", 0)
		etcdPeer.SetDir("/config/service/servicetest", 0)
		etcdPeer.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
		etcdPeer.Set("/config/host/env", "", 0)

		fmt.Println("ETCD PEER ADDRESS", werckerAdd())

		Convey("newEtcdConfig should return an etcd config", func() {
			So(etcdConf.Key.Prefix, ShouldEqual, "/config")
			So(etcdConf.Key.Hostname, ShouldEqual, "env")
			So(etcdConf.Sync, ShouldBeTrue)
			So(etcdConf.Key.System, ShouldEqual, "systemtest")
			So(etcdConf.Key.Service, ShouldEqual, "servicetest")
			So(etcdConf.Peers, ShouldContain, "http://127.0.0.1:4001")

			Convey("getEndpoints should return an array of end points", func() {
				endPoints, err := getEndpoints(etcdConf)
				So(err, ShouldBeNil)
				So(endPoints, ShouldContain, "http://127.0.0.1:4001")
			})

			Convey("getTransport should return an http transport object", func() {
				transObj, err := getTransport(etcdConf)
				So(err, ShouldBeNil)
				So(transObj.Proxy, ShouldBeNil)
				So(transObj.DisableKeepAlives, ShouldBeFalse)
				So(transObj.TLSHandshakeTimeout, ShouldEqual, 10000000000)
				So(transObj.MaxIdleConnsPerHost, ShouldBeZeroValue)
			})

			Convey("getClient should return an etcd client based on a given config", func() {
				So(err, ShouldBeNil)
				So(etcdClient, ShouldNotBeEmpty)
				So(etcdClient.CheckRetry, ShouldBeNil)

				Convey("getKeyPairs returns keypairs", func() {
					keyPairs := getKeyPairs(etcdConf, etcdClient)
					So(keyPairs, ShouldNotBeEmpty)
				})
				Convey("Testing override keys", func() {

					Convey("Setting /config/global testkey only", func() {
						etcdPeer.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
						keyPairs := getKeyPairs(etcdConf, etcdPeer)
						_, isExisting := keyPairs["systemtest_testKey"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")

						Convey("Setting /config/global/testKey", func() {
							etcdPeer.Set("/config/global/testKey", "testGlobalVal2", 0)
							keyPairs := getKeyPairs(etcdConf, etcdClient)

							_, isExisting := keyPairs["testKey"]
							So(isExisting, ShouldBeTrue)
							So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
							So(keyPairs["testKey"], ShouldEqual, "testGlobalVal2")

							Convey("Setting /config/system/systemtest/testKey should override the global testKey", func() {
								etcdPeer.Set("/config/system/systemtest/testKey", "testsystemVal", 0)
								keyPairs := getKeyPairs(etcdConf, etcdClient)
								fmt.Println("keyPairs", keyPairs)

								_, isExisting := keyPairs["testKey"]
								So(isExisting, ShouldBeTrue)
								So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
								So(keyPairs["testKey"], ShouldEqual, "testsystemVal")

							})
						})

						Convey("Setting /config/service/systemtest/testserviceKey should not be in the keypair", func() {
							etcdPeer.Set("/config/service/systemtest/testserviceKey", "testserviceVal", 0)
							keyPairs := getKeyPairs(etcdConf, etcdClient)
							fmt.Println(keyPairs)
							_, isExisting := keyPairs["testserviceKey"]
							So(isExisting, ShouldBeFalse)

							So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
							So(keyPairs["testserviceKey"], ShouldBeEmpty)
						})
					})
				})
			})
		})
	})
}
