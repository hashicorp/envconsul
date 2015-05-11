package main

import (
	"fmt"
	"strings"

	dep "github.com/hashicorp/consul-template/dependency"
)

// prefixVar implements the Flag.Value interface and allows the user
// to specify multiple -prefix keys in the CLI where each option is parsed
// as a dependency.
type prefixVar []*dep.StoreKeyPrefix

func (pv *prefixVar) Set(value string) error {
	value = strings.TrimPrefix(value, "/")
	prefix, err := dep.ParseStoreKeyPrefix(value)
	if err != nil {
		return err
	}

	if *pv == nil {
		*pv = make([]*dep.StoreKeyPrefix, 0, 1)
	}
	*pv = append(*pv, prefix)

	return nil
}

func (pv *prefixVar) String() string {
	list := make([]string, 0, len(*pv))
	for _, prefix := range *pv {
		list = append(list, prefix.Prefix)
	}
	return strings.Join(list, ", ")
}

/// ------------------------- ///

// authVar implements the Flag.Value interface and allows the user to specify
// authentication in the username[:password] form.
type authVar Auth

// Set sets the value for this authentication.
func (a *authVar) Set(value string) error {
	a.Enabled = true

	if strings.Contains(value, ":") {
		split := strings.SplitN(value, ":", 2)
		a.Username = split[0]
		a.Password = split[1]
	} else {
		a.Username = value
	}

	return nil
}

// String returns the string representation of this authentication.
func (a *authVar) String() string {
	if a.Password == "" {
		return a.Username
	}

	return fmt.Sprintf("%s:%s", a.Username, a.Password)
}
