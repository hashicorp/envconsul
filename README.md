# envconsul [![Build Status](http://img.shields.io/travis/hashicorp/envconsul.svg?style=flat-square)](http://travis-ci.org/hashicorp/envconsul)

Envconsul provides a convenient way to launch a subprocess with environment
variables populated from HashiCorp [Consul][consul] and [Vault][vault]. The tool
is inspired by [envdir][envdir] and [envchain][envchain], but works on many
major operating systems with no runtime requirements. It is also available
via a Docker container for scheduled environments.

Envconsul supports [12-factor applications][12-factor] which get their
configuration via the environment. Environment variables are dynamically
populated from Consul or Vault, but the application is unaware; applications
just read environment variables. This enables extreme flexibility and
portability for applications across systems.

**The documentation in this README corresponds to the master branch of envconsul. It may contain unreleased features or different APIs than the most recently released version. Please see the Git tag that corresponds to your version of envconsul for the proper documentation.**

## Installation

### Pre-Compiled

1. Download a pre-compiled, released version from the [envconsul
  releases][releases] page. You can download zip or tarball.

    ```shell
    $ curl -so envconsul.tgz https://releases.hashicorp.com/envconsul/0.7.3/envconsul_0.7.3_linux_amd64.tgz
    ```

1. Extract the binary using `unzip` or `tar`.

    ```shell
    $ tar -xvzf envconsul.tgz
    ```

1. Move the binary into your `$PATH`.

    ```shell
    $ mv envconsul /usr/local/bin/envconsul
    $ chmod +x /usr/local/bin/envconsul
    ```

### From Source (Go)

1. Install common build tools and [go][go].

1. Clone the repository from GitHub.

    ```shell
    $ git clone https://github.com/hashicorp/envconsul.git
    $ cd envconsul
    ```

1. Run the development make target.

    ```shell
    $ make dev
    ```

### From Source (Docker)

1. Install Docker for your platform.

1. Clone the repository from GitHub.

    ```shell
    $ git clone https://github.com/hashicorp/envconsul.git
    $ cd envconsul
    ```

1. Run the make target for your platform and architecture.

    ```shell
    $ make darwin/amd64 # or linux/amd64 or windows/amd64, etc
    ```

This process will build `envconsul` into `pkg/OS_ARCH`. You can move this into
your path or execute it directly.

## Quick Example

This short example assumes Consul is installed locally.

1. Start a Consul cluster in dev mode.

    ```shell
    $ consul agent -dev
    ```

1. Write some data.

    ```shell
    $ consul kv put my-app/address 1.2.3.4
    $ consul kv put my-app/port 80
    $ consul kv put my-app/max_conns 5
    ```

1. Execute envconsul with a subprocess (`env` in this example).

    ```shell
    $ envconsul -prefix my-app env
    ```

    Envconsul will connect to Consul, read the data from the key-value store,
    and populate environment variables corresponding to those values. Here is
    sample output.

    ```text
    address=1.2.3.4
    max_conns=5
    port=80
    ```

For more examples and use cases, please see the [examples folder][examples] in
this repository.

## Usage

For the full list of command-line options:

```shell
$ envconsul -h
```

### Command Line Interface (CLI)

The Envconsul CLI interface supports all options in the configuration file and
visa-versa. Here are some common examples of CLI usage. For the full list of
options, please run `envconsul -h`.

Render data from the prefix `my-app` into the environment.

```shell
$ envconsul -prefix my-app ruby my-app.rb
```

Render _only_ data from the two prefixes into the environment (the parent
processes environment will not be copied).

```shell
$ envconsul -pristine -prefix common -prefix my-app yarn start
```

Convert environment variables to upcase and remove any non-standard keys (like
dashes to underscores).

```shell
$ envconsul -upcase -sanitize -prefix my-app python my-app.my
```

Read secrets from Vault.

```shell
$ envconsul -secret secret/my-app ./my-app
```

### Configuration File

Configuration files are written in the [HashiCorp Configuration Language][hcl].
By proxy, this means the configuration is also JSON compatible.

```hcl
# This denotes the start of the configuration section for Consul. All values
# contained in this section pertain to Consul.
consul {
  # This block specifies the basic authentication information to pass with the
  # request. For more information on authentication, please see the Consul
  # documentation.
  auth {
    enabled  = true
    username = "test"
    password = "test"
  }

  # This is the address of the Consul agent. By default, this is
  # 127.0.0.1:8500, which is the default bind and port for a local Consul
  # agent. It is not recommended that you communicate directly with a Consul
  # server, and instead communicate with the local Consul agent. There are many
  # reasons for this, most importantly the Consul agent is able to multiplex
  # connections to the Consul server and reduce the number of open HTTP
  # connections. Additionally, it provides a "well-known" IP address for which
  # clients can connect.
  address = "127.0.0.1:8500"

  # This is the ACL token to use when connecting to Consul. If you did not
  # enable ACLs on your Consul cluster, you do not need to set this option.
  #
  # This option is also available via the environment variable CONSUL_TOKEN.
  token = "abcd1234"

  # This controls the retry behavior when an error is returned from Consul.
  # Envconsul is highly fault tolerant, meaning it does not exit in the face
  # of failure. Instead, it uses exponential back-off and retry functions
  # to wait for the cluster to become available, as is customary in distributed
  # systems.
  retry {
    # This enabled retries. Retries are enabled by default, so this is
    # redundant.
    enabled = true

    # This specifies the number of attempts to make before giving up. Each
    # attempt adds the exponential backoff sleep time. Setting this to
    # zero will implement an unlimited number of retries.
    attempts = 12

    # This is the base amount of time to sleep between retry attempts. Each
    # retry sleeps for an exponent of 2 longer than this base. For 5 retries,
    # the sleep times would be: 250ms, 500ms, 1s, 2s, then 4s.
    backoff = "250ms"

    # This is the maximum amount of time to sleep between retry attempts.
    # When max_backoff is set to zero, there is no upper limit to the
    # exponential sleep between retry attempts.
    # If max_backoff is set to 10s and backoff is set to 1s, sleep times
    # would be: 1s, 2s, 4s, 8s, 10s, 10s, ...
    max_backoff = "1m"
  }

  # This block configures the SSL options for connecting to the Consul server.
  ssl {
    # This enables SSL. Specifying any option for SSL will also enable it.
    enabled = true

    # This enables SSL peer verification. The default value is "true", which
    # will check the global CA chain to make sure the given certificates are
    # valid. If you are using a self-signed certificate that you have not added
    # to the CA chain, you may want to disable SSL verification. However, please
    # understand this is a potential security vulnerability.
    verify = false

    # This is the path to the certificate to use to authenticate. If just a
    # certificate is provided, it is assumed to contain both the certificate and
    # the key to convert to an X509 certificate. If both the certificate and
    # key are specified, Envconsul will automatically combine them into an X509
    # certificate for you.
    cert = "/path/to/client/cert"
    key  = "/path/to/client/key"

    # This is the path to the certificate authority to use as a CA. This is
    # useful for self-signed certificates or for organizations using their own
    # internal certificate authority.
    ca_cert = "/path/to/ca"

    # This is the path to a directory of PEM-encoded CA cert files. If both
    # `ca_cert` and `ca_path` is specified, `ca_cert` is preferred.
    ca_path = "path/to/certs/"

    # This sets the SNI server name to use for validation.
    server_name = "my-server.com"
  }
}

# This block defines the configuration the the child process to execute and
# manage.
exec {
  # This is the command to execute as a child process. There can be only one
  # command per process.
  command = "/usr/bin/app"

  # This is a random splay to wait before killing the command. The default
  # value is 0 (no wait), but large clusters should consider setting a splay
  # value to prevent all child processes from reloading at the same time when
  # data changes occur. When this value is set to non-zero, Envconsul will wait
  # a random period of time up to the splay value before killing the child
  # process. This can be used to prevent the thundering herd problem on
  # applications that do not gracefully reload.
  splay = "5s"

  env {
    # This specifies if the child process should not inherit the parent
    # process's environment. By default, the child will have full access to the
    # environment variables of the parent. Setting this to true will send only
    # the values specified in `custom_env` to the child process.
    pristine = false

    # This specifies additional custom environment variables in the form shown
    # below to inject into the child's runtime environment. If a custom
    # environment variable shares its name with a system environment variable,
    # the custom environment variable takes precedence. Even if pristine,
    # whitelist, or blacklist is specified, all values in this option
    # are given to the child process.
    custom = ["PATH=$PATH:/etc/myapp/bin"]

    # This specifies a list of environment variables to exclusively include in
    # the list of environment variables exposed to the child process. If
    # specified, only those environment variables matching the given patterns
    # are exposed to the child process. These strings are matched using Go's
    # glob function, so wildcards are permitted.
    whitelist = ["CONSUL_*"]

    # This specifies a list of environment variables to exclusively prohibit in
    # the list of environment variables exposed to the child process. If
    # specified, any environment variables matching the given patterns will not
    # be exposed to the child process, even if they are whitelisted. The values
    # in this option take precedence over the values in the whitelist.
    # These strings are matched using Go's glob function, so wildcards are
    # permitted.
    blacklist = ["VAULT_*"]
  }

  # This defines the signal sent to the child process when Envconsul is
  # gracefully shutting down. The application should begin a graceful cleanup.
  # If the application does not terminate before the `kill_timeout`, it will
  # be terminated (effectively "kill -9"). The default value is shown below.
  kill_signal = "SIGTERM"

  # This defines the amount of time to wait for the child process to gracefully
  # terminate when Envconsul exits. After this specified time, the child
  # process will be force-killed (effectively "kill -9"). The default value is
  # "30s".
  kill_timeout = "2s"
}

# This is the signal to listen for to trigger a graceful stop. The default
# value is shown below. Setting this value to the empty string will cause it
# to not listen for any graceful stop signals.
kill_signal = "SIGINT"

# This is the log level. If you find a bug in Envconsul, please enable debug or
# trace logs so we can help identify the issue. This is also available as a
# command line flag.
log_level = "warn"

# This is the maximum interval to allow "stale" data. By default, only the
# Consul leader will respond to queries; any requests to a follower will
# forward to the leader. In large clusters with many requests, this is not as
# scalable, so this option allows any follower to respond to a query, so long
# as the last-replicated data is within these bounds. Higher values result in
# less cluster load, but are more likely to have outdated data.
max_stale = "10m"

# This is the path to store a PID file which will contain the process ID of the
# Envconsul process. This is useful if you plan to send custom signals
# to the process.
pid_file = "/path/to/pid"

# This specifies a prefix in Consul to watch. This may be specified multiple
# times to watch multiple prefixes, and the bottom-most prefix takes
# precedence, should any values overlap.
prefix {
  # This tells Envconsul to use a custom formatter when printing the key. The
  # value between `{{ key }}` will be replaced with the key.
  format = "custom_{{ key }}"

  # This tells Envconsul to not prefix the keys with their parent "folder".
  no_prefix = false

  # This is the path of the key in Consul or Vault from which to read data.
  path = "foo/bar"
}

# This tells Envconsul to not include the parent processes' environment when
# launching the child process.
pristine = false

# This is the signal to listen for to trigger a reload event. The default
# value is shown below. Setting this value to the empty string will cause it
# to not listen for any reload signals.
reload_signal = "SIGHUP"

# This tell Envconsul to remove any non-standard values from environment
# variable keys and replace them with underscores.
sanitize = false

# This specifies a secret in Vault to watch. This may be specified multiple
# times to watch multiple secrets, and the bottom-most secret takes
# precedence, should any values overlap.
secret {
  # See `prefix` as they are the same options.
}

# This block defines the configuration for connecting to a syslog server for
# logging.
syslog {
  # This enables syslog logging. Specifying any other option also enables
  # syslog logging.
  enabled = true

  # This is the name of the syslog facility to log to.
  facility = "LOCAL5"
}

# This tells Envconsul to convert environment variable keys to uppercase (which
# is more common and a bit more standard).
upcase = false

# This denotes the start of the configuration section for Vault. All values
# contained in this section pertain to Vault.
vault {
  # This is the address of the Vault leader. The protocol (http(s)) portion
  # of the address is required.
  address = "https://vault.service.consul:8200"

  # This is the grace period between lease renewal and secret re-acquisition.
  # When renewing a secret, if the remaining lease is less than or equal to the
  # configured grace, Envconsul will request a new credential. This
  # prevents Vault from revoking the credential at expiration and Envconsul
  # having a stale credential.
  #
  # Note: If you set this to a value that is higher than your default TTL or
  # max TTL, Envconsul will always read a new secret!
  grace = "15s"

  # This is the token to use when communicating with the Vault server.
  # Like other tools that integrate with Vault, Envconsul makes the
  # assumption that you provide it with a Vault token; it does not have the
  # incorporated logic to generate tokens via Vault's auth methods.
  #
  # This value can also be specified via the environment variable VAULT_TOKEN.
  token = "abcd1234"

  # This tells Envconsul that the provided token is actually a wrapped
  # token that should be unwrapped using Vault's cubbyhole response wrapping
  # before being used. Please see Vault's cubbyhole response wrapping
  # documentation for more information.
  unwrap_token = true

  # This option tells Envconsul to automatically renew the Vault token given.
  # If you are unfamiliar with Vault's architecture, Vault requires tokens be
  # renewed at some regular interval or they will be revoked. Envconsul will
  # automatically renew the token at half the lease duration of the token. The
  # default value is true, but this option can be disabled if you want to renew
  # the Vault token using an out-of-band process.
  #
  # Note that secrets specified as a prefix are always renewed, even if this
  # option is set to false. This option only applies to the top-level Vault
  # token itself.
  renew_token = true

  # This section details the retry options for connecting to Vault. Please see
  # the retry options in the Consul section for more information (they are the
  # same).
  retry {
    # ...
  }

  # This section details the SSL options for connecting to the Vault server.
  # Please see the SSL options in the Consul section for more information (they
  # are the same).
  ssl {
    # ...
  }
}

# This is the quiescence timers; it defines the minimum and maximum amount of
# time to wait for the cluster to reach a consistent state before relaunching
# the app. This is useful to enable in systems that have a lot of flapping,
# because it will reduce the the number of times the app is restarted.
wait {
  min = "5s"
  max = "10s"
}
```

Note that not all fields are required. If you are not retrieving secrets from
Vault, you do not need to specify a Vault configuration section. Similarly, if
you are not logging to syslog, you do not need to specify a syslog
configuration.

For additional security, tokens may also be read from the environment using the
`CONSUL_TOKEN` or `VAULT_TOKEN` environment variables respectively. It is highly
recommended that you do not put your tokens in plain-text in a configuration
file.

Instruct Envconsul to use a configuration file with the `-config` flag:

```shell
$ envconsul -config "config.hcl"
```

This argument may be specified multiple times to load multiple configuration
files. The right-most configuration takes the highest precedence. If the path to
a directory is provided (as opposed to the path to a file), all of the files in
the given directory will be merged in
[lexical order](http://golang.org/pkg/path/filepath/#Walk), recursively. Please
note that symbolic links are _not_ followed.

**Commands specified on the CLI take precedence over a config file!**

**Vault secrets always take precedence over consul prefixes. This is to mitigate
**a security vulnerability!**

### Signals

By default, almost all signals are proxied to the child process, with some
exceptions. There are multiple configuration options related to signals.

- `kill_signal` - This is the signal that Envconsul should listen to to kill
  _itself_. This is useful when you want your application to respond to a
  different signal than the child process.

- `reload_signal` - This is the signal that Envconsul should listen to to reload
  its own configuration. This is useful when using configuration files. This
  signal will not be proxied to the child process if configured. By specifying
  this as the empty string, Envconsul will not listen for reload signals.

- `exec.kill_signal` - This is the signal that Envconsul will send to the
  child process to gracefully terminate it. This is the signal that your child
  application listens to for graceful termination.

- `exec.reload_signal` - This signal exists, but it is never used. Configuring
  it will have no affect, since Envconsul does not send reload signals to  child
  processes.

## Examples

### Redis

Redis is a command key-value storage engine. If Redis is configured to read the
given environment variables, you can use `envconsul` to start and manage the
process:

```shell
# Ensure "daemonize no" is set in the redis configuration first.
$ envconsul \
  -consul demo.consul.io \
  -prefix redis/config \
  redis-server [opts...]
```

### Env

This example is a great way to see `envconsul` in action. In practice, it is
unlikely to be a useful use of envconsul though:

```shell
$ envconsul \
  -consul=demo.consul.io \
  -prefix redis/config \
  -once \
  env
ADDRESS=1.2.3.4
PORT=55
```

We can also ask envconsul to poll for configuration changes and automatically
restart the process:

```
$ envconsul \
  -consul=demo.consul.io \
  -prefix redis/config \
  python -c 'import os, time; print os.environ; time.sleep(1000);'
{ 'ADDRESS': '1.2.3.4', 'PORT': '55' }
-----
{ 'ADDRESS': '1.2.3.4' }
-----
{ 'ADDRESS': '1.2.3.4', 'MAXCONNS': '50' }
-----
```

### Vault

With the Vault integration, it is possible to pull secrets from Vault directly
into the environment using envconsul. The only restriction is that the data must
be "flat" and all keys and values must be strings or string-like values.
envconsul will return an error if you try to read from a value that returns a
map, for example.

First, you must add the vault address and token information to the configuration
file. It is not possible to specify these values via the command line:

```hcl
vault {
  address = "https://vault.service.consul:8200"
  token   = "abcd1234" # May also be specified via the envvar VAULT_TOKEN
  renew   = true

  ssl {
    enabled = true
    verify  = true
    cert    = "/path/to/client/cert.pem"
    ca_cert = "/path/to/ca/cert.pem"
  }
}
```

Assuming a secret exists at secret/passwords that was created like so:

```sh
$ vault write secret/passwords username=foo password=bar
```

envconsul can pull those values into the environment:

```text
$ envconsul \
    -config="./config.hcl" \
    -secret="secret/passwords" \
    env

secret_passwords_username=foo
secret_passwords_password=bar
```

Notice that the environment variables are prefixed with the path. The slashes in
the path are converted to underscores, followed by the key:

```text
secret/passwords     => secret_passwords
mysql/creds/readonly => mysql_creds_readonly
```

This behavior may be disabled by setting `no_prefix`

```javascript
secret {
  no_prefix = true
  path      = "secret/passwords"
}

username=foo
password=bar
```

You can also apply key transformations to the data:

```text
$ envconsul \
    -config="./config.hcl" \
    -secret="mysql/creds/readonly" \
    -upcase \
    env

MYSQL_CREDS_READONLY_USERNAME=root-aefa635a-18
MYSQL_CREDS_READONLY_PASSWORD=132ae3ef-5a64-7499-351e-bfe59f3a2a21
```

It is highly encouraged that you specify the format for vault keys to include a
common prefix, like:

```hcl
secret {
  path   = "secret/passwords"
  format = "secret_{{ key }}"
}
```

The format string is passed to the go formatter and "{{ key }}" dictates where
the key will go. This will help filter out the environment when execing to a
child-process, for example.


## Debugging

Envconsul can print verbose debugging output. To set the log level for
Envconsul, use the `-log-level` flag:

```shell
$ envconsul -log-level info ...
```

```text
<timestamp> [INFO] (cli) received redis from Watcher
<timestamp> [INFO] (cli) invoking Runner
# ...
```

You can also specify the level as debug:

```shell
$ envconsul -log-level debug ...
```

```text
<timestamp> [DEBUG] (cli) creating Runner
<timestamp> [DEBUG] (cli) creating Consul API client
<timestamp> [DEBUG] (cli) creating Watcher
<timestamp> [DEBUG] (cli) looping for data
<timestamp> [DEBUG] (watcher) starting watch
<timestamp> [DEBUG] (watcher) all pollers have started, waiting for finish
<timestamp> [DEBUG] (redis) starting poll
<timestamp> [DEBUG] (service redis) querying Consul with &{...}
<timestamp> [DEBUG] (service redis) Consul returned 2 services
<timestamp> [DEBUG] (redis) writing data to channel
<timestamp> [DEBUG] (redis) starting poll
<timestamp> [INFO] (cli) received redis from Watcher
<timestamp> [INFO] (cli) invoking Runner
<timestamp> [DEBUG] (service redis) querying Consul with &{...}
# ...
```

## Contributing

To build and install Envconsul locally, you will need to install the Docker
engine:

- [Docker for Mac](https://docs.docker.com/engine/installation/mac/)
- [Docker for Windows](https://docs.docker.com/engine/installation/windows/)
- [Docker for Linux](https://docs.docker.com/engine/installation/linux/ubuntulinux/)

Clone the repository:

```shell
$ git clone https://github.com/hashicorp/envconsul.git
```

To compile the `envconsul` binary for your local machine:

```shell
$ make dev
```

If you want to run the tests:

```shell
$ make test
```

Or to run a specific test in the suite:

```shell
go test ./... -run SomeTestFunction_name
```

[12-factor]: https://12factor.net/
[envchain]: https://github.com/sorah/envchain
[envdir]: https://pypi.python.org/pypi/envdir
[consul]: https://www.consul.io/ "Service discovery and configuration made easy"
[hcl]: https://github.com/hashicorp/hcl "HashiCorp Configuration Language (HCL)"
[go]: https://golang.org "Go programming language"
[releases]: https://releases.hashicorp.com/envconsul/ "Envconsul releases page"
[vault]: https://www.vaultproject.io/ "A tool for managing secrets"
