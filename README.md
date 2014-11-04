envconsul
=========

envconsul provides a convienent way to populate values from [Consul][] into an child process environment using the `envconsul` daemon.

The daemon `envconsul` allows applications to be configured with environmental variables, without having knowledge about the existence of Consul. This makes it especially easy to configure applications throughout all your environments: development, testing, production, etc.

envconsul is inspired by [envdir][] in its simplicity, name, and function.

Installation
------------
You can download a released `envconsul` artifact from [the envconsul release page][Releases] on GitHub. If you wish to compile from source, you will need to have buildtools and [Go][] installed:

```shell
$ git clone https://github.com/hashicorp/envconsul.git
$ cd envconsul
$ make
```

This process will create `bin/envconsul` which make be invoked as a binary.


Usage
-----
### Options
| Option | Required | Description |
| ------ | -------- |------------ |
| `consul`    | _(required)_ | The location of the Consul instance to query (may be an IP address or FQDN) with port. |
| `token`     | | The [Consul API token][Consul ACLs]. |
| `config`    | | The path to a configuration file on disk, relative to the current working directory. Values specified on the CLI take precedence over values specified in the configuration file |
| `wait`      | | The `minimum(:maximum)` to wait before triggering a reload, separated by a colon (`:`). If the optional maximum value is omitted, it is assumed to be 4x the required minimum value. |
| `timeout`   | | The duration to wait for SIGTERM to finish before sending SIGKILL |
| `sanitize`  | | Replace invalid characters in keys to underscores |
| `upcase`    | | Convert all environment variable keys to uppercase |
| `once`      | | Run envconsul once and exit (as opposed to the default behavior of daemon). |

### Command Line
The CLI interface supports all of the options detailed above.

Query the nyc1 demo Consul instance, rending all the keys in `config/redis`, and printing the environment.

```shell
$ envconsul \
  -consul demo.consul.io \
  redis/config@nyc1 env
```

Query a local Consul instance, converting special characters in keys to undercores and uppercasing the keys:

```shell
$ envconsul \
  -consul 127.0.0.1:8500 \
  -sanitize \
  -upcase \
  redis/config env
```

### Configuration File
The envconsul configuration file is written in [HashiCorp Configuration Language (HCL)][HCL]. By proxy, this means the envconsul configuration file is JSON-compatible. For more information, please see the [HCL specification][HCL].

The Configuration file syntax interface supports all of the options detailed above.

```javascript
consul = "127.0.0.1:8500"
token = "abcd1234"
timeout = "5s"
sanitize = true
```

**Commands specified on the command line take precedence over those defined in a config file!**


Examples
--------
### Redis
Redis is a command key-value storage engine. If Redis is configured to read the given environment variables, you can use `envconsul` to start and manage the process:

```shell
$ envconsul \
  -consul demo.consul.io \
  redis/config service redis start
```

### Env
This example is a great way to see `envconsul` in action. In practice, it is unlikely to be a useful use of envconsul though:

```shell
$ envconsul \
  -consul=demo.consul.io \
  redis/config env \
  -once
ADDRESS=1.2.3.4
PORT=55
```

We can also ask envconsul to poll for configuration changes and automatically restar the process:

```
$ envconsul \
  -consul=demo.consul.io \
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
