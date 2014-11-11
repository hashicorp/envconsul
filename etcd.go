package main

import (
	"net/http"
	"net/url"

	"github.com/codegangsta/cli"
	"github.com/coreos/go-etcd/etcd"
	"github.com/zvelo/envetcd/transport"
)

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
