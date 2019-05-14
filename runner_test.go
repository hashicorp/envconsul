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
		path     string
		data     *dependency.Secret
		notFound bool
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
		"int_secret_skipped": {
			"kv/foo",
			&dependency.Secret{
				Data: map[string]interface{}{
					"key_field": 1,
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

			if len(env) > 1 {
				t.Fatalf("Expected only 1 value in this test")
			}

			keyName := tc.path + "_key_field"
			keyName = strings.Replace(keyName, "/", "_", -1)

			var value string
			value, ok := env[keyName]
			if !ok && !tc.notFound {
				t.Fatalf("expected (%s) key, but was not found", keyName)
			}
			if ok && tc.notFound {
				t.Fatalf("expected to not find key, but (%s) was found", keyName)
			}
			if ok && value != secretValue {
				t.Fatalf("values didn't match, expected (%s), got (%s)", secretValue, value)
			}
		})
	}
}

func TestRunner_appendPrefixes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		path     string
		noPrefix bool
		data     []*dependency.KeyPair
		keyName  string
	}{
		{
			name:     "false noprefix appends path",
			path:     "app/my_service",
			noPrefix: false,
			data: []*dependency.KeyPair{
				&dependency.KeyPair{
					Key:   "mykey",
					Value: "myValue",
				},
			},
			keyName: "app_my_service_mykey",
		},
		{
			name:     "true noprefix excludes path",
			path:     "app/my_service",
			noPrefix: true,
			data: []*dependency.KeyPair{
				&dependency.KeyPair{
					Key:   "mykey",
					Value: "myValue",
				},
			},
			keyName: "mykey",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path:     config.String(tc.path),
						NoPrefix: config.Bool(tc.noPrefix),
					},
				},
			}
			c := DefaultConfig().Merge(&cfg)
			r, err := NewRunner(c, true)
			if err != nil {
				t.Fatal(err)
			}
			kvq, err := dependency.NewKVListQuery(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			env := make(map[string]string)
			appendError := r.appendPrefixes(env, kvq, tc.data)
			if appendError != nil {
				t.Fatalf("got err: %s", appendError)
			}

			if len(env) > 1 {
				t.Fatalf("Expected only 1 value in this test")
			}

			var value string
			value, ok := env[tc.keyName]
			if !ok {
				t.Fatalf("expected (%s) key, but was not found", tc.keyName)
			}
			if ok && value != tc.data[0].Value {
				t.Fatalf("values didn't match, expected (%s), got (%s)", tc.data[0].Value, value)
			}
		})
	}
}
