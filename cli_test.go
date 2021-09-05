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
	"github.com/hashicorp/go-gatedio"
)

func TestCLI_ParseFlags(t *testing.T) {
	t.Parallel()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	cases := []struct {
		name string
		f    []string
		e    *Config
		err  bool
	}{
		// Deprecations
		// TODO: remove this in 0.8.0
		{
			"auth",
			[]string{"-auth", "abcd:efgh"},
			&Config{
				Consul: &config.ConsulConfig{
					Auth: &config.AuthConfig{
						Username: config.String("abcd"),
						Password: config.String("efgh"),
					},
				},
			},
			false,
		},
		{
			"consul",
			[]string{"-consul", "127.0.0.1:8500"},
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("127.0.0.1:8500"),
				},
			},
			false,
		},
		{
			"retry",
			[]string{"-retry", "10s"},
			&Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Backoff:    config.TimeDuration(10 * time.Second),
						MaxBackoff: config.TimeDuration(10 * time.Second),
					},
				},
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Backoff:    config.TimeDuration(10 * time.Second),
						MaxBackoff: config.TimeDuration(10 * time.Second),
					},
				},
			},
			false,
		},
		{
			"splay",
			[]string{"-splay", "10s"},
			&Config{
				Exec: &config.ExecConfig{
					Splay: config.TimeDuration(10 * time.Second),
				},
			},
			false,
		},
		{
			"ssl",
			[]string{"-ssl"},
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Enabled: config.Bool(true),
					},
				},
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"ssl_verify",
			[]string{"-ssl-verify"},
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Verify: config.Bool(true),
					},
				},
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Verify: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"ssl_ca-cert",
			[]string{"-ssl-ca-cert", "foo"},
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						CaCert: config.String("foo"),
					},
				},
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						CaCert: config.String("foo"),
					},
				},
			},
			false,
		},
		{
			"ssl_cert",
			[]string{"-ssl-cert", "foo"},
			&Config{
				Consul: &config.ConsulConfig{
					SSL: &config.SSLConfig{
						Cert: config.String("foo"),
					},
				},
				Vault: &config.VaultConfig{
					SSL: &config.SSLConfig{
						Cert: config.String("foo"),
					},
				},
			},
			false,
		},
		{
			"timeout",
			[]string{"-timeout", "10s"},
			&Config{
				Exec: &config.ExecConfig{
					Timeout: config.TimeDuration(10 * time.Second),
				},
			},
			false,
		},
		{
			"token",
			[]string{"-token", "abcd1234"},
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
			"config",
			[]string{"-config", f.Name()},
			&Config{},
			false,
		},
		{
			"config_multi",
			[]string{
				"-config", f.Name(),
				"-config", f.Name(),
			},
			&Config{},
			false,
		},
		{
			"consul_addr",
			[]string{"-consul-addr", "1.2.3.4"},
			&Config{
				Consul: &config.ConsulConfig{
					Address: config.String("1.2.3.4"),
				},
			},
			false,
		},
		{
			"consul_auth_empty",
			[]string{"-consul-auth", ""},
			nil,
			true,
		},
		{
			"consul_auth_username",
			[]string{"-consul-auth", "username"},
			&Config{
				Consul: &config.ConsulConfig{
					Auth: &config.AuthConfig{
						Username: config.String("username"),
					},
				},
			},
			false,
		},
		{
			"consul_auth_username_password",
			[]string{"-consul-auth", "username:password"},
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
			"consul-retry",
			[]string{"-consul-retry"},
			&Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul-retry-attempts",
			[]string{"-consul-retry-attempts", "20"},
			&Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Attempts: config.Int(20),
					},
				},
			},
			false,
		},
		{
			"consul-retry-backoff",
			[]string{"-consul-retry-backoff", "30s"},
			&Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						Backoff: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul-retry-max-backoff",
			[]string{"-consul-retry-max-backoff", "60s"},
			&Config{
				Consul: &config.ConsulConfig{
					Retry: &config.RetryConfig{
						MaxBackoff: config.TimeDuration(60 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul-ssl",
			[]string{"-consul-ssl"},
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
			"consul-ssl-ca-cert",
			[]string{"-consul-ssl-ca-cert", "ca_cert"},
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
			"consul-ssl-ca-path",
			[]string{"-consul-ssl-ca-path", "ca_path"},
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
			"consul-ssl-cert",
			[]string{"-consul-ssl-cert", "cert"},
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
			"consul-ssl-key",
			[]string{"-consul-ssl-key", "key"},
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
			"consul-ssl-server-name",
			[]string{"-consul-ssl-server-name", "server_name"},
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
			"consul-ssl-verify",
			[]string{"-consul-ssl-verify"},
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
			"consul-token",
			[]string{"-consul-token", "token"},
			&Config{
				Consul: &config.ConsulConfig{
					Token: config.String("token"),
				},
			},
			false,
		},
		{
			"consul-transport-dial-keep-alive",
			[]string{"-consul-transport-dial-keep-alive", "30s"},
			&Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						DialKeepAlive: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul-transport-dial-timeout",
			[]string{"-consul-transport-dial-timeout", "30s"},
			&Config{
				Consul: &config.ConsulConfig{
					Transport: &config.TransportConfig{
						DialTimeout: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul-transport-disable-keep-alives",
			[]string{"-consul-transport-disable-keep-alives"},
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
			"consul-transport-max-idle-conns-per-host",
			[]string{"-consul-transport-max-idle-conns-per-host", "100"},
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
			"consul-transport-tls-handshake-timeout",
			[]string{"-consul-transport-tls-handshake-timeout", "30s"},
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
			[]string{"-exec", "command"},
			&Config{
				Exec: &config.ExecConfig{
					Enabled: config.Bool(true),
					Command: config.String("command"),
				},
			},
			false,
		},
		{
			"exec-kill-signal",
			[]string{"-exec-kill-signal", "SIGUSR1"},
			&Config{
				Exec: &config.ExecConfig{
					KillSignal: config.Signal(syscall.SIGUSR1),
				},
			},
			false,
		},
		{
			"exec-kill-timeout",
			[]string{"-exec-kill-timeout", "10s"},
			&Config{
				Exec: &config.ExecConfig{
					KillTimeout: config.TimeDuration(10 * time.Second),
				},
			},
			false,
		},
		{
			"exec-splay",
			[]string{"-exec-splay", "10s"},
			&Config{
				Exec: &config.ExecConfig{
					Splay: config.TimeDuration(10 * time.Second),
				},
			},
			false,
		},
		{
			"kill-signal",
			[]string{"-kill-signal", "SIGUSR1"},
			&Config{
				KillSignal: config.Signal(syscall.SIGUSR1),
			},
			false,
		},
		{
			"log-level",
			[]string{"-log-level", "DEBUG"},
			&Config{
				LogLevel: config.String("DEBUG"),
			},
			false,
		},
		{
			"max-stale",
			[]string{"-max-stale", "10s"},
			&Config{
				MaxStale: config.TimeDuration(10 * time.Second),
			},
			false,
		},
		{
			"pid-file",
			[]string{"-pid-file", "/var/pid/file"},
			&Config{
				PidFile: config.String("/var/pid/file"),
			},
			false,
		},
		{
			"prefix",
			[]string{"-prefix", "foo/bar"},
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("foo/bar"),
					},
				},
			},
			false,
		},
		{
			"prefix_multi",
			[]string{
				"-prefix", "foo/bar",
				"-prefix", "zip/zap",
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
			false,
		},
		{
			"no-prefix",
			[]string{"-prefix", "foo/bar", "-no-prefix"},
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path:     config.String("foo/bar"),
						NoPrefix: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"no-prefix",
			[]string{"-secret", "foo/bar", "-no-prefix"},
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path:     config.String("foo/bar"),
						NoPrefix: config.Bool(true),
					},
				},
			},
			false,
		},
		{
			"no-prefix-false",
			[]string{"-prefix", "foo/bar", "-no-prefix=false"},
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path:     config.String("foo/bar"),
						NoPrefix: config.Bool(false),
					},
				},
			},
			false,
		},
		{
			"no-prefix-nil-default",
			[]string{"-prefix", "foo/bar"},
			&Config{
				Prefixes: &PrefixConfigs{
					&PrefixConfig{
						Path:     config.String("foo/bar"),
						NoPrefix: nil,
					},
				},
			},
			false,
		},
		{
			"pristine",
			[]string{"-pristine"},
			&Config{
				Pristine: config.Bool(true),
			},
			false,
		},
		{
			"reload-signal",
			[]string{"-reload-signal", "SIGUSR1"},
			&Config{
				ReloadSignal: config.Signal(syscall.SIGUSR1),
			},
			false,
		},
		{
			"sanitize",
			[]string{"-sanitize"},
			&Config{
				Sanitize: config.Bool(true),
			},
			false,
		},
		{
			"secret",
			[]string{"-secret", "foo/bar"},
			&Config{
				Secrets: &PrefixConfigs{
					&PrefixConfig{
						Path: config.String("foo/bar"),
					},
				},
			},
			false,
		},
		{
			"secret_multi",
			[]string{
				"-secret", "foo/bar",
				"-secret", "zip/zap",
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
			false,
		},
		{
			"service-query",
			[]string{
				"-service-query", "service",
			},
			&Config{
				Services: &ServiceConfigs{
					&ServiceConfig{
						Query: config.String("service"),
					},
				},
			},
			false,
		},
		{
			"query_multi",
			[]string{
				"-service-query", "service",
				"-service-query", "tag.service",
				"-service-query", "tag.service@datacenter",
			},
			&Config{
				Services: &ServiceConfigs{
					&ServiceConfig{
						Query: config.String("service"),
					},
					&ServiceConfig{
						Query: config.String("tag.service"),
					},
					&ServiceConfig{
						Query: config.String("tag.service@datacenter"),
					},
				},
			},
			false,
		},
		{
			"service_format",
			[]string{
				"-service-query", "service",
				"-service-format-id", "id",
				"-service-format-name", "name",
				"-service-format-address", "host",
				"-service-format-tag", "tag",
				"-service-format-port", "port",
			},
			&Config{
				Services: &ServiceConfigs{
					&ServiceConfig{
						Query:         config.String("service"),
						FormatId:      config.String("id"),
						FormatName:    config.String("name"),
						FormatAddress: config.String("host"),
						FormatTag:     config.String("tag"),
						FormatPort:    config.String("port"),
					},
				},
			},
			false,
		},
		{
			"service_format_multy",
			[]string{
				"-service-query", "foo",
				"-service-format-id", "foo/id",
				"-service-format-name", "foo/name",
				"-service-format-address", "foo/host",
				"-service-format-tag", "foo/tag",
				"-service-format-port", "foo/port",
				"-service-query", "bar",
				"-service-format-id", "bar/id",
				"-service-format-name", "bar/name",
				"-service-format-address", "bar/host",
				"-service-format-tag", "bar/tag",
				"-service-format-port", "bar/port",
			},
			&Config{
				Services: &ServiceConfigs{
					&ServiceConfig{
						Query:         config.String("foo"),
						FormatId:      config.String("foo/id"),
						FormatName:    config.String("foo/name"),
						FormatAddress: config.String("foo/host"),
						FormatTag:     config.String("foo/tag"),
						FormatPort:    config.String("foo/port"),
					},
					&ServiceConfig{
						Query:         config.String("bar"),
						FormatId:      config.String("bar/id"),
						FormatName:    config.String("bar/name"),
						FormatAddress: config.String("bar/host"),
						FormatTag:     config.String("bar/tag"),
						FormatPort:    config.String("bar/port"),
					},
				},
			},
			false,
		},
		{
			"syslog",
			[]string{"-syslog"},
			&Config{
				Syslog: &config.SyslogConfig{
					Enabled: config.Bool(true),
				},
			},
			false,
		},
		{
			"syslog-facility",
			[]string{"-syslog-facility", "LOCAL0"},
			&Config{
				Syslog: &config.SyslogConfig{
					Facility: config.String("LOCAL0"),
				},
			},
			false,
		},
		{
			"upcase",
			[]string{"-upcase"},
			&Config{
				Upcase: config.Bool(true),
			},
			false,
		},
		{
			"vault-addr",
			[]string{"-vault-addr", "vault_addr"},
			&Config{
				Vault: &config.VaultConfig{
					Address: config.String("vault_addr"),
				},
			},
			false,
		},
		{
			"vault-namespace",
			[]string{"-vault-namespace", "vault_namespace"},
			&Config{
				Vault: &config.VaultConfig{
					Namespace: config.String("vault_namespace"),
				},
			},
			false,
		},
		{
			"vault-retry",
			[]string{"-vault-retry"},
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
			"vault-retry-attempts",
			[]string{"-vault-retry-attempts", "20"},
			&Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Attempts: config.Int(20),
					},
				},
			},
			false,
		},
		{
			"vault-retry-backoff",
			[]string{"-vault-retry-backoff", "30s"},
			&Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						Backoff: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-retry-max-backoff",
			[]string{"-vault-retry-max-backoff", "60s"},
			&Config{
				Vault: &config.VaultConfig{
					Retry: &config.RetryConfig{
						MaxBackoff: config.TimeDuration(60 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-renew-token",
			[]string{"-vault-renew-token"},
			&Config{
				Vault: &config.VaultConfig{
					RenewToken: config.Bool(true),
				},
			},
			false,
		},
		{
			"vault-ssl",
			[]string{"-vault-ssl"},
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
			"vault-ssl-ca-cert",
			[]string{"-vault-ssl-ca-cert", "ca_cert"},
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
			"vault-ssl-ca-path",
			[]string{"-vault-ssl-ca-path", "ca_path"},
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
			"vault-ssl-cert",
			[]string{"-vault-ssl-cert", "cert"},
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
			"vault-ssl-key",
			[]string{"-vault-ssl-key", "key"},
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
			"vault-ssl-server-name",
			[]string{"-vault-ssl-server-name", "server_name"},
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
			"vault-ssl-verify",
			[]string{"-vault-ssl-verify"},
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
			"vault-token",
			[]string{"-vault-token", "token"},
			&Config{
				Vault: &config.VaultConfig{
					Token: config.String("token"),
				},
			},
			false,
		},
		{
			"vault-transport-dial-keep-alive",
			[]string{"-vault-transport-dial-keep-alive", "30s"},
			&Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						DialKeepAlive: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-transport-dial-timeout",
			[]string{"-vault-transport-dial-timeout", "30s"},
			&Config{
				Vault: &config.VaultConfig{
					Transport: &config.TransportConfig{
						DialTimeout: config.TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"vault-transport-disable-keep-alives",
			[]string{"-vault-transport-disable-keep-alives"},
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
			"vault-transport-max-idle-conns-per-host",
			[]string{"-vault-transport-max-idle-conns-per-host", "100"},
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
			"vault-transport-tls-handshake-timeout",
			[]string{"-vault-transport-tls-handshake-timeout", "30s"},
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
			"vault-unwrap-token",
			[]string{"-vault-unwrap-token"},
			&Config{
				Vault: &config.VaultConfig{
					UnwrapToken: config.Bool(true),
				},
			},
			false,
		},
		{
			"wait_min",
			[]string{"-wait", "10s"},
			&Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(10 * time.Second),
					Max: config.TimeDuration(40 * time.Second),
				},
			},
			false,
		},
		{
			"wait_min_max",
			[]string{"-wait", "10s:30s"},
			&Config{
				Wait: &config.WaitConfig{
					Min: config.TimeDuration(10 * time.Second),
					Max: config.TimeDuration(30 * time.Second),
				},
			},
			false,
		},

		// Edge cases
		{
			"command",
			[]string{"my", "command", "to", "run"},
			&Config{
				Exec: &config.ExecConfig{
					Enabled: config.Bool(true),
					Command: config.String("my command to run"),
				},
			},
			false,
		},
		{
			"command_and_exec",
			[]string{
				"-exec", "command 1",
				"command", "2",
			},
			&Config{
				Exec: &config.ExecConfig{
					Enabled: config.Bool(true),
					Command: config.String("command 1"),
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			out := gatedio.NewByteBuffer()
			cli := NewCLI(out, out)

			a, _, _, _, err := cli.ParseFlags(tc.f)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if tc.e != nil {
				tc.e = DefaultConfig().Merge(tc.e)
			}

			if !reflect.DeepEqual(tc.e, a) {
				t.Errorf("\nexp: %#v\nact: %#v\nout: %q", tc.e, a, out.String())
			}
		})
	}
}
