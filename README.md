envetcd
=========

envetcd provides a convienent way to populate values from [etcd][etcd] into an child process environment using the `envetcd` daemon.

The daemon `envetcd` allows applications to be configured with environmental variables, without having knowledge about the existence of etcd. This makes it especially easy to configure applications throughout all your environments: development, testing, production, etc.

envetcd is inspired by [envdir][] in its simplicity, name, and function.

Installation
------------
You can download a released `envetcd` artifact from [the envetcd release page][Releases] on GitHub. If you wish to compile from source, you will need to have buildtools and [Go][] installed:

```shell
$ git clone https://github.com/zvelo/envetcd.git
$ cd envetcd
$ make
```

This process will create `bin/envetcd` which make be invoked as a binary.


Usage
-----

### Options

| Option      | Required     | Description                                                                          |
| ----------- | ------------ | ------------------------------------------------------------------------------------ |
| `etcd`      | _(required)_ | The location of the etcd instance to query (may be an IP address or FQDN) with port. |
| `sanitize`  |              | Replace invalid characters in keys to underscores                                    |
| `upcase`    |              | Convert all environment variable keys to uppercase                                   |

### Command Line

The CLI interface supports all of the options detailed above.

Query the nyc1 demo Consul instance, rending all the keys in `config/redis`, and printing the environment.

```shell
$ envetcd \
  -etcd demo.consul.io \
  redis/config@nyc1 env
```

Query a local etcd instance, converting special characters in keys to undercores and uppercasing the keys:

```shell
$ envetcd \
  -etcd 127.0.0.1:4001 \
  -sanitize \
  -upcase \
  redis/config env
```

Examples
--------
### Redis
Redis is a command key-value storage engine. If Redis is configured to read the given environment variables, you can use `envetcd` to start and manage the process:

```shell
$ envetcd \
  -etcd demo.consul.io \
  redis/config service redis start
```

### Env
This example is a great way to see `envetcd` in action. In practice, it is unlikely to be a useful use of envetcd though:

```shell
$ envetcd \
  -etcd=demo.consul.io \
  redis/config env \
  -once
ADDRESS=1.2.3.4
PORT=55
```

We can also ask envetcd to poll for configuration changes and automatically restar the process:

```
$ envetcd \
  -etcd=demo.consul.io \
  redis/config /bin/sh -c "env; echo "-----"; sleep 1000"
ADDRESS=1.2.3.4
PORT=55
-----
ADDRESS=1.2.3.4
-----
ADDRESS=1.2.3.4
MAXCONNS=50
-----
```

Contributing
------------
To hack on envetcd, you will need a modern [Go][] environment. To compile the `envetcd` binary and run the test suite, simply execute:

```shell
$ make
```

This will compile the `envetcd` binary into `bin/envetcd` and run the test suite.

If you just want to run the tests:

```shell
$ make
```

Or to run a specific test in the suite:

```shell
go test ./... -run SomeTestFunction_name
```

Submit Pull Requests and Issues to the envetcd project on GitHub.

[etcd]: https://github.com/coreos/etcd "A highly-available key value store for shared configuration and service discovery"
[envdir]: http://cr.yp.to/daemontools/envdir.html "envdir"
[Releases]: https://github.com/zvelo/envetcd/releases "envetcd releases page"
[Go]: http://golang.org "Go the language"
