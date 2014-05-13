# envconsul

envconsul sets environmental variables for processes by reading them
from [Consul's K/V store](http://www.consul.io).

envconsul allows applications to be configured with environmental variables,
without having to be knowledgable about the existence of Consul. This makes
it especially easy to configure applications throughout all your
environments: development, testing, production, etc.

envconsul is inspired by [envdir](http://cr.yp.to/daemontools/envdir.html)
in its simplicity, name, and function.

## Usage

Download a release from the
[releases page](#).

Then just run `envconsul`. We run the example below against our
[NYC demo server](http://nyc1.demo.consul.io). This lets you set
keys/values in a public place to just quickly test envconsul. Note
that the demo server will clear the k/v store every 30 minutes.

After setting the `prefix/FOO` key to "bar" on the demo server,
we can see it work:

```
$ envconsul -addr="nyc1.demo.consul.io:80" prefix env
FOO=bar
```

We can also ask envconsul to watch for any configuration changes
and restart our process:

```
$ envconsul -addr="nyc1.demo.consul.io:80" -reload \
  prefix /bin/sh -c "env; echo "-----"; sleep 1000"
FOO=bar
-----
FOO=baz
-----
FOO=baz
BAR=foo
```

The above output happened by setting keys and values within
the online demo UI while envconsul was running.
