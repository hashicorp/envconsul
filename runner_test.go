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

	secretValue1 := "somevalue"
	secretValue2 := "somevalue2"

	cases := map[string]struct {
		path     string
		data     *dependency.Secret
		notFound bool
	}{
		"kv1_secret": {
			"kv/bar",
			&dependency.Secret{
				Data: map[string]interface{}{
					"key_field":  secretValue1,
					"key_field2": secretValue2,
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
						"key_field":  secretValue1,
						"key_field2": secretValue2,
					},
				},
			},
			false,
		},
		"kv2_secret_destroyed": {
			"secret/data/foo",
			&dependency.Secret{
				Data: map[string]interface{}{
					"metadata": map[string]interface{}{
						"destroyed": bool(true),
						"version":   "2",
					},
					"data": nil,
				},
			},
			true,
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

			if len(env) > 2 {
				t.Fatalf("Expected only 2 values in this test")
			}

			keyName1 := tc.path + "_key_field"
			keyName1 = strings.Replace(keyName1, "/", "_", -1)

			var value string
			value, ok := env[keyName1]
			if !ok && !tc.notFound {
				t.Fatalf("expected (%s) key, but was not found", keyName1)
			}
			if ok && tc.notFound {
				t.Fatalf("expected to not find key, but (%s) was found", keyName1)
			}
			if ok && value != secretValue1 {
				t.Fatalf("values didn't match, expected (%s), got (%s)", secretValue1, value)
			}

			keyName2 := tc.path + "_key_field2"
			keyName2 = strings.Replace(keyName2, "/", "_", -1)

			value, ok = env[keyName2]
			if !ok && !tc.notFound {
				t.Fatalf("expected (%s) key, but was not found", keyName2)
			}
			if ok && tc.notFound {
				t.Fatalf("expected to not find key, but (%s) was found", keyName2)
			}
			if ok && value != secretValue2 {
				t.Fatalf("values didn't match, expected (%s), got (%s)", secretValue2, value)
			}
		})
	}
}
