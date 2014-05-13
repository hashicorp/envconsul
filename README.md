# envconsul

envconsul sets environmental variables for processes by reading them
from [Consul's K/V store](http://www.consul.io).

envconsul allows applications to be configured with environmental variables,
without having to be knowledgable about the existence of Consul. This makes
it especially easy to configure applications throughout all your
environments: development, testing, production, etc.

envconsul is inspired by [envdir](http://cr.yp.to/daemontools/envdir.html)
in its simplicity, name, and function.
