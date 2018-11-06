package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/consul-template/config"
	"github.com/hashicorp/consul-template/dependency"
)

func TestRunner_appendSecrets(t *testing.T) {
	t.Parallel()

	secretValue := "somevalue"

	cases := map[string]struct {
		path string
		data *dependency.Secret
		err  bool
	}{
		"kv1_secret": {
			"kv/bar",
			&dependency.Secret{
				Data: map[string]interface{}{
					"key_field": secretValue,
				},
			},
			false,
		},
		"kv2_secret": {
			"secret/data/foo",
			&dependency.Secret{
				Data: map[string]interface{}{
					"metadata": map[string]interface{}{
						"destroyed": bool(false),
						"version":   "1",
					},
					"data": map[string]interface{}{
						"key_field": secretValue,
					},
				},
			},
			false,
		},
	}

	for name, tc := range cases {
		t.Run(fmt.Sprintf("%s", name), func(t *testing.T) {
			cfg := Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String(tc.path),
					},
				},
			}
			c := DefaultConfig().Merge(&cfg)
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

			if len(env) > 1 {
				t.Fatalf("Expected only 1 value in this test")
			}

			keyName := tc.path + "_key_field"
			keyName = strings.Replace(keyName, "/", "_", -1)

			var value string
			value, ok := env[keyName]
			if !ok {
				t.Fatalf("expected (%s) key, but was not found", keyName)
			}
			if value != secretValue {
				t.Fatalf("values didn't match, expected (%s), got (%s)", secretValue, value)
			}
		})
	}
}
