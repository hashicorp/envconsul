package util

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
)

var (
	// EtcdFlags is a convenience variable for simple etcd cli arguments
	EtcdFlags = []cli.Flag{
		cli.StringSliceFlag{
			Name:   "peers, C",
			EnvVar: "ETCD_PEERS",
			Value:  &cli.StringSlice{},
			Usage:  "a machine address in the etcd cluster (default: \"127.0.0.1:4001\"), may be specified multiple times",
		},
		cli.StringFlag{
			Name:   "ca-file",
			EnvVar: "ETCD_CA_FILE",
			Usage:  "etcd certificate authority file",
		},
		cli.StringFlag{
			Name:   "cert-file",
			EnvVar: "ETCD_CERT_FILE",
			Usage:  "etcd tls client certificate file",
		},
		cli.StringFlag{
			Name:   "key-file",
			EnvVar: "ETCD_KEY_FILE",
			Usage:  "etcd tls client key file",
		},
		cli.BoolFlag{
			Name:   "no-sync",
			EnvVar: "ETCD_NO_SYNC",
			Usage:  "don't synchronize etcd cluster information before watching",
		},
		cli.BoolFlag{
			Name:   "use-default-gateway, d",
			EnvVar: "ENVETCD_USE_DEFAULT_GATEWAY",
			Usage:  "expose the default gateway as $ENVETCD_DEFAULT_GATEWAY",
		},
	}
)

// EtcdConfig includes a peers list and any TLS info.
type EtcdConfig struct {
	TLS               *transport.TLSInfo
	Peers             []string
	Sync              bool
	UseDefaultGateway bool
}

// NewEtcdConfig creates an etcdConfig declaration from the command line context
// provided by cli.  This includes potential TLS files.
func NewEtcdConfig(c *cli.Context) *EtcdConfig {
	ret := &EtcdConfig{
		Sync:              !c.GlobalBool("no-sync"),
		Peers:             c.GlobalStringSlice("peers"),
		UseDefaultGateway: c.GlobalBool("use-default-gateway"),
		TLS: &transport.TLSInfo{
			CAFile:   c.GlobalString("ca-file"),
			CertFile: c.GlobalString("cert-file"),
			KeyFile:  c.GlobalString("key-file"),
		},
	}

	if len(ret.Peers) > 0 {
		ret.UseDefaultGateway = false
	} else {
		if ret.UseDefaultGateway {
			ip, err := DefaultRoute()
			if err != nil {
				log.Printf("[INFO] envetcd error getting default gateway: %v\n", err)
			} else {
				ret.Peers = []string{fmt.Sprintf("http://%s:4001", ip.String())}
			}
		}

		if len(ret.Peers) == 0 {
			ret.UseDefaultGateway = false
			ret.Peers = append(ret.Peers, "127.0.0.1:4001")
		}
	}

	return ret
}

// massagePeers updates the peers listed in the EtcdConfig after
// coercing them to proper URLs
func massagePeers(c *EtcdConfig) error {
	for i, ep := range c.Peers {
		u, err := url.Parse(ep)
		if err != nil {
			return err
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		c.Peers[i] = u.String()
	}

	return nil
}

// GetEtcdClient returns an etcd Client.  Uses endpoints.
func GetEtcdClient(c *EtcdConfig) (*etcd.Client, error) {
	if c == nil {
		return nil, errors.New("config is nil")
	}

	// coerce specified peers into URLs suitable for creating the Client
	if err := massagePeers(c); err != nil {
		return nil, err
	}

	// Create the new client
	client := etcd.NewClient(c.Peers)

	if c.TLS != nil {
		// Create a new transport using keyfiles
		tr, err := transport.NewTransport(*c.TLS)
		if err != nil {
			return nil, err
		}

		client.SetTransport(tr)
	}

	// Perform an initial cluster sync unless configured not to
	if c.Sync {
		log.Printf("[DEBUG] (etcd) synchronizing cluster")
		if ok := client.SyncCluster(); !ok {
			return nil, errors.New("[ERR] cannot sync with the cluster using endpoints \"" + strings.Join(c.Peers, "\", \"") + "\"")
		}
	}

	log.Printf("[DEBUG] (etcd) Cluster-Endpoints: %s\n", strings.Join(client.GetCluster(), ", "))
	return client, nil
}
