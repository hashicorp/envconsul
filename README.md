envetcd
=========

[![wercker status](https://app.wercker.com/status/7af57352c44ae04c4d6546ecf86a6deb/s "wercker status")](https://app.wercker.com/project/bykey/7af57352c44ae04c4d6546ecf86a6deb) [![Coverage Status](https://img.shields.io/coveralls/zvelo/envetcd.svg)](https://coveralls.io/r/zvelo/envetcd)

envetcd provides a convienent way to populate values from [etcd][] into a child process environment using the `envetcd` daemon.

The daemon `envetcd` allows applications to be configured with environmental variables, without having knowledge about the existence of etcd. This makes it especially easy to configure applications throughout all your environments: development, testing, production, etc.

envetcd was forked from [envconsul][] which was inspired by [envdir][] in its simplicity, name, and function.

Installation
------------
You can download a released `envetcd` artifact from [the envetcd release page][Releases] on GitHub. If you wish to compile from source, you will need to have buildtools and [Go][] installed:

```shell
$ git clone https://github.com/zvelo/envetcd.git
$ cd envetcd
$ make
```

This process will create `envetcd` which make be invoked as a binary.


Usage
-----

### Options

| Option        | Environment Variable   | Default          | Description                                                               |
| ------------- | ---------------------- | ---------------- | ------------------------------------------------------------------------- |
| `peers`       | `$ENVETCD_PEERS`       | `127.0.0.1:4001` | a comma-delimited list of machine addresses in the cluster                |
| `ca-file`     | `$ENVETCD_CA_FILE`     |                  | certificate authority file                                                |
| `cert-file`   | `$ENVETCD_CERT_FILE`   |                  | tls client certificate file                                               |
| `key-file`    | `$ENVETCD_KEY_FILE`    |                  | tls client key file                                                       |
| `hostname`    | `$HOSTNAME`            |                  | computer hostname for host specific configuration                         |
| `system`      | `$ENVETCD_SYSTEM`      |                  | system name for system specific configuration                             |
| `service`     | `$ENVETCD_SERVICE`     |                  | service name for service specific configuration                           |
| `prefix`      | `$ENVETCD_PREFIX`      | `/config`        | etcd prefix for all keys                                                  |
| `log-level`   | `$ENVETCD_LOG_LEVEL`   | `WARN`           | set log level (`DEBUG`, `INFO`, `WARN`, `ERR`)                            |
| `no-sync`     | `$ENVETCD_NO_SYNC`     | `false`          | don't synchronize cluster information before sending request              |
| `clean-env`   | `$ENVETCD_CLEAN_ENV`   | `false`          | don't inherit any environment variables other than those pulled from etcd |
| `no-sanitize` | `$ENVETCD_NO_SANITIZE` | `false`          | don't remove bad characters from environment keys                         |
| `no-upcase`   | `$ENVETCD_NO_UPCASE`   | `false`          | don't convert all environment keys to uppercase                           |

### Command Line

The CLI interface supports all of the options detailed above.

Query the default etcd instance, rending all the keys in `/config/redis`, and printing the environment.

```shell
$ envetcd \
  --no-sanitize \
  --no-upcase \
  --service redis \
  --system storage \
  env
```

Query a local etcd instance, converting special characters in keys to undercores and uppercasing the keys:

```shell
$ envetcd \
  --peers 127.0.0.1:4001 \
  --prefix /config \
  --service redis \
  env
```

Examples
--------
### Redis
Redis is a command key-value storage engine. If Redis is configured to read the given environment variables, you can use `envetcd` to start and manage the process:

```shell
$ envetcd \
  --service redis \
  service redis start
```

### Env
This example is a great way to see `envetcd` in action. In practice, it is unlikely to be a useful use of envetcd though:

```shell
$ envetcd \
  --service redis \
  env
ADDRESS=1.2.3.4
PORT=55
```

Contributing
------------
To hack on envetcd, you will need a modern [Go][] environment. To compile the `envetcd` binary and run the test suite, simply execute:

```shell
$ make
```

This will compile the `envetcd` binary.

If you want to run the tests:

```shell
$ make test
```

Or to run a specific test in the suite:

```shell
go test ./... -run SomeTestFunction_name
```

Submit Pull Requests and Issues to the envetcd project on GitHub.

[envconsul]: https://github.com/hashicorp/envconsul "Read and set environmental variables for processes from Consul"
[etcd]: https://github.com/coreos/etcd "A highly-available key value store for shared configuration and service discovery"
[envdir]: http://cr.yp.to/daemontools/envdir.html "envdir"
[Releases]: https://github.com/zvelo/envetcd/releases "envetcd releases page"
[Go]: http://golang.org "Go the language"
