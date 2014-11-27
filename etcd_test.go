package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/coreos/go-etcd/etcd"
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
					/*So(keyPairs["service_zvelo-nsqd_PortA"], ShouldEqual, "4150")
					So(keyPairs["service_zvelo-nsqd_PortB"], ShouldEqual, "4151")
					So(keyPairs["service_zvelo-nsqd_LookupAddress"], ShouldEqual, "172.17.8.101")
					So(keyPairs["port"], ShouldEqual, "1234")
					*/
				})
				Convey("Testing override keys", func() {
					etcdAddress := fmt.Sprintf("%s%s", "http://", werckerPeer)
					etcdClient := etcd.NewClient([]string{etcdAddress})

					etcdClient.Delete("/config/global/systemtest", true)
					etcdClient.Delete("/config/system/systemtest", true)
					etcdClient.Delete("/config/service/systemtest", true)

					Convey("Setting /config/global testkey only", func() {
						etcdClient.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
						keyPairs := getKeyPairs(etcdConf, etcdClient)
						_, isExisting := keyPairs["system_testsystem_testkey"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["system_testsystem_testkey"], ShouldEqual, "testGlobalVal")

						Convey("Setting /config/global/testKey", func() {
							etcdClient.Set("/config/global/testKey", "testGlobalVal2", 0)
							keyPairs := getKeyPairs(etcdConf, etcdClient)

							_, isExisting := keyPairs["testKey"]
							So(isExisting, ShouldBeTrue)
							So(keyPairs["system_testsystem_testkey"], ShouldEqual, "testGlobalVal")
							So(keyPairs["testKey"], ShouldEqual, "testGlobalVal2")

							Convey("Setting /config/system/systemtest/testKey should override the global testKey", func() {
								etcdClient.Set("/config/system/systemtest/testKey", "testsystemVal", 0)
								keyPairs := getKeyPairs(etcdConf, etcdClient)
								_, isExisting := keyPairs["testKey"]
								So(isExisting, ShouldBeTrue)
								So(keyPairs["system_testsystem_testkey"], ShouldEqual, "testGlobalVal")
								So(keyPairs["testKey"], ShouldEqual, "testsystemVal")

							})
						})

						Convey("Setting /config/service/systemtest/testserviceKey should not be in the keypair", func() {
							etcdClient.Set("/config/service/systemtest/testserviceKey", "testserviceVal", 0)
							keyPairs := getKeyPairs(etcdConf, etcdClient)
							fmt.Println(keyPairs)
							_, isExisting := keyPairs["testserviceKey"]
							So(isExisting, ShouldBeFalse)

							So(keyPairs["system_testsystem_testkey"], ShouldEqual, "testGlobalVal")
							So(keyPairs["testserviceKey"], ShouldBeEmpty)
						})
					})

				})
			})
		})
	})
}
