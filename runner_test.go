package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/consul-template/config"
	"github.com/hashicorp/consul-template/dependency"
	"github.com/y0ssar1an/q"
)

func TestRunner_appendSecrets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		f    []string
		e    *Config
		path string
		data *dependency.Secret
		err  bool
	}{
		{
			"kv1_secret",
			[]string{"-secret", "kv/bar"},
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("kv/bar"),
					},
				},
			},
			"kv/bar",
			&dependency.Secret{
				Data: map[string]interface{}{
					"key_field": "some_value",
				},
			},
			false,
		},
		{
			"kv2_secret",
			[]string{"-secret", "secret/data/foo"},
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("Secret/data/foo"),
					},
				},
			},
			"secret/data/foo",
			&dependency.Secret{
				Data: map[string]interface{}{
					"metadata": map[string]interface{}{
						"deletion_time": "",
						"destroyed":     bool(false),
						"version":       "1",
						"created_time":  "2018-11-06T16:43:59.705051Z",
					},
					"data": map[string]interface{}{
						"kvVersion": "kv2",
					},
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			c := DefaultConfig().Merge(tc.e)
			r, err := NewRunner(c, true)
			if err != nil {
				t.Fatal(err)
			}
			vrq, err := dependency.NewVaultReadQuery(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			env := make(map[string]string)
			appendError := r.appendSecrets(env, vrq, tc.data)
			if appendError != nil {
				t.Fatalf("got err: %s", appendError)
			}
			q.Q("what is end env:", env)
		})
	}
}
