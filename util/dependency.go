package util

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

// Dependency is an interface
type Dependency interface {
	Fetch(*etcd.Client) (interface{}, *etcd.Response, error)
	HashCode() string
	Key() string
	Display() string
}

/// ------------------------- ///

// KeyPrefixDependency is the representation of a requested key dependency
// from inside a template.
type KeyPrefixDependency struct {
	rawKey string
	Prefix string
}

// Fetch queries the etcd API defined by the given client and returns a slice
// of KeyPair objects
func (d *KeyPrefixDependency) Fetch(client *etcd.Client) (interface{}, *etcd.Response, error) {
	log.Printf("[DEBUG] (%s) querying etcd", d.Display())

	const noSort = false
	const recursive = true

	resp, err := client.Get(d.Prefix, noSort, recursive)
	if err != nil {
		return nil, resp, err
	}

	var keyPairs []*KeyPair

	d.buildKeyPairs(keyPairs, resp.Node)

	return keyPairs, resp, nil
}

func (d *KeyPrefixDependency) buildKeyPairs(keyPairs []*KeyPair, node *etcd.Node) {
	if node.Dir {
		for _, child := range node.Nodes {
			d.buildKeyPairs(keyPairs, child)
		}

		return
	}

	key := strings.TrimPrefix(node.Key, d.Prefix)
	key = strings.TrimLeft(key, "/")

	keyPairs = append(keyPairs, &KeyPair{
		Path:  node.Key,
		Key:   key,
		Value: node.Value,
	})
}

func (d *KeyPrefixDependency) HashCode() string {
	return fmt.Sprintf("KeyPrefixDependency|%s", d.Key())
}

func (d *KeyPrefixDependency) Key() string {
	return d.rawKey
}

func (d *KeyPrefixDependency) Display() string {
	return fmt.Sprintf(`keyPrefix "%s"`, d.rawKey)
}

// ParseKeyDependency parses a string of the format a(/b(/c...))
func ParseKeyPrefixDependency(s string) (*KeyPrefixDependency, error) {
	// TODO(jrubin) get rid of datacenter
	re := regexp.MustCompile(`\A` +
		`(?P<prefix>[[:word:]\.\-\/]+)?` +
		`(@(?P<datacenter>[[:word:]\.\-]+))?` +
		`\z`)
	names := re.SubexpNames()
	match := re.FindAllStringSubmatch(s, -1)

	if len(match) == 0 {
		return nil, errors.New("invalid key prefix dependency format")
	}

	r := match[0]

	m := map[string]string{}
	for i, n := range r {
		if names[i] != "" {
			m[names[i]] = n
		}
	}

	prefix := m["prefix"]

	kpd := &KeyPrefixDependency{
		rawKey: s,
		Prefix: prefix,
	}

	return kpd, nil
}

// KeyPair is a simple Key-Value pair
type KeyPair struct {
	Path  string
	Key   string
	Value string
}
