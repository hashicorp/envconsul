package main

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"

	"github.com/codegangsta/cli"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
)

// order of precedence:
// global < system < service < host

var etcdKeyTemplates = [...]string{
	"{{.Key.Prefix}}/global",
	"{{if .Key.System}}{{.Key.Prefix}}/system/{{.Key.System}}{{end}}",
	"{{if .Key.Service}}{{.Key.Prefix}}/service/{{.Key.Service}}{{end}}",
	"{{if .Key.Hostname}}{{.Key.Prefix}}/host/{{.Key.Hostname}}{{end}}",
}

type keyData struct {
	Prefix, System, Service, Hostname string
}

type etcdConfig struct {
	Key   keyData
	TLS   transport.TLSInfo
	Peers []string
	Sync  bool
}

func newEtcdConfig(c *cli.Context) *etcdConfig {
	return &etcdConfig{
		Key: keyData{
			Prefix:   c.GlobalString("prefix"),
			System:   c.GlobalString("system"),
			Service:  c.GlobalString("service"),
			Hostname: c.GlobalString("hostname"),
		},
		Sync:  !c.GlobalBool("no-sync"),
		Peers: c.GlobalStringSlice("peers"),
		TLS: transport.TLSInfo{
			CAFile:   c.GlobalString("ca-file"),
			CertFile: c.GlobalString("cert-file"),
			KeyFile:  c.GlobalString("key-file"),
		},
	}
}

// KeyPairs is a slice of KeyPair pointers
type KeyPairs map[string]string

func dumpCURL(client *etcd.Client) {
	client.OpenCURL()
	for {
		log.Printf("[DEBUG] (etcd) Curl-Example: %s\n", client.RecvCURL())
	}
}

func getEndpoints(c *etcdConfig) ([]string, error) {
	eps := c.Peers
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

func getTransport(c *etcdConfig) (*http.Transport, error) {
	return transport.NewTransport(c.TLS)
}

func getClient(c *etcdConfig) (*etcd.Client, error) {
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

	go dumpCURL(client)

	// Sync cluster.
	if c.Sync {
		log.Printf("[DEBUG] (etcd) synchronizing cluster")
		if ok := client.SyncCluster(); !ok {
			handleError(errors.New("cannot sync with the cluster using endpoints "+strings.Join(endpoints, ", ")), exitCodeEtcdError)
		}
	}

	log.Printf("[DEBUG] (etcd) Cluster-Endpoints: %s\n", strings.Join(client.GetCluster(), ", "))

	return client, nil
}

func getKeyPairs(c *etcdConfig, client *etcd.Client) KeyPairs {
	const noSort = false
	const recursive = true

	keyPairs := make(KeyPairs)

	for i, tmpl := range etcdKeyTemplates {
		t := template.Must(template.New("etcdKey" + strconv.Itoa(i)).Parse(tmpl))
		var buf bytes.Buffer
		err := t.Execute(&buf, c)
		if err != nil {
			log.Printf("[ERR] error executing template: %s", err.Error())
			continue
		}

		if buf.Len() == 0 {
			continue
		}

		dir := buf.String()

		log.Printf("[DEBUG] (etcd) querying etcd dir: %s, sort: %t, recursive: %t", dir, noSort, recursive)
		resp, err := client.Get(dir, noSort, recursive)
		if err != nil {
			log.Printf("[ERR] (etcd) %s", err.Error())
			continue
		}

		oldLen := len(keyPairs)
		addKeyPair(keyPairs, dir, resp.Node)
		log.Printf("[DEBUG] etcd returned %d key pairs", len(keyPairs)-oldLen)
	}

	return keyPairs
}

func addKeyPair(keyPairs KeyPairs, dir string, node *etcd.Node) {
	if node.Dir {
		for _, child := range node.Nodes {
			addKeyPair(keyPairs, dir, child)
		}

		return
	}

	key := strings.TrimPrefix(node.Key, dir) // strip the prefix directory from the key
	key = strings.TrimLeft(key, "/")         // strip any leading slashes
	key = strings.Replace(key, "/", "_", -1) // convert any remaining slashes to underscores (for any nested keys)

	keyPairs[key] = node.Value
}
