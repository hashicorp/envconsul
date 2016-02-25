envconsul
=========

[![Latest Version](http://img.shields.io/github/release/hashicorp/envconsul.svg?style=flat-square)][release]
[![Build Status](http://img.shields.io/travis/hashicorp/envconsul.svg?style=flat-square)][travis]

[release]: https://github.com/hashicorp/envconsul/releases
[travis]: http://travis-ci.org/hashicorp/envconsul

envconsul provides a convenient way to populate values from [Consul][] into a child process environment using the `envconsul` daemon.

The daemon `envconsul` allows applications to be configured with environment variables, without having knowledge about the existence of Consul. This makes it especially easy to configure applications throughout all your environments: development, testing, production, etc.

envconsul is inspired by [envdir][] in its simplicity, name, and function.

**The documentation in this README corresponds to the master branch of envconsul. It may contain unreleased features or different APIs than the most recently released version. Please see the Git tag that corresponds to your version of envconsul for the proper documentation.**

Installation
------------
You can download a released `envconsul` artifact from [the envconsul release page][Releases] on GitHub. If you wish to compile from source, you will need to have buildtools and [Go][] installed:

```shell
$ git clone https://github.com/hashicorp/envconsul.git
$ cd envconsul
$ make
```

This process will create `bin/envconsul` which may be invoked as a binary.


Usage
-----
### Options
|       Option      | Description |
| ----------------- |------------ |
| `auth`            | The basic authentication username (and optional password), separated by a colon. There is no default value.
| `consul`*         | The location of the Consul instance to query (may be an IP address or FQDN) with port.
| `max-stale`       | The maximum staleness of a query. If specified, Consul will distribute work among all servers instead of just the leader. The default value is 0 (none).
| `ssl`             | Use HTTPS while talking to Consul. Requires the Consul server to be configured to serve secure connections. The default value is false.
| `ssl-verify`      | Verify certificates when connecting via SSL. This requires the use of `-ssl`. The default value is true.
| `syslog`          | Send log output to syslog (in addition to stdout and stderr). The default value is false.
| `syslog-facility` | The facility to use when sending to syslog. This requires the use of `-syslog`. The default value is `LOCAL0`.
| `token`           | The [Consul API token][Consul ACLs]. There is no default value.
| `kill-signal`     | Kill signal to send to child process. Defaults to `SIGTERM` but can be one of `SIGHUP,SIGTERM,SIGINT,SIGQUIT,SIGUSR1,SIGUSR2`
| `wait`            | The `minimum(:maximum)` to wait before rendering a command to fire, separated by a colon (`:`). If the optional maximum value is omitted, it is assumed to be 4x the required minimum value. There is no default value.
| `retry`           | The amount of time to wait if Consul returns an error when communicating with the API. The default value is 5 seconds.
| `prefix`          | A prefix to watch in Consul. This may be specified multiple times.
| `secret`          | A secret to watch in Vault. This may be specified multiple times.
| `sanitize`        | Replace invalid characters in keys to underscores.
| `separator`       | Use separator instead of underscore.
| `splay`           | The maximum time to wait before restarting the program, from which a random value is chosen.
| `upcase`          | Convert all environment variable keys to uppercase.
| `config`          | The path to a configuration file or directory of configuration files on disk, relative to the current working directory. Values specified on the CLI take precedence over values specified in the configuration file. There is no default value.
| `log-level`       | The log level for output. This applies to the stdout/stderr logging as well as syslog logging (if enabled). Valid values are "debug", "info", "warn", and "err". The default value is "warn".
| `pristine`       | Only use variables retrieved from consul, do not inherit existing environment variables.
| `once`            | Run envconsul once and exit (as opposed to the default behavior of daemon). _(CLI-only)_
| `version`         | Output version information and quit. _(CLI-only)_

\* = Required parameter

Multiple prefixes are merged in the order they are specified, with the right-most prefix taking precedence over its left siblings/ Vault secrets always take precedence over consul prefixes.

For example, consider:

```shell
$ envconsul -prefix global/config -prefix redis/config
```

In this example, the values of `redis` take precedence over the values in `global`. If they had the following structure:

```text
# Global
A=1
B=1
C=1

# Redis
A=2
```

The resulting environment would be:

```text
A=2
B=1
C=1
```

#### Custom Kill Signal

Envconsul by default will send the `SIGTERM` signal to the child process. If you want to override this to pass in a custom signal use the `kill-signal` config option. This option takes one of `SIGHUP,SIGTERM,SIGINT,SIGQUIT,SIGUSR1,SIGUSR2` or for Windows `SIGINT,SIGTERM,SIGQUIT`.

### Command Line
The CLI interface supports all of the options detailed above.

Query the nyc1 demo Consul instance, rending all the keys in `config/redis`, and printing the environment.

```shell
$ envconsul \
    -consul demo.consul.io \
    -prefix redis/config@nyc3 \
    env
```

Query a local Consul instance, converting special characters in keys to undercores and uppercasing the keys:

```shell
$ envconsul \
    -consul 127.0.0.1:8500 \
    -sanitize \
    -upcase \
    -prefix redis/config \
    env
```

### Configuration File
The envconsul configuration file is written in [HashiCorp Configuration Language (HCL)][HCL]. By proxy, this means the envconsul configuration file is JSON-compatible. For more information, please see the [HCL specification][HCL].

The Configuration file syntax interface supports all of the options detailed above, but the dashes are replaced with underscores.

```javascript
consul    = "127.0.0.1:8500"
token     = "abcd1234"
max_stale = "10m"
timeout   = "5s"
retry     = "10s"
sanitize  = true
splay     = "5s"

kill_signal = "SIGHUP"

vault {
  address = "https://vault.service.consul:8200"
  token   = "abcd1234" // May also be specified via the envvar VAULT_TOKEN
  renew   = true

  ssl {
    enabled = true
    verify  = true
    cert    = "/path/to/client/cert.pem"
    ca_cert = "/path/to/ca/cert.pem"
  }
}

prefix {
  path = "config/global"
}

prefix {
  path   = "config/redis"
  format = "prod_{{ key }}"
}

secret {
  path = "secret/creds"
}

auth {
  enabled = true
  username = "test"
  password = "test"
}

ssl {
  enabled = true
  verify = false
}

syslog {
  enabled = true
  facility = "LOCAL5"
}
```

Please note: Vault secrets always take precedence over consul prefixes. This is to mitigate a security vulnerability.

Examples
--------
### Redis
Redis is a command key-value storage engine. If Redis is configured to read the given environment variables, you can use `envconsul` to start and manage the process:

```shell
# Ensure "daemonize no" is set in the redis configuration first.
$ envconsul \
  -consul demo.consul.io \
  -prefix redis/config \
  redis-server [opts...]
```

### Env
This example is a great way to see `envconsul` in action. In practice, it is unlikely to be a useful use of envconsul though:

```shell
$ envconsul \
  -consul=demo.consul.io \
  -prefix redis/config \
  -once \
  env
ADDRESS=1.2.3.4
PORT=55
```

We can also ask envconsul to poll for configuration changes and automatically restart the process:

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

### Vault Secrets
With the Vault integration, it is possible to pull secrets from Vault directly into the environment using envconsul. The only restriction is that the data must be "flat" and all keys and values must be strings or string-like values. envconsul will return an error if you try to read from a value that returns a map, for example.

First, you must add the vault address and token information to the configuration file. It is not possible to specify these values via the command line:

```javascript
vault {
  address = "https://vault.service.consul:8200"
  token   = "abcd1234" // May also be specified via the envvar VAULT_TOKEN
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

```
$ vault write secret/passwords username=foo password=bar
```

envconsul can pull those values into the environment:

```
$ envconsul \
    -config="./config.hcl" \
    -secret="secret/passwords" \
    env

secret_passwords_username=foo
secret_passwords_password=bar
```

Notice that the environment variables are prefixed with the path. The slashes in the path are converted to underscores, followed by the key:

```text
secret/passwords     => secret_passwords
mysql/creds/readonly => mysql_creds_readonly
```

You can also apply key transformations to the data:

```
$ envconsul \
    -config="./config.hcl" \
    -secret="mysql/creds/readonly" \
    -upcase \
    env

MYSQL_CREDS_READONLY_USERNAME=root-aefa635a-18
MYSQL_CREDS_READONLY_PASSWORD=132ae3ef-5a64-7499-351e-bfe59f3a2a21
```

It is highly encouraged that you specify the format for vault keys to include a common prefix, like:

```javascript
secret {
  path   = "secret/passwords"
  format = "secret_{{ key }}"
}
```

The format string is passed to the go formatter and "%s" dictates where the key will go. This will help filter out the environment when execing to a child-process, for example.

Debugging
---------
envconsul can print verbose debugging output. To set the log level for envconsul, use the `-log-level` flag:

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
<timestamp> [INFO] (watcher) adding "storeKeyPrefix(redis/config)"
<timestamp> [DEBUG] (watcher) "storeKeyPrefix(redis/config)" starting
<timestamp> [DEBUG] (cli) looping for data
<timestamp> [DEBUG] (view) "storeKeyPrefix(redis/config)" starting fetch
<timestamp> [DEBUG] ("storeKeyPrefix(redis/config)") querying Consul with ...
<timestamp> [DEBUG] ("storeKeyPrefix(redis/config)") Consul returned 0 key pairs
<timestamp> [INFO] (view) "storeKeyPrefix(redis/config)" received data from consul
<timestamp> [INFO] (cli) received "storeKeyPrefix(redis/config)" from Watcher
<timestamp> [DEBUG] (cli) detected quiescence, starting timers
<timestamp> [DEBUG] (cli) looping for data
<timestamp> [DEBUG] (cli) quiescence minTimer fired, invoking Runner
<timestamp> [DEBUG] (view) "storeKeyPrefix(redis/config)" starting fetch
<timestamp> [DEBUG] ("storeKeyPrefix(redis/config)") querying Consul with ...
# ...
```

Quiescence
----------
If you have a large number of services that are in flux, you may want to specify a quiescence timer. This will prevent commands from running until a stable state is reached (or a maximum timeout you specify). You can specify the quiescence interval using the `-wait` flag on the command line:

```shell
envconsul -wait "10s:50s"
```

This tells envconsul to wait for a period of 10 seconds while we do not have data before running/restarting the command, but to wait no more than 50 seconds.


Contributing
------------
To hack on envconsul, you will need a modern [Go][] environment. To compile the `envconsul` binary and run the test suite, simply execute:

```shell
$ make
```

This will compile the `envconsul` binary into `bin/envconsul` and run the test suite.

If you just want to run the tests:

```shell
$ make
```

Or to run a specific test in the suite:

```shell
go test ./... -run SomeTestFunction_name
```

Submit Pull Requests and Issues to the envconsul project on GitHub.



[Consul]: http://consul.io/ "Service discovery and configuration made easy"
[envdir]: http://cr.yp.to/daemontools/envdir.html "envdir"
[Releases]: https://github.com/hashicorp/envconsul/releases "envconsul releases page"
[HCL]: https://github.com/hashicorp/hcl "HashiCorp Configuration Language (HCL)"
[Go]: http://golang.org "Go the language"
[Consul ACLs]: http://www.consul.io/docs/internals/acl.html "Consul ACLs"
