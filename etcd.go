package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	"github.com/zvelo/envetcd/transport"
)

func getPeersFlagValue(config *Config) []string {
	peerstr := config.Etcd

	// Use an environment variable if nothing was supplied on the
	// command line
	if peerstr == "" {
		peerstr = os.Getenv("ENVETCD_PEERS")
	}

	// If we still don't have peers, use a default
	if peerstr == "" {
		peerstr = "127.0.0.1:4001"
	}

	return strings.Split(peerstr, ",")
}

func getEndpoints(config *Config) ([]string, error) {
	eps := getPeersFlagValue(config)
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

func getTransport(config *Config) (*http.Transport, error) {
	tls := transport.TLSInfo{
		CAFile:   config.CAFile,
		CertFile: config.CertFile,
		KeyFile:  config.KeyFile,
	}
	return transport.NewTransport(tls)

}

func (cli *CLI) getClient(config *Config) (*etcd.Client, error) {
	endpoints, err := getEndpoints(config)
	if err != nil {
		return nil, err
	}

	tr, err := getTransport(config)
	if err != nil {
		return nil, err
	}

	client := etcd.NewClient(endpoints)
	client.SetTransport(tr)

	return client, nil
}
