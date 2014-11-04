envconsul Changelog
===================

## v0.3.0 (November 4, 2014)

FEATURES:

  * Watch and reload by default - previously you needed to specify the `-reload`
  flag for envconsul to poll, but this is now the default behavior - you can
  restore the old behavior using the new `-once` flag
  * Leverage watching libraries from Consul Template
  * Unified command interface with Consul Template
  * Added support for quiescene using the new `-wait` option
  * Added support for Consul ACLs using the new `-token` option
  * Added support for reading configuration from file using the new `-config`
  option - the config file is HCL

IMPROVEMENTS:

  * Added `-timeout` parameter for specifying the interval to wait for SIGTERM
  to return before sending SIGKILL
  * Added `-version` flag to print the current version of envconsul
  * Added a full debug log tracer which can be set using `ENV_CONSUL_LOG=debug`
  * Drastically improved documentation with usage examples and feature
  documentation
  * Add significantly more test coverage (still not 100%, but more more
  thoroughly tested)

DEPRECATIONS:

  * `-addr` is deprecated in favor of `-consul` and will be removed in the next
  major release
  * `-dc` is deprecated in favor of using the inline `@dc` syntax and will be
  removed in the next major release
  * `-errExit`, `-terminate`, and `-reload` are all deprecated in favor of
  `-once`. envconsul now intelligently exits where appropriate


## v0.2.0 (July 16, 2014)

FEATURES:

  * Sanitize and upcase by default
  * If `-reload` is not set, don't watch keys
  * Preserve the original process environment

BUG FIXES:

  * Fixed issue with prefix listing missing final forward slash
  * Fixed panic condition on error

## v0.1.0 (May 13, 2014)

  * Initial release
