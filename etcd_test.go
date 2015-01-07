package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/coreos/go-etcd/etcd"
	"github.com/zvelo/zvelo-services/util"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	util.InitLogger("WARN")
}

func genEtcdURI() (string, string, string) {
	etcdHost := os.Getenv("WERCKER_ETCD_HOST")
	if len(etcdHost) == 0 {
		etcdHost = os.Getenv("ETCD_HOST")
	}

	etcdPort := os.Getenv("WERCKER_ETCD_PORT")
	if len(etcdPort) == 0 {
		etcdPort = os.Getenv("ETCD_PORT")
	}

	return fmt.Sprintf("http://%s:%s", etcdHost, etcdPort), etcdHost, etcdPort
}

func genTestConfig() {
	etcdURI, _, _ := genEtcdURI()

	config.Peers = []string{etcdURI}
	config.Prefix = "/config"
	config.Hostname = "env"
	config.Sync = true
	config.System = "systemtest"
	config.Service = "servicetest"
}

func TestEtcd(t *testing.T) {
	Convey("When getting keys from etcd", t, func() {
		genTestConfig()
		_, etcdHost, etcdPort := genEtcdURI()

		So(etcdHost, ShouldNotBeEmpty)
		So(etcdPort, ShouldNotBeEmpty)

		client := etcd.NewClient(config.Peers)
		client.SetDir("/config/system/systemtest", 0)
		client.SetDir("/config/service/servicetest", 0)
		client.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
		client.Set("/config/host/env", "", 0)

		Convey("configt should be valid", func() {
			So(config.Prefix, ShouldEqual, "/config")
			So(config.Hostname, ShouldEqual, "env")
			So(config.Sync, ShouldBeTrue)
			So(config.System, ShouldEqual, "systemtest")
			So(config.Service, ShouldEqual, "servicetest")
			So(config.Peers, ShouldNotBeEmpty)

			Convey("getClient should return an etcd client based on a given config", func() {
				etcdClient, err := getClient()
				So(err, ShouldBeNil)
				So(etcdClient, ShouldNotBeEmpty)
				So(etcdClient.CheckRetry, ShouldBeNil)

				Convey("getKeyPairs returns keypairs", func() {
					keyPairs := getKeyPairs(client)
					So(keyPairs, ShouldNotBeEmpty)
				})

				Convey("Testing override keys", func() {
					Convey("Setting /config/global testkey only", func() {
						client.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
						keyPairs := getKeyPairs(client)

						_, isExisting := keyPairs["systemtest_testKey"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")

					})

					Convey("Setting /config/global/testKey", func() {
						client.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
						client.Set("/config/global/testKey", "testGlobalVal2", 0)
						keyPairs := getKeyPairs(client)

						_, isExisting := keyPairs["testKey"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
						So(keyPairs["testKey"], ShouldEqual, "testGlobalVal2")
					})

					Convey("Setting /config/system/systemtest/testKey should override the global testKey", func() {
						client.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
						client.Set("/config/global/testKey", "testGlobalVal2", 0)
						client.Set("/config/system/systemtest/testKey", "testsystemVal", 0)
						keyPairs := getKeyPairs(client)
						_, isExisting := keyPairs["testKey"]

						So(isExisting, ShouldBeTrue)
						So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
						So(keyPairs["testKey"], ShouldEqual, "testsystemVal")
					})
				})

				Convey("Setting /config/service/systemtest/testserviceKey should not be in the keypair", func() {
					client.Set("/config/service/systemtest/testserviceKey", "testserviceVal", 0)
					keyPairs := getKeyPairs(client)

					_, isExisting := keyPairs["testserviceKey"]
					So(isExisting, ShouldBeFalse)
					So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
					So(keyPairs["testserviceKey"], ShouldBeEmpty)

					client.Delete("/config", true)
				})

				Convey("Testing nested keys", func() {
					Convey("Adding key-value pairs in systemtest root /config/system/systemtest/", func() {
						client.Set("/config/system/systemtest/nestkey1", "nestval1", 0)
						client.Set("/config/system/systemtest/nestkey2", "nestval2", 0)
						keyPairs := getKeyPairs(client)

						_, isExisting := keyPairs["nestkey1"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["nestkey1"], ShouldEqual, "nestval1")

						_, isExisting = keyPairs["nestkey2"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["nestkey2"], ShouldEqual, "nestval2")

						Convey("Adding key-value pairs in systemtest first nest directory /config/system/systemtest/nest1/", func() {
							client.Set("/config/system/systemtest/nest1/nest1key1", "nest1val1", 0)
							client.Set("/config/system/systemtest/nest1/nest1key2", "nest1val2", 0)
							client.Set("/config/system/systemtest/nest2/nest2key1", "nest2val1", 0)
							client.Set("/config/system/systemtest/nest2/nest2key2", "nest2val2", 0)
							keyPairs = getKeyPairs(client)

							_, isExisting = keyPairs["nest1_nest1key1"]
							So(isExisting, ShouldBeTrue)
							So(keyPairs["nest1_nest1key1"], ShouldEqual, "nest1val1")

							_, isExisting = keyPairs["nest1_nest1key2"]
							So(isExisting, ShouldBeTrue)
							So(keyPairs["nest1_nest1key2"], ShouldEqual, "nest1val2")

							_, isExisting = keyPairs["nest2_nest2key1"]
							So(isExisting, ShouldBeTrue)
							So(keyPairs["nest2_nest2key1"], ShouldEqual, "nest2val1")

							_, isExisting = keyPairs["nest1_nest1key2"]
							So(isExisting, ShouldBeTrue)
							So(keyPairs["nest2_nest2key2"], ShouldEqual, "nest2val2")

							Convey("Adding key-value pairs in systemtest second nest directory /config/system/systemtest/nest1/nest2", func() {
								client.Set("/config/system/systemtest/nest1/nest2/nest2key1", "nest2val1", 0)
								client.Set("/config/system/systemtest/nest1/nest2/nest2key2", "nest2val2", 0)
								keyPairs = getKeyPairs(client)

								_, isExisting = keyPairs["nest1_nest2_nest2key1"]
								So(isExisting, ShouldBeTrue)
								So(keyPairs["nest1_nest2_nest2key1"], ShouldEqual, "nest2val1")

								_, isExisting = keyPairs["nest1_nest2_nest2key1"]
								So(isExisting, ShouldBeTrue)
								So(keyPairs["nest1_nest2_nest2key2"], ShouldEqual, "nest2val2")

								client.Delete("/config", true)
							})
						})
					})
				})
			})
		})
	})
}
