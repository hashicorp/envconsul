package envetcd

import (
	"os"
	"testing"

	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
	"github.com/zvelo/zvelo-services/util"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	config *Config
)

func init() {
	util.InitLogger("WARN")

	// $ETCD_ENDPOINT should look like "http://127.0.0.1:4001"

	config = &Config{
		Etcd: &util.EtcdConfig{
			Peers: []string{os.Getenv("ETCD_ENDPOINT")},
			Sync:  true,
			TLS:   &transport.TLSInfo{},
		},
		Prefix:   "/config",
		Hostname: "env",
		Service:  "general-servicetest",
	}
}

func TestEtcd(t *testing.T) {
	Convey("When getting keys from etcd", t, func() {
		Convey("ETCD_ENDPOINT environment variable", func() {
			So(os.Getenv("ETCD_ENDPOINT"), ShouldNotBeBlank)
		})

		client := etcd.NewClient(config.Etcd.Peers)
		client.SetDir("/config/system/general", 0)
		client.SetDir("/config/service/general-servicetest", 0)
		client.Set("/config/global/somedir/testKey", "globaltestVal", 0)
		client.Set("/config/host/env", "", 0)

		Convey("config should be valid", func() {
			So(config.Prefix, ShouldEqual, "/config")
			So(config.Hostname, ShouldEqual, "env")
			So(config.Etcd.Sync, ShouldBeTrue)
			// So(config.System, ShouldEqual, "general")
			So(config.Service, ShouldEqual, "general-servicetest")
			So(config.Etcd.Peers, ShouldNotBeEmpty)
			So(config.Etcd.TLS, ShouldNotBeNil)

			etcdClient, err := util.GetEtcdClient(config.Etcd)
			So(err, ShouldBeNil)
			So(etcdClient, ShouldNotBeEmpty)
			So(etcdClient.CheckRetry, ShouldBeNil)

			Convey("GetKeyPairs returns keypairs", func() {
				keyPairs, err := GetKeyPairs(config)
				So(err, ShouldBeNil)
				So(keyPairs, ShouldNotBeEmpty)
			})

			Convey("Testing override keys", func() {
				Convey("Setting /config/global testkey only", func() {
					client.Set("/config/global/somedir/testKey", "globaltestVal", 0)
					keyPairs, err := GetKeyPairs(config)
					So(err, ShouldBeNil)

					_, isExisting := keyPairs["somedir_testKey"]
					So(isExisting, ShouldBeTrue)
					So(keyPairs["somedir_testKey"], ShouldEqual, "globaltestVal")

				})

				Convey("Setting /config/global/testKey", func() {
					client.Set("/config/global/somedir/testKey", "globaltestVal", 0)
					client.Set("/config/global/testKey", "testGlobalVal2", 0)
					keyPairs, err := GetKeyPairs(config)
					So(err, ShouldBeNil)

					_, isExisting := keyPairs["testKey"]
					So(isExisting, ShouldBeTrue)
					So(keyPairs["somedir_testKey"], ShouldEqual, "globaltestVal")
					So(keyPairs["testKey"], ShouldEqual, "testGlobalVal2")
				})

				Convey("Setting /config/system/general/testKey should override the global testKey", func() {
					client.Set("/config/global/somedir/testKey", "globaltestVal", 0)
					client.Set("/config/global/testKey", "testGlobalVal2", 0)
					client.Set("/config/system/general/testKey", "testsystemVal", 0)
					keyPairs, err := GetKeyPairs(config)
					So(err, ShouldBeNil)
					_, isExisting := keyPairs["testKey"]

					So(isExisting, ShouldBeTrue)
					So(keyPairs["somedir_testKey"], ShouldEqual, "globaltestVal")
					So(keyPairs["testKey"], ShouldEqual, "testsystemVal")
				})
			})

			Convey("Setting /config/service/general/testserviceKey should not be in the keypair", func() {
				client.Set("/config/service/general/testserviceKey", "testserviceVal", 0)
				keyPairs, err := GetKeyPairs(config)
				So(err, ShouldBeNil)

				_, isExisting := keyPairs["testserviceKey"]
				So(isExisting, ShouldBeFalse)
				So(keyPairs["somedir_testKey"], ShouldEqual, "globaltestVal")
				So(keyPairs["testserviceKey"], ShouldBeEmpty)

				client.Delete("/config", true)
			})

			Convey("Testing nested keys", func() {
				Convey("Adding key-value pairs in general root /config/system/general/", func() {
					client.Set("/config/system/general/nestkey1", "nestval1", 0)
					client.Set("/config/system/general/nestkey2", "nestval2", 0)
					keyPairs, err := GetKeyPairs(config)
					So(err, ShouldBeNil)

					_, isExisting := keyPairs["nestkey1"]
					So(isExisting, ShouldBeTrue)
					So(keyPairs["nestkey1"], ShouldEqual, "nestval1")

					_, isExisting = keyPairs["nestkey2"]
					So(isExisting, ShouldBeTrue)
					So(keyPairs["nestkey2"], ShouldEqual, "nestval2")

					Convey("Adding key-value pairs in general first nest directory /config/system/general/nest1/", func() {
						client.Set("/config/system/general/nest1/nest1key1", "nest1val1", 0)
						client.Set("/config/system/general/nest1/nest1key2", "nest1val2", 0)
						client.Set("/config/system/general/nest2/nest2key1", "nest2val1", 0)
						client.Set("/config/system/general/nest2/nest2key2", "nest2val2", 0)
						keyPairs, err := GetKeyPairs(config)
						So(err, ShouldBeNil)

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

						Convey("Adding key-value pairs in general second nest directory /config/system/general/nest1/nest2", func() {
							client.Set("/config/system/general/nest1/nest2/nest2key1", "nest2val1", 0)
							client.Set("/config/system/general/nest1/nest2/nest2key2", "nest2val2", 0)
							keyPairs, err := GetKeyPairs(config)
							So(err, ShouldBeNil)

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
}
