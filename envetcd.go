package envetcd

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/coreos/go-etcd/etcd"
	"github.com/zvelo/zvelo-services/util"
)

// order of precedence:
// global < system < service < host

// Regexp for invalid characters in keys
var invalidRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// Config contains all of the parameters needed to run GetKeyPairs
type Config struct {
	Etcd              *util.EtcdConfig
	Sanitize          bool
	Upcase            bool
	UseDefaultGateway bool
	Prefix            string
	System            string
	Service           string
	Hostname          string
}

var (
	etcdKeyTemplates = [...]string{
		"{{.Prefix}}/global",
		"{{if .System}}{{.Prefix}}/system/{{.System}}{{end}}",
		"{{if .Service}}{{.Prefix}}/service/{{.Service}}{{end}}",
		"{{if .Hostname}}{{.Prefix}}/host/{{.Hostname}}{{end}}",
	}

	gatewayIP *net.IP
)

// KeyPairs is a slice of KeyPair pointers
type KeyPairs map[string]string

func init() {
	if ip, err := util.DefaultRoute(); err != nil {
		log.Printf("[INFO] envetcd error getting default gateway: %v\n", err)
	} else {
		gatewayIP = &ip
	}
}

// Set modifies the current environment with variables retrieved from etcd. Set
// will not overwrite existing variables.
// On linux systems, the default gateway will be automatically used as the etcd
// endpoint.
// If $ETCD_ENDPOINT is set, it will override the default gateway.
// $ETCD_ENDPOINT should look like "http://127.0.0.1:4001".
// service should be set by the application calling Set and not derived from
// an environment variable.
// Set will also use some other environment variables if they exist.
// $ETCD_PREFIX defaults to "/config"
// $HOSTNAME will be honored if it is set.
// An error is returned only if there was an actual error. Inability to
// determine the etcd endpoint as tolerated and not considered an error. In this
// case Set will simply not do anyting and it is up to the application to ensure
// that it has been configured properly through other means such as environment
// variables or command line flags.
func Set(service string) error {
	etcdEndpoint := os.Getenv("ETCD_ENDPOINT")

	useSync := true
	if len(os.Getenv("ETCD_NO_SYNC")) > 0 {
		useSync = false
	}
	useDefaultGateway := true
	if len(os.Getenv("ENVETCD_USE_DEFAULT_GATEWAY")) > 0 {
		if val, err := strconv.ParseBool(os.Getenv("ENVETCD_USE_DEFAULT_GATEWAY")); err != nil {
			log.Printf("[INFO] envetcd.Set could not parse $ENVETCD_USE_DEFAULT_GATEWAY, defaulting to true: %v\n", err)
		} else {
			useDefaultGateway = val
		}
	}

	if gatewayIP != nil && useDefaultGateway && len(etcdEndpoint) == 0 {
		etcdEndpoint = fmt.Sprintf("http://%s:4001", gatewayIP.String())
	}

	if len(etcdEndpoint) == 0 {
		log.Println("[INFO] envetcd.Set returned after it could not determine the etcd endpoint")
		return nil
	}

	config := &Config{
		Etcd: &util.EtcdConfig{
			Peers: []string{etcdEndpoint},
			Sync:  useSync,
		},
		Sanitize:          true,
		Upcase:            true,
		UseDefaultGateway: useDefaultGateway,
		Prefix:            os.Getenv("ETCD_PREFIX"),
		Service:           service,
		Hostname:          os.Getenv("HOSTNAME"),
	}

	if len(config.Etcd.Peers[0]) == 0 {
		config.Etcd.Peers[0] = "http://127.0.0.1:4001"
	}

	if len(config.Prefix) == 0 {
		config.Prefix = "/config"
	}

	keyPairs, err := GetKeyPairs(config)
	if err != nil {
		return err
	}

	if keyPairs["LOG_LEVEL"] == "DEBUG" {
		log.Printf("[DEBUG] envetcd: %v => %v\n", "ETCD_ENDPOINT", etcdEndpoint)
	}
	keyPairs["ETCD_ENDPOINT"] = etcdEndpoint

	for key, value := range keyPairs {
		if len(os.Getenv(key)) == 0 {
			os.Setenv(key, value)
		}
	}

	return nil
}

// GetKeyPairs takes a given config and client, and returns all key pairs
func GetKeyPairs(config *Config) (KeyPairs, error) {
	const noSort = false
	const recursive = true

	if len(config.Service) > 0 && len(config.System) == 0 {
		if index := strings.Index(config.Service, "-"); index != -1 {
			config.System = config.Service[:index]
		}
	}

	client, err := util.GetEtcdClient(config.Etcd)
	if err != nil {
		return nil, err
	}

	keyPairs := KeyPairs{}

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

	keyPairs["ETCD_PEERS"] = strings.Join(client.GetCluster(), ", ")

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

	if config.UseDefaultGateway && gatewayIP != nil {
		keyPairs["ENVETCD_DEFAULT_GATEWAY"] = gatewayIP.String()
	}

	var keys []string
	for key := range keyPairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		log.Printf("[DEBUG] envetcd: %v => %v\n", key, keyPairs[key])
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
