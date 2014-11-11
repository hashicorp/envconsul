package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
)

// KeyPair is a simple Key-Value pair
type KeyPair struct {
	Path  string
	Key   string
	Value string
}

// KeyPairs is a slice of KeyPair pointers
type KeyPairs []*KeyPair

func getEndpoints(c *cli.Context) ([]string, error) {
	eps := c.StringSlice("peers")
	for i, ep := range eps {
		u, err := url.Parse(ep)
		if err != nil {
			return nil, err
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		eps[i] = u.String()
	}
	return eps, nil
}

func getTransport(c *cli.Context) (*http.Transport, error) {
	tls := transport.TLSInfo{
		CAFile:   c.String("ca-file"),
		CertFile: c.String("cert-file"),
		KeyFile:  c.String("key-file"),
	}
	return transport.NewTransport(tls)

}

func getClient(c *cli.Context) (*etcd.Client, error) {
	endpoints, err := getEndpoints(c)
	if err != nil {
		return nil, err
	}

	tr, err := getTransport(c)
	if err != nil {
		return nil, err
	}

	client := etcd.NewClient(endpoints)
	client.SetTransport(tr)

	return client, nil
}

func getKeyPairs(client *etcd.Client, prefix string) (KeyPairs, error) {
	const noSort = false
	const recursive = true

	resp, err := client.Get(prefix, noSort, recursive)
	if err != nil {
		return nil, err
	}

	var keyPairs []*KeyPair

	buildKeyPairs(prefix, keyPairs, resp.Node)

	// log.Println("keyPairs", keyPairs)

	return keyPairs, nil
}

func buildKeyPairs(prefix string, keyPairs []*KeyPair, node *etcd.Node) {
	//log.Printf("[DEBUG] (%s) on node %s", d.Display(), node.Key)
	if node.Dir {
		for _, child := range node.Nodes {
			buildKeyPairs(prefix, keyPairs, child)
		}

		return
	}

	key := strings.TrimPrefix(node.Key, prefix)
	key = strings.TrimLeft(key, "/")

	keyPairs = append(keyPairs, &KeyPair{
		Path:  node.Key,
		Key:   key,
		Value: node.Value,
	})
}
