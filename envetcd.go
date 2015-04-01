package envetcd

import (
	"bytes"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
)

// order of precedence:
// global < system < service < host

// Regexp for invalid characters in keys
var invalidRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// Config contains all of the parameters needed to run GetKeyPairs
type Config struct {
	Peers    []string
	Sync     bool
	TLS      *transport.TLSInfo
	System   string
	Service  string
	Hostname string
	Prefix   string
	Sanitize bool
	Upcase   bool
}

var etcdKeyTemplates = [...]string{
	"{{.Prefix}}/global",
	"{{if .System}}{{.Prefix}}/system/{{.System}}{{end}}",
	"{{if .Service}}{{.Prefix}}/service/{{.Service}}{{end}}",
	"{{if .Hostname}}{{.Prefix}}/host/{{.Hostname}}{{end}}",
}

// KeyPairs is a slice of KeyPair pointers
type KeyPairs map[string]string

func massagePeers(config *Config) error {
	for i, ep := range config.Peers {

		u, err := url.Parse(ep)
		if err != nil {
			return err
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		config.Peers[i] = u.String()
	}

	return nil
}

func getClient(config *Config) (*etcd.Client, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	if config.TLS == nil {
		config.TLS = &transport.TLSInfo{}
	}

	tr, err := transport.NewTransport(*config.TLS)
	if err != nil {
		return nil, err
	}

	if err := massagePeers(config); err != nil {
		return nil, err
	}

	client := etcd.NewClient(config.Peers)
	client.SetTransport(tr)

	// Sync cluster.
	if config.Sync {
		if ok := client.SyncCluster(); !ok {
			return nil, errors.New("cannot sync with the cluster using endpoints " + strings.Join(config.Peers, ", "))
		}
	}

	return client, nil
}

// GetKeyPairs takes a given config and client, and returns all key pairs
func GetKeyPairs(config *Config) (KeyPairs, error) {
	const noSort = false
	const recursive = true

	client, err := getClient(config)
	if err != nil {
		return nil, err
	}

	keyPairs := make(KeyPairs)

	// For each etcd directory key template
	// fetch from etcd and add to keypairs
	for i, tmpl := range etcdKeyTemplates {
		t := template.Must(template.New("etcdKey" + strconv.Itoa(i)).Parse(tmpl))
		var buf bytes.Buffer
		err := t.Execute(&buf, config)
		if err != nil {
			continue
		}

		if buf.Len() == 0 {
			continue
		}

		dir := buf.String()

		resp, err := client.Get(dir, noSort, recursive)
		if err != nil {
			continue
		}

		addKeyPair(config, keyPairs, dir, resp.Node)
	}

	// Expose --service --system and --hostname command variables to the child process' emvironment,
	// ignoring if these are not set in etcd (or set in etcd. Command line arguments take precedence.)
	if "" != config.Service {
		keyPairs["ENVETCD_SERVICE"] = config.Service
	}

	if "" != config.System {
		keyPairs["ENVETCD_SYSTEM"] = config.System
	}

	if "" != config.Hostname {
		keyPairs["ENVETCD_HOSTNAME"] = config.Hostname
	}

	return keyPairs, nil
}

func addKeyPair(config *Config, keyPairs KeyPairs, dir string, node *etcd.Node) {
	if node.Dir {
		for _, child := range node.Nodes {
			addKeyPair(config, keyPairs, dir, child)
		}
		return
	}

	key := strings.TrimPrefix(node.Key, dir) // strip the prefix directory from the key
	if key == "" {
		return
	}

	key = strings.TrimLeft(key, "/")         // strip any leading slashes
	key = strings.Replace(key, "/", "_", -1) // convert any remaining slashes to underscores (for any nested keys)

	if config.Sanitize {
		key = invalidRegexp.ReplaceAllString(key, "_")
	}

	if config.Upcase {
		key = strings.ToUpper(key)
	}

	keyPairs[key] = node.Value
}
