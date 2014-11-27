package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/coreos/go-etcd/etcd"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TODO(jrubin) test recursive keys
func TestEtcd(t *testing.T) {
	Convey("When getting keys from etcd", t, func() {

		weh := os.Getenv("WERCKER_ETCD_HOST")
		fmt.Println("WERCKER_ETCD_HOST", weh)
		zeh := os.Getenv("ZVELO_ETCD_HOST")
		fmt.Println("ZVELO_ETCD_HOST", zeh)

		os.Setenv("ENVETCD_NO_SANITIZE", "true")
		os.Setenv("ENVETCD_NO_UPCASE", "true")
		//os.Setenv("ZVELO_ETCD_HOST", "127.0.0.1")

		appTest.Name = "testApp"
		appTest.Author = "Karl Dominguez"
		appTest.Email = "kdominguez@zvelo.com"
		appTest.Version = "0.0.4"
		appTest.Usage = "get environment variables from etcd"
		appTest.Flags = []cli.Flag{
			cli.StringSliceFlag{
				Name:   "peers, C",
				EnvVar: "ENVETCD_PEERS",
				Value:  &cli.StringSlice{werckerAdd()},
				Usage:  "a comma-delimited list of machine addresses in the cluster (default: \"127.0.0.1:4001\")",
			},
			cli.StringFlag{
				Name:   "ca-file",
				EnvVar: "ENVETCD_CA_FILE",
				Usage:  "certificate authority file",
			},
			cli.StringFlag{
				Name:   "cert-file",
				EnvVar: "ENVETCD_CERT_FILE",
				Usage:  "tls client certificate file",
			},
			cli.StringFlag{
				Name:   "key-file",
				EnvVar: "ENVETCD_KEY_FILE",
				Usage:  "tls client key file",
			},
			cli.StringFlag{
				Name:   "hostname",
				EnvVar: "HOSTNAME",
				Value:  "env",
				Usage:  "computer hostname for host specific configuration",
			},
			cli.StringFlag{
				Name:   "system",
				EnvVar: "ENVETCD_SYSTEM",
				Value:  "systemtest",
				Usage:  "system name for system specific configuration",
			},
			cli.StringFlag{
				Name:   "service",
				EnvVar: "ENVETCD_SERVICE",
				Value:  "servicetest",
				Usage:  "service name for service specific configuration",
			},
			cli.StringFlag{
				Name:   "prefix",
				EnvVar: "ENVETCD_PREFIX",
				Value:  "/config",
				Usage:  "etcd prefix for all keys",
			},
			cli.StringFlag{
				Name:   "log-level, l",
				EnvVar: "ENVETCD_LOG_LEVEL",
				Value:  "DEBUG",
				Usage:  "set log level (DEBUG, INFO, WARN, ERR)",
			},
			cli.StringFlag{
				Name:   "output, o",
				Value:  "testOut.txt",
				EnvVar: "ENVETCD_OUTPUT",
				Usage:  "write stdout from the command to this file",
			},
			cli.BoolFlag{
				Name:   "no-sync",
				EnvVar: "ENVETCD_NO_SYNC",
				Usage:  "don't synchronize cluster information before sending request",
			},
			cli.BoolFlag{
				Name:   "clean-env, c",
				EnvVar: "ENVETCD_CLEAN_ENV",
				Usage:  "don't inherit any environment variables other than those pulled from etcd",
			},
			cli.BoolFlag{
				Name:   "no-sanitize",
				EnvVar: "ENVETCD_NO_SANITIZE",
				Usage:  "don't remove bad characters from environment keys",
			},
			cli.BoolFlag{
				Name:   "no-upcase",
				EnvVar: "ENVETCD_NO_UPCASE",
				Usage:  "don't convert all environment keys to uppercase",
			},
		}
		appTest.Action = run

		set := flagSet(appTest.Name, appTest.Flags)
		ctx := cli.NewContext(appTest, set, set)

		etcdConf := newEtcdConfig(ctx)
		etcdClient, err := getClient(etcdConf)

		etcdAddress := fmt.Sprintf("%s%s", "http://", werckerPeer)
		etcdPeer := etcd.NewClient([]string{etcdAddress})
		etcdPeer.Delete("/config", true)
		etcdPeer.SetDir("/config/system/systemtest", 0)
		etcdPeer.SetDir("/config/service/servicetest", 0)
		etcdPeer.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
		etcdPeer.Set("/config/host/env", "", 0)

		Convey("newEtcdConfig should return an etcd config", func() {
			So(etcdConf.Key.Prefix, ShouldEqual, "/config")
			So(etcdConf.Key.Hostname, ShouldEqual, "env")
			So(etcdConf.Sync, ShouldBeTrue)
			So(etcdConf.Key.System, ShouldEqual, "systemtest")
			So(etcdConf.Key.Service, ShouldEqual, "servicetest")
			//So(etcdConf.Peers, ShouldContain, "http://127.0.0.1:4001")

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
