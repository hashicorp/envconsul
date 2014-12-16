package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/coreos/go-etcd/etcd"

	. "github.com/smartystreets/goconvey/convey"
)

// TODO(jrubin) test recursive keys
func TestEtcd(t *testing.T) {
	Convey("When getting keys from etcd", t, func() {

		os.Setenv("ENVETCD_NO_SANITIZE", "true")
		os.Setenv("ENVETCD_NO_UPCASE", "true")

		appTest.Name = "testApp"
		appTest.Author = "Karl Dominguez"
		appTest.Email = "kdominguez@zvelo.com"
		appTest.Version = version
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

		// Craete the new cli contet with test flags
		set := flagSet(appTest.Name, appTest.Flags)
		ctx := cli.NewContext(appTest, set, set)

		// get config (keydata, TLS, and peers etc) from context by mapping flags into
		// a useful config struct.
		etcdConf := newEtcdConfig(ctx)

		etcdAddress := fmt.Sprintf("%s%s", "http://", werckerPeer)
		etcdPeer := etcd.NewClient([]string{etcdAddress})
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

			Convey("getEndpoints should return an array of end points", func() {
				endPoints, err := getEndpoints(etcdConf)
				So(err, ShouldBeNil)
				So(endPoints, ShouldNotBeEmpty)
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
				etcdClient, err := getClient(etcdConf)
				So(err, ShouldBeNil)
				So(etcdClient, ShouldNotBeEmpty)
				So(etcdClient.CheckRetry, ShouldBeNil)

				Convey("getKeyPairs returns keypairs", func() {
					keyPairs := getKeyPairs(etcdConf, etcdPeer)
					So(keyPairs, ShouldNotBeEmpty)

				})
				Convey("Testing override keys", func() {

					Convey("Setting /config/global testkey only", func() {
						etcdPeer.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
						keyPairs := getKeyPairs(etcdConf, etcdPeer)

						_, isExisting := keyPairs["systemtest_testKey"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")

					})

					Convey("Setting /config/global/testKey", func() {
						etcdPeer.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
						etcdPeer.Set("/config/global/testKey", "testGlobalVal2", 0)
						keyPairs := getKeyPairs(etcdConf, etcdPeer)

						_, isExisting := keyPairs["testKey"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
						So(keyPairs["testKey"], ShouldEqual, "testGlobalVal2")
					})
					Convey("Setting /config/system/systemtest/testKey should override the global testKey", func() {
						etcdPeer.Set("/config/global/systemtest/testKey", "globaltestVal", 0)
						etcdPeer.Set("/config/global/testKey", "testGlobalVal2", 0)
						etcdPeer.Set("/config/system/systemtest/testKey", "testsystemVal", 0)
						keyPairs := getKeyPairs(etcdConf, etcdPeer)
						_, isExisting := keyPairs["testKey"]

						So(isExisting, ShouldBeTrue)
						So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
						So(keyPairs["testKey"], ShouldEqual, "testsystemVal")
					})
				})

				Convey("Setting /config/service/systemtest/testserviceKey should not be in the keypair", func() {
					etcdPeer.Set("/config/service/systemtest/testserviceKey", "testserviceVal", 0)
					keyPairs := getKeyPairs(etcdConf, etcdPeer)

					_, isExisting := keyPairs["testserviceKey"]
					So(isExisting, ShouldBeFalse)
					So(keyPairs["systemtest_testKey"], ShouldEqual, "globaltestVal")
					So(keyPairs["testserviceKey"], ShouldBeEmpty)

					fmt.Println("\nCurrent Directories\n")
					resp2, err := etcdPeer.Get("/config/", false, true)
					if err != nil {
						fmt.Println(err)
					}
					printLs(resp2)
					etcdPeer.Delete("/config", true)
				})

				Convey("Testing nested keys", func() {
					Convey("Adding key-value pairs in systemtest root /config/system/systemtest/", func() {
						etcdPeer.Set("/config/system/systemtest/nestkey1", "nestval1", 0)
						etcdPeer.Set("/config/system/systemtest/nestkey2", "nestval2", 0)
						keyPairs := getKeyPairs(etcdConf, etcdPeer)

						_, isExisting := keyPairs["nestkey1"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["nestkey1"], ShouldEqual, "nestval1")

						_, isExisting = keyPairs["nestkey2"]
						So(isExisting, ShouldBeTrue)
						So(keyPairs["nestkey2"], ShouldEqual, "nestval2")

						Convey("Adding key-value pairs in systemtest first nest directory /config/system/systemtest/nest1/", func() {
							etcdPeer.Set("/config/system/systemtest/nest1/nest1key1", "nest1val1", 0)
							etcdPeer.Set("/config/system/systemtest/nest1/nest1key2", "nest1val2", 0)
							etcdPeer.Set("/config/system/systemtest/nest2/nest2key1", "nest2val1", 0)
							etcdPeer.Set("/config/system/systemtest/nest2/nest2key2", "nest2val2", 0)
							keyPairs = getKeyPairs(etcdConf, etcdPeer)

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
								etcdPeer.Set("/config/system/systemtest/nest1/nest2/nest2key1", "nest2val1", 0)
								etcdPeer.Set("/config/system/systemtest/nest1/nest2/nest2key2", "nest2val2", 0)
								keyPairs = getKeyPairs(etcdConf, etcdPeer)

								_, isExisting = keyPairs["nest1_nest2_nest2key1"]
								So(isExisting, ShouldBeTrue)
								So(keyPairs["nest1_nest2_nest2key1"], ShouldEqual, "nest2val1")

								_, isExisting = keyPairs["nest1_nest2_nest2key1"]
								So(isExisting, ShouldBeTrue)
								So(keyPairs["nest1_nest2_nest2key2"], ShouldEqual, "nest2val2")

								fmt.Println("\nCurrent Directories\n")
								resp2, err := etcdPeer.Get("/config/", false, true)
								if err != nil {
									fmt.Println(err)
								}
								printLs(resp2)
								etcdPeer.Delete("/config", true)
							})
						})
					})
				})
			})
		})
	})
}

func printLs(resp *etcd.Response) {
	if !resp.Node.Dir {
		fmt.Println(resp.Node.Key)
	}
	for _, node := range resp.Node.Nodes {
		rPrint(node)
	}
}

// rPrint recursively prints out the nodes in the node structure.
func rPrint(n *etcd.Node) {

	if n.Dir {
		fmt.Println(n.Key)
	}

	for _, node := range n.Nodes {
		rPrint(node)
	}
}
