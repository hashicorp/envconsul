package main

import (
	"fmt"
	"reflect"
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
		noPrefix *bool
		data     *dependency.Secret
		keyName  string
		notFound bool
	}{
		{
			name:     "kv1 secret",
			path:     "kv/bar",
			noPrefix: config.Bool(false),
			data: &dependency.Secret{
				Data: map[string]interface{}{
					"bar": secretValue,
				},
			},
			keyName:  "kv_bar_bar",
			notFound: false,
		},
		{
			name:     "kv2 secret",
			path:     "secret/data/foo",
			noPrefix: config.Bool(false),
			data: &dependency.Secret{
				Data: map[string]interface{}{
					"metadata": map[string]interface{}{
						"destroyed": bool(false),
						"version":   "1",
					},
					"data": map[string]interface{}{
						"bar": secretValue,
					},
				},
			},
			keyName:  "secret_data_foo_bar",
			notFound: false,
		},
		{
			name:     "kv2 secret destroyed",
			path:     "secret/data/foo",
			noPrefix: config.Bool(false),
			data: &dependency.Secret{
				Data: map[string]interface{}{
					"metadata": map[string]interface{}{
						"destroyed": bool(true),
						"version":   "2",
					},
					"data": nil,
				},
			},
			keyName:  "",
			notFound: true,
		},
		{
			name:     "kv2 secret noprefix excludes path",
			path:     "secret/data/foo",
			noPrefix: config.Bool(true),
			data: &dependency.Secret{
				Data: map[string]interface{}{
					"metadata": map[string]interface{}{
						"destroyed": bool(false),
						"version":   "1",
					},
					"data": map[string]interface{}{
						"bar": secretValue,
					},
				},
			},
			keyName:  "bar",
			notFound: false,
		},
		{
			name:     "kv2 secret false noprefix includes path",
			path:     "secret/data/foo",
			noPrefix: config.Bool(false),
			data: &dependency.Secret{
				Data: map[string]interface{}{
					"metadata": map[string]interface{}{
						"destroyed": bool(false),
						"version":   "1",
					},
					"data": map[string]interface{}{
						"bar": secretValue,
					},
				},
			},
			keyName:  "secret_data_foo_bar",
			notFound: false,
		},
		{
			name:     "kv2 secret default noprefix includes path",
			path:     "secret/data/foo",
			noPrefix: nil,
			data: &dependency.Secret{
				Data: map[string]interface{}{
					"metadata": map[string]interface{}{
						"destroyed": bool(false),
						"version":   "1",
					},
					"data": map[string]interface{}{
						"bar": secretValue,
					},
				},
			},
			keyName:  "secret_data_foo_bar",
			notFound: false,
		},
		{
			name:     "int secret skipped",
			path:     "kv/foo",
			noPrefix: config.Bool(false),
			data: &dependency.Secret{
				Data: map[string]interface{}{
					"bar": 1,
				},
			},
			notFound: true,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {
			cfg := Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path:     config.String(tc.path),
						NoPrefix: tc.noPrefix,
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

			var value string
			value, ok := env[tc.keyName]
			if !ok && !tc.notFound {
				t.Fatalf("expected (%s) key, but was not found", tc.keyName)
			}
			if ok && tc.notFound {
				t.Fatalf("expected to not find key, but (%s) was found", tc.keyName)
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
		noPrefix *bool
		data     []*dependency.KeyPair
		keyName  string
	}{
		{
			name:     "false noprefix appends path",
			path:     "app/my_service",
			noPrefix: config.Bool(false),
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
			noPrefix: config.Bool(true),
			data: []*dependency.KeyPair{
				&dependency.KeyPair{
					Key:   "mykey",
					Value: "myValue",
				},
			},
			keyName: "mykey",
		},
		{
			name:     "null noprefix excludes path",
			path:     "app/my_service",
			noPrefix: nil,
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
						NoPrefix: tc.noPrefix,
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
		{
			name:     "pristine env with no custom vars yields empty env",
			env:      map[string]string{"PATH": "/bin"},
			pristine: true,
			output:   map[string]string{},
		},
		{
			name:     "pristine env with custom vars only keeps custom vars",
			env:      map[string]string{"PATH": "/bin"},
			pristine: true,
			custom:   []string{"GOPATH=/usr/go"},
			output:   map[string]string{"GOPATH": "/usr/go"},
		},
		{
			name:   "custom vars overwrite input vars",
			env:    map[string]string{"PATH": "/bin"},
			custom: []string{"PATH=/usr/bin"},
			output: map[string]string{"PATH": "/usr/bin"},
		},
		{
			name:      "whitelist filters input by key",
			env:       map[string]string{"GOPATH": "/usr/go", "GO111MODULES": "true", "PATH": "/bin"},
			whitelist: []string{"GO*"},
			output:    map[string]string{"GOPATH": "/usr/go", "GO111MODULES": "true"},
		},
		{
			name:      "blacklist takes precedence over whitelist",
			env:       map[string]string{"GOPATH": "/usr/go", "PATH": "/bin", "EDITOR": "vi"},
			whitelist: []string{"GO*", "EDITOR"},
			blacklist: []string{"GO*"},
			output:    map[string]string{"EDITOR": "vi"},
		},
		{
			name:      "custom takes precedence over blacklist",
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
