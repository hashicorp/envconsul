package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/consul-template/config"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		i    string
		e    *Config
		err  bool
	}{
		// Deprecations
		// TODO: remove this in 0.8.0
		{
			"auth",
			`auth {
				enabled  = true
				username = "foo"
				password = "bar"
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Auth: &config.AuthConfig{
						Enabled:  config.Bool(true),
						Username: config.String("foo"),
						Password: config.String("bar"),
					},
				},
			},
			false,
		},
		{
			"consul_top_level",
			`consul = "127.0.0.1:8500"`,
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("127.0.0.1:8500"),
				},
			},
			false,
		},
		{
			"path_top_level",
			`path = "/foo/bar"`,
			&Config{},
			false,
		},
		{
			"retry_top_level",
			`retry = "5s"`,
			&Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Backoff:    config.TimeDuration(5 * time.Second),
						MaxBackoff: config.TimeDuration(5 * time.Second),
					},
				},
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Backoff:    config.TimeDuration(5 * time.Second),
						MaxBackoff: config.TimeDuration(5 * time.Second),
					},
				},
			},
			false,
		},
		{
			"retry_top_level_int",
			`retry = 5`,
			&Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Backoff:    config.TimeDuration(5 * time.Nanosecond),
						MaxBackoff: config.TimeDuration(5 * time.Nanosecond),
					},
				},
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Backoff:    config.TimeDuration(5 * time.Nanosecond),
						MaxBackoff: config.TimeDuration(5 * time.Nanosecond),
					},
				},
			},
			false,
		},
		{
			"ssl",
			`ssl {
				enabled = true
				verify  = false
				cert    = "cert"
				key     = "key"
				ca_cert = "ca_cert"
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Enabled: config.Bool(true),
						Verify:  config.Bool(false),
						CaCert:  config.String("ca_cert"),
						Cert:    config.String("cert"),
						Key:     config.String("key"),
					},
				},
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Enabled: config.Bool(true),
						Verify:  config.Bool(false),
						CaCert:  config.String("ca_cert"),
						Cert:    config.String("cert"),
						Key:     config.String("key"),
					},
				},
			},
			false,
		},
		{
			"splay_top_level",
			`splay = "5s"`,
			&Config{
				Exec: &config.ExecConfig{
					Splay: config.TimeDuration(5 * time.Second),
				},
			},
			false,
		},
		{
			"timeout_top_level",
			`timeout = "10s"`,
			&Config{
				Exec: &config.ExecConfig{
					KillTimeout: config.TimeDuration(10 * time.Second),
				},
			},
			false,
		},
		{
			"token_top_level",
			`token = "abcd1234"`,
			&Config{
				Consul: &config.ConsulConfig{
					Token: config.String("abcd1234"),
				},
			},
			false,
		},
		// End Depreations
		// TODO remove in 0.8.0

		{
			"consul_address",
			`consul {
				address = "1.2.3.4"
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("1.2.3.4"),
				},
			},
			false,
		},
		{
			"consul_auth",
			`consul {
				auth {
					username = "username"
					password = "password"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Auth: &config.AuthConfig{
						Username: config.String("username"),
						Password: config.String("password"),
					},
				},
			},
			false,
		},
		{
			"consul_retry",
			`consul {
				retry {
					backoff  = "2s"
					attempts = 10
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Attempts: config.Int(10),
						Backoff:  config.TimeDuration(2 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul_ssl",
			`consul {
				ssl {}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{},
				},
			},
			false,
		},
		{
			"consul_ssl_enabled",
			`consul {
				ssl {
					enabled = true
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_verify",
			`consul {
				ssl {
					verify = true
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Verify: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_cert",
			`consul {
				ssl {
					cert = "cert"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Cert: config.String("cert"),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_key",
			`consul {
				ssl {
					key = "key"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Key: config.String("key"),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_ca_cert",
			`consul {
				ssl {
					ca_cert = "ca_cert"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						CaCert: config.String("ca_cert"),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_ca_path",
			`consul {
				ssl {
					ca_path = "ca_path"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						CaPath: config.String("ca_path"),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_server_name",
			`consul {
				ssl {
					server_name = "server_name"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						ServerName: config.String("server_name"),
					},
				},
			},
			false,
		},
		{
			"consul_token",
			`consul {
				token = "token"
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Token: config.String("token"),
				},
			},
			false,
		},
		{
			"consul_transport_dial_keep_alive",
			`consul {
				transport {
					dial_keep_alive = "10s"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						DialKeepAlive: config.TimeDuration(10 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul_transport_dial_timeout",
			`consul {
				transport {
					dial_timeout = "10s"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						DialTimeout: config.TimeDuration(10 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul_transport_disable_keep_alives",
			`consul {
				transport {
					disable_keep_alives = true
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						DisableKeepAlives: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul_transport_max_idle_conns_per_host",
			`consul {
				transport {
					max_idle_conns_per_host = 100
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						MaxIdleConnsPerHost: config.Int(100),
					},
				},
			},
			false,
		},
		{
			"consul_transport_tls_handshake_timeout",
			`consul {
				transport {
					tls_handshake_timeout = "30s"
				}
			}`,
			&Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						TLSHandshakeTimeout: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"exec",
			`exec {}`,
			&Config{
				Exec: &config.ExecConfig{},
			},
			false,
		},
		{
			"exec_command",
			`exec {
				command = "command"
			}`,
			&Config{
				Exec: &config.ExecConfig{
					Command: config.String("command"),
				},
			},
			false,
		},
		{
			"exec_enabled",
			`exec {
				enabled = true
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					Enabled: config.Bool(true),
				},
			},
			false,
		},
		{
			"exec_env",
			`exec {
				env {}
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					Env: &config.EnvConfig{},
				},
			},
			false,
		},
		{
			"exec_env_blacklist",
			`exec {
				env {
					blacklist = ["a", "b"]
				}
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					Env: &config.EnvConfig{
						Blacklist: []string{"a", "b"},
					},
				},
			},
			false,
		},
		{
			"exec_env_custom",
			`exec {
				env {
					custom = ["a=b", "c=d"]
				}
			}`,
			&Config{
				Exec: &config.ExecConfig{
					Env: &config.EnvConfig{
						Custom: []string{"a=b", "c=d"},
					},
				},
			},
			false,
		},
		{
			"exec_env_pristine",
			`exec {
				env {
					pristine = true
				}
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					Env: &config.EnvConfig{
						Pristine: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"exec_env_whitelist",
			`exec {
				env {
					whitelist = ["a", "b"]
				}
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					Env: &config.EnvConfig{
						Whitelist: []string{"a", "b"},
					},
				},
			},
			false,
		},
		{
			"exec_kill_signal",
			`exec {
				kill_signal = "SIGUSR1"
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					KillSignal: config.Signal(syscall.SIGUSR1),
				},
			},
			false,
		},
		{
			"exec_kill_timeout",
			`exec {
				kill_timeout = "30s"
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					KillTimeout: config.TimeDuration(30 * time.Second),
				},
			},
			false,
		},
		{
			"exec_reload_signal",
			`exec {
				reload_signal = "SIGUSR1"
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					ReloadSignal: config.Signal(syscall.SIGUSR1),
				},
			},
			false,
		},
		{
			"exec_splay",
			`exec {
				splay = "30s"
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					Splay: config.TimeDuration(30 * time.Second),
				},
			},
			false,
		},
		{
			"exec_timeout",
			`exec {
				timeout = "30s"
			 }`,
			&Config{
				Exec: &config.ExecConfig{
					Timeout: config.TimeDuration(30 * time.Second),
				},
			},
			false,
		},
		{
			"kill_signal",
			`kill_signal = "SIGUSR1"`,
			&Config{
				KillSignal: config.Signal(syscall.SIGUSR1),
			},
			false,
		},
		{
			"log_level",
			`log_level = "WARN"`,
			&Config{
				LogLevel: config.String("WARN"),
			},
			false,
		},
		{
			"max_stale",
			`max_stale = "10s"`,
			&Config{
				MaxStale: config.TimeDuration(10 * time.Second),
			},
			false,
		},
		{
			"pid_file",
			`pid_file = "/var/pid"`,
			&Config{
				PidFile: config.String("/var/pid"),
			},
			false,
		},
		{
			"prefix",
			`prefix {}`,
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{},
				},
			},
			false,
		},
		{
			"prefix_multi",
			`prefix {}
			prefix{}`,
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{},
					&PrefixConfig{},
				},
			},
			false,
		},
		{
			"prefix_format",
			`prefix {
				format = "foo_%s"
			}`,
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Format: config.String("foo_%s"),
					},
				},
			},
			false,
		},
		{
			"prefix_no_prefix",
			`prefix {
				no_prefix = true
			}`,
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						NoPrefix: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"prefix_path",
			`prefix {
				path = "foo/bar/baz"
			}`,
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("foo/bar/baz"),
					},
				},
			},
			false,
		},
		{
			"prefix_path_template",
			`prefix {
				path = "foo/{{ env \"BAR\" }}"
			}`,
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String(`foo/{{ env "BAR" }}`),
					},
				},
			},
			false,
		},
		{
			"pristine",
			`pristine = true`,
			&Config{
				Pristine: config.Bool(true),
			},
			false,
		},
		{
			"reload_signal",
			`reload_signal = "SIGUSR1"`,
			&Config{
				ReloadSignal: config.Signal(syscall.SIGUSR1),
			},
			false,
		},
		{
			"sanitize",
			`sanitize = true`,
			&Config{
				Sanitize: config.Bool(true),
			},
			false,
		},
		{
			"secret",
			`secret {}`,
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{},
				},
			},
			false,
		},
		{
			"secret_multi",
			`secret {}
			secret{}`,
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{},
					&PrefixConfig{},
				},
			},
			false,
		},
		{
			"secret_format",
			`secret {
				format = "foo_%s"
			}`,
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Format: config.String("foo_%s"),
					},
				},
			},
			false,
		},
		{
			"secret_no_prefix",
			`secret {
				no_prefix = true
			}`,
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						NoPrefix: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"secret_path",
			`secret {
				path = "foo/bar/baz"
			}`,
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("foo/bar/baz"),
					},
				},
			},
			false,
		},
		{
			"syslog",
			`syslog {}`,
			&Config{
				Syslog: &config.SyslogConfig{},
			},
			false,
		},
		{
			"syslog_enabled",
			`syslog {
				enabled = true
			}`,
			&Config{
				Syslog: &config.SyslogConfig{
					Enabled: config.Bool(true),
				},
			},
			false,
		},
		{
			"syslog_facility",
			`syslog {
				facility = "facility"
			}`,
			&Config{
				Syslog: &config.SyslogConfig{
					Facility: config.String("facility"),
				},
			},
			false,
		},
		{
			"upcase",
			`upcase = true`,
			&Config{
				Upcase: config.Bool(true),
			},
			false,
		},
		{
			"vault",
			`vault {}`,
			&Config{
				Vault: &config.VaultConfig{},
			},
			false,
		},
		{
			"vault_enabled",
			`vault {
				enabled = true
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Enabled: config.Bool(true),
				},
			},
			false,
		},
		{
			"vault_address",
			`vault {
				address = "address"
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Address: config.String("address"),
				},
			},
			false,
		},
		{
			"vault_grace",
			`vault {
				grace = "5m"
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Grace: config.TimeDuration(5 * time.Minute),
				},
			},
			false,
		},
		{
			"vault_token",
			`vault {
				token = "token"
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Token: config.String("token"),
				},
			},
			false,
		},
		{
			"vault_transport_dial_keep_alive",
			`vault {
				transport {
					dial_keep_alive = "10s"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						DialKeepAlive: config.TimeDuration(10 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault_transport_dial_timeout",
			`vault {
				transport {
					dial_timeout = "10s"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						DialTimeout: config.TimeDuration(10 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault_transport_disable_keep_alives",
			`vault {
				transport {
					disable_keep_alives = true
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						DisableKeepAlives: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"vault_transport_max_idle_conns_per_host",
			`vault {
				transport {
					max_idle_conns_per_host = 100
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						MaxIdleConnsPerHost: config.Int(100),
					},
				},
			},
			false,
		},
		{
			"vault_transport_tls_handshake_timeout",
			`vault {
				transport {
					tls_handshake_timeout = "30s"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						TLSHandshakeTimeout: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault_unwrap_token",
			`vault {
				unwrap_token = true
			}`,
			&Config{
				Vault: &config.VaultConfig{
					UnwrapToken: config.Bool(true),
				},
			},
			false,
		},
		{
			"vault_renew_token",
			`vault {
				renew_token = true
			}`,
			&Config{
				Vault: &config.VaultConfig{
					RenewToken: config.Bool(true),
				},
			},
			false,
		},
		{
			"vault_retry_backoff",
			`vault {
				retry {
					backoff = "5s"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Backoff: config.TimeDuration(5 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault_retry_enabled",
			`vault {
				retry {
					enabled = true
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"vault_retry_disabled",
			`vault {
				retry {
					enabled = false
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Enabled: config.Bool(false),
					},
				},
			},
			false,
		},
		{
			"vault_retry_max_attempts",
			`vault {
				retry {
					attempts = 10
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Attempts: config.Int(10),
					},
				},
			},
			false,
		},
		{
			"vault_ssl",
			`vault {
				ssl {}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{},
				},
			},
			false,
		},
		{
			"vault_ssl_enabled",
			`vault {
				ssl {
					enabled = true
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"vault_ssl_verify",
			`vault {
				ssl {
					verify = true
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Verify: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"vault_ssl_cert",
			`vault {
				ssl {
					cert = "cert"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Cert: config.String("cert"),
					},
				},
			},
			false,
		},
		{
			"vault_ssl_key",
			`vault {
				ssl {
					key = "key"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Key: config.String("key"),
					},
				},
			},
			false,
		},
		{
			"vault_ssl_ca_cert",
			`vault {
				ssl {
					ca_cert = "ca_cert"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						CaCert: config.String("ca_cert"),
					},
				},
			},
			false,
		},
		{
			"vault_ssl_ca_path",
			`vault {
				ssl {
					ca_path = "ca_path"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						CaPath: config.String("ca_path"),
					},
				},
			},
			false,
		},
		{
			"vault_ssl_server_name",
			`vault {
				ssl {
					server_name = "server_name"
				}
			}`,
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						ServerName: config.String("server_name"),
					},
				},
			},
			false,
		},
		{
			"wait",
			`wait {
				min = "10s"
				max = "20s"
			}`,
			&Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(10 * time.Second),
					Max: config.TimeDuration(20 * time.Second),
				},
			},
			false,
		},
		{
			// Previous wait declarations used this syntax, but now use the stanza
			// syntax. Keep this around for backwards-compat.
			"wait_as_string",
			`wait = "10s:20s"`,
			&Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(10 * time.Second),
					Max: config.TimeDuration(20 * time.Second),
				},
			},
			false,
		},

		// General validation
		{
			"invalid_key",
			`not_a_valid_key = "hello"`,
			nil,
			true,
		},
		{
			"invalid_stanza",
			`not_a_valid_stanza {
				a = "b"
			}`,
			nil,
			true,
		},
		{
			"mapstructure_error",
			`consul = true`,
			nil,
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			c, err := Parse(tc.i)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.e, c) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.e, c)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *Config
		b    *Config
		r    *Config
	}{
		{
			"nil_a",
			nil,
			&Config{},
			&Config{},
		},
		{
			"nil_b",
			&Config{},
			nil,
			&Config{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&Config{},
			&Config{},
			&Config{},
		},
		{
			"consul",
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("consul"),
				},
			},
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("consul-diff"),
				},
			},
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("consul-diff"),
				},
			},
		},
		{
			"exec",
			&Config{
				Exec: &config.ExecConfig{
					Command: config.String("command"),
				},
			},
			&Config{
				Exec: &config.ExecConfig{
					Command: config.String("command-diff"),
				},
			},
			&Config{
				Exec: &config.ExecConfig{
					Command: config.String("command-diff"),
				},
			},
		},
		{
			"kill_signal",
			&Config{
				KillSignal: config.Signal(syscall.SIGUSR1),
			},
			&Config{
				KillSignal: config.Signal(syscall.SIGUSR2),
			},
			&Config{
				KillSignal: config.Signal(syscall.SIGUSR2),
			},
		},
		{
			"log_level",
			&Config{
				LogLevel: config.String("log_level"),
			},
			&Config{
				LogLevel: config.String("log_level-diff"),
			},
			&Config{
				LogLevel: config.String("log_level-diff"),
			},
		},
		{
			"max_stale",
			&Config{
				MaxStale: config.TimeDuration(10 * time.Second),
			},
			&Config{
				MaxStale: config.TimeDuration(20 * time.Second),
			},
			&Config{
				MaxStale: config.TimeDuration(20 * time.Second),
			},
		},
		{
			"pid_file",
			&Config{
				PidFile: config.String("pid_file"),
			},
			&Config{
				PidFile: config.String("pid_file-diff"),
			},
			&Config{
				PidFile: config.String("pid_file-diff"),
			},
		},
		{
			"prefix",
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("foo/bar"),
					},
				},
			},
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("zip/zap"),
					},
				},
			},
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("foo/bar"),
					},
					&PrefixConfig{
						Path: config.String("zip/zap"),
					},
				},
			},
		},
		{
			"pristine",
			&Config{
				Pristine: config.Bool(true),
			},
			&Config{
				Pristine: config.Bool(false),
			},
			&Config{
				Pristine: config.Bool(false),
			},
		},
		{
			"reload_signal",
			&Config{
				ReloadSignal: config.Signal(syscall.SIGUSR1),
			},
			&Config{
				ReloadSignal: config.Signal(syscall.SIGUSR2),
			},
			&Config{
				ReloadSignal: config.Signal(syscall.SIGUSR2),
			},
		},
		{
			"sanitize",
			&Config{
				Sanitize: config.Bool(true),
			},
			&Config{
				Sanitize: config.Bool(false),
			},
			&Config{
				Sanitize: config.Bool(false),
			},
		},
		{
			"secret",
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("foo/bar"),
					},
				},
			},
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("zip/zap"),
					},
				},
			},
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("foo/bar"),
					},
					&PrefixConfig{
						Path: config.String("zip/zap"),
					},
				},
			},
		},
		{
			"syslog",
			&Config{
				Syslog: &config.SyslogConfig{
					Enabled: config.Bool(true),
				},
			},
			&Config{
				Syslog: &config.SyslogConfig{
					Enabled: config.Bool(false),
				},
			},
			&Config{
				Syslog: &config.SyslogConfig{
					Enabled: config.Bool(false),
				},
			},
		},
		{
			"upcase",
			&Config{
				Upcase: config.Bool(true),
			},
			&Config{
				Upcase: config.Bool(false),
			},
			&Config{
				Upcase: config.Bool(false),
			},
		},
		{
			"vault",
			&Config{
				Vault: &config.VaultConfig{
					Enabled: config.Bool(true),
				},
			},
			&Config{
				Vault: &config.VaultConfig{
					Enabled: config.Bool(false),
				},
			},
			&Config{
				Vault: &config.VaultConfig{
					Enabled: config.Bool(false),
				},
			},
		},
		{
			"wait",
			&Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(10 * time.Second),
					Max: config.TimeDuration(20 * time.Second),
				},
			},
			&Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(20 * time.Second),
					Max: config.TimeDuration(50 * time.Second),
				},
			},
			&Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(20 * time.Second),
					Max: config.TimeDuration(50 * time.Second),
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			if !reflect.DeepEqual(tc.r, r) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, r)
			}
		})
	}
}

func TestFromPath(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	emptyDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(emptyDir)

	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(configDir)
	cf1, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	d := []byte(`
		consul {
			address = "1.2.3.4"
		}
	`)
	if err = ioutil.WriteFile(cf1.Name(), d, 0644); err != nil {
		t.Fatal(err)
	}
	cf2, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	d = []byte(`
		consul {
			token = "token"
		}
	`)
	if err := ioutil.WriteFile(cf2.Name(), d, 0644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		path string
		e    *Config
		err  bool
	}{
		{
			"missing_dir",
			"/not/a/real/dir",
			nil,
			true,
		},
		{
			"file",
			f.Name(),
			&Config{},
			false,
		},
		{
			"empty_dir",
			emptyDir,
			nil,
			false,
		},
		{
			"config_dir",
			configDir,
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("1.2.3.4"),
					Token:   config.String("token"),
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			c, err := FromPath(tc.path)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.e, c) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.e, c)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cases := []struct {
		env string
		val string
		e   *Config
		err bool
	}{
		{
			"CONSUL_HTTP_ADDR",
			"1.2.3.4",
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("1.2.3.4"),
				},
			},
			false,
		},
		{
			"CONSUL_TEMPLATE_LOG",
			"DEBUG",
			&Config{
				LogLevel: config.String("DEBUG"),
			},
			false,
		},
		{
			"CONSUL_TOKEN",
			"token",
			&Config{
				Consul: &config.ConsulConfig{
					Token: config.String("token"),
				},
			},
			false,
		},
		{
			"VAULT_ADDR",
			"http://1.2.3.4:8200",
			&Config{
				Vault: &config.VaultConfig{
					Address: config.String("http://1.2.3.4:8200"),
				},
			},
			false,
		},
		{
			"VAULT_TOKEN",
			"abcd1234",
			&Config{
				Vault: &config.VaultConfig{
					Token: config.String("abcd1234"),
				},
			},
			false,
		},
		{
			"VAULT_UNWRAP_TOKEN",
			"true",
			&Config{
				Vault: &config.VaultConfig{
					UnwrapToken: config.Bool(true),
				},
			},
			false,
		},
		{
			"VAULT_UNWRAP_TOKEN",
			"false",
			&Config{
				Vault: &config.VaultConfig{
					UnwrapToken: config.Bool(false),
				},
			},
			false,
		},
		{
			"VAULT_CA_PATH",
			"ca_path",
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						CaPath: config.String("ca_path"),
					},
				},
			},
			false,
		},
		{
			"VAULT_CA_CERT",
			"ca_cert",
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						CaCert: config.String("ca_cert"),
					},
				},
			},
			false,
		},
		{
			"VAULT_TLS_SERVER_NAME",
			"server_name",
			&Config{
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						ServerName: config.String("server_name"),
					},
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.env), func(t *testing.T) {
			if err := os.Setenv(tc.env, tc.val); err != nil {
				t.Fatal(err)
			}
			defer os.Unsetenv(tc.env)

			r := DefaultConfig()
			r.Merge(tc.e)

			c := DefaultConfig()
			if !reflect.DeepEqual(r, c) {
				t.Errorf("\nexp: %#v\nact: %#v", r, c)
			}
		})
	}
}
