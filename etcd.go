package main

import (
	"bytes"
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
	"{{.Prefix}}/global",
	"{{if .System}}{{.Prefix}}/system/{{.System}}{{end}}",
	"{{if .Service}}{{.Prefix}}/service/{{.Service}}{{end}}",
	"{{if .Hostname}}{{.Prefix}}/host/{{.Hostname}}{{end}}",
}

type etcdKeyData struct {
	Prefix, System, Service, Hostname string
}

// KeyPairs is a slice of KeyPair pointers
type KeyPairs map[string]string

func getEndpoints(c *cli.Context) ([]string, error) {
	eps := c.GlobalStringSlice("peers")
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
		CAFile:   c.GlobalString("ca-file"),
		CertFile: c.GlobalString("cert-file"),
		KeyFile:  c.GlobalString("key-file"),
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

func getKeyPairs(c *cli.Context, client *etcd.Client) KeyPairs {
	const noSort = false
	const recursive = true

	tmplData := etcdKeyData{
		Prefix:   c.GlobalString("prefix"),
		System:   c.GlobalString("system"),
		Service:  c.GlobalString("service"),
		Hostname: c.GlobalString("hostname"),
	}

	keyPairs := make(KeyPairs)

	for i, tmpl := range etcdKeyTemplates {
		t := template.Must(template.New("etcdKey" + strconv.Itoa(i)).Parse(tmpl))
		var buf bytes.Buffer
		err := t.Execute(&buf, tmplData)
		if err != nil {
			log.Println("error executing template:", err)
			continue
		}

		if buf.Len() == 0 {
			continue
		}

		dir := buf.String()
		resp, err := client.Get(dir, noSort, recursive)
		if err != nil {
			log.Println(err)
			continue
		}

		addKeyPair(keyPairs, dir, resp.Node)
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
