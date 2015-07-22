package util

import (
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	config *EtcdConfig
)

func init() {
	log.SetLevel(log.ErrorLevel)

	// $ETCD_ENDPOINT should look like "http://127.0.0.1:4001"

	config = &EtcdConfig{
		Peers: []string{os.Getenv("ETCD_ENDPOINT")},
		Sync:  true,
		TLS:   &transport.TLSInfo{},
	}
}

func TestEtcd(t *testing.T) {
	Convey("When getting keys from etcd", t, func() {
		Convey("ETCD_ENDPOINT environment variable", func() {
			So(os.Getenv("ETCD_ENDPOINT"), ShouldNotBeBlank)
		})

		client := etcd.NewClient(config.Peers)
		client.SetDir("/config/system/general", 0)
		client.SetDir("/config/service/general-servicetest", 0)
		client.Set("/config/global/somedir/testKey", "globaltestVal", 0)
		client.Set("/config/host/env", "", 0)

		Convey("config should be valid", func() {
			So(config.Sync, ShouldBeTrue)
			// So(config.System, ShouldEqual, "general")
			So(config.Peers, ShouldNotBeEmpty)
			So(config.TLS, ShouldNotBeNil)

			Convey("massagePeers should work", func() {
				peersOrig := config.Peers
				config.Peers = []string{"127.0.0.1:4001", "http://127.0.0.1:4001"}
				defer func() { config.Peers = peersOrig }()

				So(massagePeers(config), ShouldBeNil)
				So(len(config.Peers), ShouldEqual, 2)
				So(config.Peers[0], ShouldEqual, "http://127.0.0.1:4001")
				So(config.Peers[1], ShouldEqual, "http://127.0.0.1:4001")

				config.Peers = []string{":"}
				err := massagePeers(config)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "parse :: missing protocol scheme")

			})

			Convey("GetEtcdClient should return an etcd client based on a given config", func() {
				etcdClient, err := GetEtcdClient(config)
				So(err, ShouldBeNil)
				So(etcdClient, ShouldNotBeEmpty)
				So(etcdClient.CheckRetry, ShouldBeNil)
			})
		})
	})
}
