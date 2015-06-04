package envetcd

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
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
	Etcd          *util.EtcdConfig
	Sanitize      bool
	Upcase        bool
	Prefix        string
	System        string
	Service       string
	Hostname      string
	TemplateFiles []string
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

func getEnvSlice(key string) []string {
	ret := []string{}

	vals := os.Getenv(key)
	if len(vals) == 0 {
		return ret
	}

	for _, val := range strings.Split(vals, ",") {
		val = strings.TrimSpace(val)
		if len(val) > 0 {
			ret = append(ret, val)
		}
	}

	return ret
}

func getEnvBool(key string, dflt bool) bool {
	val := os.Getenv(key)
	if len(val) == 0 {
		return dflt
	}

	ret, err := strconv.ParseBool(val)
	if err != nil {
		log.Printf("[WARN] error parsing environment bool ($%s): %s", key, err)
		return dflt
	}

	return ret
}

func getEnvDefault(key, dflt string) string {
	ret := os.Getenv(key)
	if len(ret) == 0 {
		return dflt
	}
	return ret
}

func initLogger() {
	logLevel := "WARN"
	if val := os.Getenv("LOG_LEVEL"); len(val) > 0 {
		logLevel = val
	}

	util.InitLogger(logLevel)
}

// Set modifies the current environment with variables retrieved from etcd. Set
// will not overwrite existing variables.
// On linux systems, the default gateway will be automatically used as the etcd
// endpoint.
// If $ETCD_PEERS is set, it will override the default gateway.
// $ETCD_PEERS should look like "http://127.0.0.1:4001".
// service should be set by the application calling Set and not derived from
// an environment variable.
// Set will also use some other environment variables if they exist.
// $ENVETCD_PREFIX defaults to "/config"
// $HOSTNAME will be honored if it is set.
// An error is returned only if there was an actual error. Inability to
// determine the etcd endpoint as tolerated and not considered an error. In this
// case Set will simply not do anyting and it is up to the application to ensure
// that it has been configured properly through other means such as environment
// variables or command line flags.
func Set(service string) error {
	initLogger()

	useDefaultGateway := getEnvBool("ETCD_USE_DEFAULT_GATEWAY", true)

	peers := getEnvSlice("ETCD_PEERS")

	if gatewayIP != nil && useDefaultGateway && len(peers) == 0 {
		peers = []string{fmt.Sprintf("http://%s:4001", gatewayIP.String())}
	} else {
		useDefaultGateway = false
	}

	if len(peers) == 0 {
		log.Println("[INFO] envetcd.Set returned after it could not determine the etcd endpoint")
		return nil
	}

	config := &Config{
		Etcd: &util.EtcdConfig{
			Peers:             peers,
			Sync:              !getEnvBool("ETCD_NO_SYNC", false),
			UseDefaultGateway: useDefaultGateway,
		},
		Sanitize:      true,
		Upcase:        true,
		Prefix:        getEnvDefault("ENVETCD_PREFIX", "/config"),
		Service:       service,
		Hostname:      os.Getenv("HOSTNAME"),
		TemplateFiles: getEnvSlice("ENVETCD_TEMPLATES"),
	}

	if len(config.Etcd.Peers[0]) == 0 {
		config.Etcd.Peers[0] = "http://127.0.0.1:4001"
	}

	keyPairs, err := GetKeyPairs(config)
	if err != nil {
		return err
	}

	etcdPeers := strings.Join(peers, ", ")
	log.Printf("[DEBUG] envetcd: %v => %v\n", "ETCD_PEERS", etcdPeers)
	keyPairs["ETCD_PEERS"] = etcdPeers

	for key, value := range keyPairs {
		if len(os.Getenv(key)) == 0 {
			os.Setenv(key, value)
		}
	}

	return nil
}

func processTemplates(keyPairs KeyPairs, templateFiles []string) {
	const ext = ".tmpl"

	for _, tmpl := range templateFiles {
		if filepath.Ext(tmpl) != ext {
			log.Printf("[WARN] template file does not end with '.tmpl' (%s)", tmpl)
			continue
		}

		outFileName := tmpl[0 : len(tmpl)-len(ext)]
		outFile, err := os.Create(outFileName)
		if err != nil {
			log.Printf("[WARN] error creating file (%s): %s", outFileName, err)
			continue
		}

		tpl, err := template.ParseFiles(tmpl)
		if err != nil {
			log.Printf("[WARN] error parsing template (%s): %s", tmpl, err)
			continue
		}

		log.Printf("[INFO] updating template %s", tmpl)
		if err := tpl.Execute(outFile, keyPairs); err != nil {
			log.Printf("[WARN] error writing template file (%s): %s", outFileName, err)
			continue
		}
	}
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

	if config.Etcd.UseDefaultGateway && len(config.Etcd.Peers) == 1 {
		keyPairs["ENVETCD_DEFAULT_GATEWAY"] = config.Etcd.Peers[0]
	}

	var keys []string
	for key := range keyPairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		log.Printf("[DEBUG] envetcd: %v => %v\n", key, keyPairs[key])
	}

	processTemplates(keyPairs, config.TemplateFiles)

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
