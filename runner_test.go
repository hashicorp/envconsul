package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/consul-template/config"
	"github.com/hashicorp/consul-template/dependency"
)

func TestRunner_appendSecrets(t *testing.T) {
	t.Parallel()

	secretValue := "somevalue"

	tt := []struct {
		name     string
		path     string
		data     *dependency.Secret
		notFound bool
	}{
		{
			"kv1_secret",
			"kv/bar",
			&dependency.Secret{
				Data: map[string]interface{}{
					"key_field": secretValue,
				},
			},
			false,
		},
		{
			"kv2_secret",
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
		{
			"kv2_secret_destroyed",
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

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
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

func TestRunner_configEnv(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name      string
		env       map[string]string
		pristine  bool
		custom    []string
		whitelist []string
		blacklist []string
		output    map[string]string
	}{
		// pristine env with no custom vars leads to empty env
		{
			name:     "pristine without custom vars",
			env:      map[string]string{"PATH": "/bin"},
			pristine: true,
			output:   map[string]string{},
		},
		// pristine env with custom vars only keeps custom vars
		{
			name:     "pristine with custom vars",
			env:      map[string]string{"PATH": "/bin"},
			pristine: true,
			custom:   []string{"GOPATH=/usr/go"},
			output:   map[string]string{"GOPATH": "/usr/go"},
		},
		// custom vars overwrite input vars
		{
			name:   "custom and dependency env",
			env:    map[string]string{"PATH": "/bin"},
			custom: []string{"PATH=/usr/bin"},
			output: map[string]string{"PATH": "/usr/bin"},
		},
		// whitelist filters input by key
		{
			name:      "whitelist only",
			env:       map[string]string{"GOPATH": "/usr/go", "GO111MODULES": "true", "PATH": "/bin"},
			whitelist: []string{"GO*"},
			output:    map[string]string{"GOPATH": "/usr/go", "GO111MODULES": "true"},
		},
		// blacklist takes precedence over whitelist
		{
			name:      "whitelist and blacklist",
			env:       map[string]string{"GOPATH": "/usr/go", "PATH": "/bin", "EDITOR": "vi"},
			whitelist: []string{"GO*", "EDITOR"},
			blacklist: []string{"GO*"},
			output:    map[string]string{"EDITOR": "vi"},
		},
		// custom attached after white/black list rules are applied
		{
			name:      "blacklist and custom",
			env:       map[string]string{"PATH": "/bin", "EDITOR": "vi"},
			blacklist: []string{"EDITOR*"},
			custom:    []string{"EDITOR=nvim"},
			output:    map[string]string{"EDITOR": "nvim", "PATH": "/bin"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Exec: &config.ExecConfig{
					Env: &config.EnvConfig{
						Pristine:  &tc.pristine,
						Blacklist: tc.blacklist,
						Whitelist: tc.whitelist,
						Custom:    tc.custom,
					},
				},
			}
			c := DefaultConfig().Merge(&cfg)
			r, err := NewRunner(c, true)
			if err != nil {
				t.Fatal(err)
			}

			result := r.applyConfigEnv(tc.env)

			if !reflect.DeepEqual(result, tc.output) {
				t.Fatalf("expected: %v\n got: %v", tc.output, result)
			}
		})
	}
}
