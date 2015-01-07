package main

import (
	"bytes"
	"errors"
	"log"
	"strconv"
	"strings"
	"text/template"

	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
)

// order of precedence:
// global < system < service < host

var etcdKeyTemplates = [...]string{
	"{{.Prefix}}/global",
	"{{if .System}}{{.Prefix}}/system/{{.System}}{{end}}",
	"{{if .Service}}{{.Prefix}}/service/{{.Service}}{{end}}",
	"{{if .Hostname}}{{.Prefix}}/host/{{.Hostname}}{{end}}",
}

// KeyPairs is a slice of KeyPair pointers
type KeyPairs map[string]string

func dumpCURL(client *etcd.Client) {
	client.OpenCURL()
	for {
		log.Printf("[DEBUG] (etcd) Curl-Example: %s\n", client.RecvCURL())
	}
}

func getClient() (*etcd.Client, error) {
	tr, err := transport.NewTransport(config.TLS)
	if err != nil {
		return nil, err
	}

	client := etcd.NewClient(config.Peers)
	client.SetTransport(tr)

	go dumpCURL(client)

	// Sync cluster.
	if config.Sync {
		log.Printf("[DEBUG] (etcd) synchronizing cluster")
		if ok := client.SyncCluster(); !ok {
			return nil, errors.New("cannot sync with the cluster using endpoints " + strings.Join(config.Peers, ", "))
		}
	}

	log.Printf("[DEBUG] (etcd) Cluster-Endpoints: %s\n", strings.Join(client.GetCluster(), ", "))

	return client, nil
}

// getKeyPairs takes a given config and client, and returns all key pairs
func getKeyPairs(client *etcd.Client) KeyPairs {
	const noSort = false
	const recursive = true

	keyPairs := make(KeyPairs)

	// For each etcd directory key template
	// fetch from etcd and add to keypairs
	for i, tmpl := range etcdKeyTemplates {
		t := template.Must(template.New("etcdKey" + strconv.Itoa(i)).Parse(tmpl))
		var buf bytes.Buffer
		err := t.Execute(&buf, config)
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

	// Expose --service --system and --hostname command variables to the child process' emvironment,
	// ignoring if these are not set in etcd (or set in etcd. Command line arguments take precedence.)
	if "" != config.Service {
		log.Printf("[DEBUG] cli: --service %v", config.Service)
		keyPairs["ENVETCD_SERVICE"] = config.Service
	}
	if "" != config.System {
		log.Printf("[DEBUG] cli: --system %v", config.System)
		keyPairs["ENVETCD_SYSTEM"] = config.System
	}
	if "" != config.Hostname {
		log.Printf("[DEBUG] cli: --hostname %v", config.Hostname)
		keyPairs["ENVETCD_HOSTNAME"] = config.Hostname
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
	if key == "" {
		log.Printf("[WARN] (etcd) Key empty for value %v (missing a subdirectory?). Skipping this key.", node.Value)
		return
	}
	key = strings.TrimLeft(key, "/")         // strip any leading slashes
	key = strings.Replace(key, "/", "_", -1) // convert any remaining slashes to underscores (for any nested keys)
	keyPairs[key] = node.Value
}
