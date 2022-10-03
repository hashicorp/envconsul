## envconsul CHANGELOG

## v0.13.1 (Oct 03, 2022)

BUG FIXES:
* Fix not passing through configuration setting properly to underlying consul-template config [[GH-312](https://github.com/hashicorp/envconsul/pull/312), [GH-310](https://github.com/hashicorp/envconsul/issues/310)]
* Fix issue with vault agent token file reading [[GH-314](https://github.com/hashicorp/envconsul/pull/314), [GH-275](https://github.com/hashicorp/envconsul/issues/275)]


## v0.13.0 (Jul 19, 2022)

BUG FIXES:
* Fix using an interactive shell as the command [[GH-306](https://github.com/hashicorp/envconsul/pull/306), [GH-305](https://github.com/hashicorp/envconsul/issues/305)]
* Fix command argument parsing [GH-304](https://github.com/hashicorp/envconsul/pull/304), [GH-297](https://github.com/hashicorp/envconsul/issues/297)]
* Version and help output to STDOUT [[GH-289](https://github.com/hashicorp/envconsul/pull/289), [GH-279](https://github.com/hashicorp/envconsul/issues/279)]

IMPROVEMENTS:
* Support Vault K8s authorization [[GH-281](https://github.com/hashicorp/envconsul/pull/281), [GH-274](https://github.com/hashicorp/envconsul/issues/274)]
* Allow multiple secrets with same name [[GH-296](https://github.com/hashicorp/envconsul/pull/296)]
* Don't log  Go scheduler signals [[GH-288](https://github.com/hashicorp/envconsul/pull/288), [GH-285](https://github.com/hashicorp/envconsul/issues/285)]
* Build darwin-arm64 binaries [[GH-276](https://github.com/hashicorp/envconsul/issues/276)]


## v0.12.1 (Nov 17, 2021)

BUG FIXES:

* Fix Vault lease duration default value [[GH-273](https://github.com/hashicorp/envconsul/pull/273), [GH-272](https://github.com/hashicorp/envconsul/issues/272)].

## v0.12.0 (Oct 07, 2021)

IMPROVEMENTS:
* New docker image [[GH-265](https://github.com/hashicorp/envconsul/pull/265), [GH-258](https://github.com/hashicorp/envconsul/issues/258), [GH-229](https://github.com/hashicorp/envconsul/issues/229)]
* Add support for Vault Namespaces [[GH-262](https://github.com/hashicorp/envconsul/pull/262), [GH-184](https://github.com/hashicorp/envconsul/issues/184)]
* New feature to allow only fetching a subset of keys from Vault Secret [[GH-259](https://github.com/hashicorp/envconsul/pull/259), [GH-235](https://github.com/hashicorp/envconsul/issues/235)]
* Ability to customize environment variable names [[GH-236](https://github.com/hashicorp/envconsul/pull/236), [GH-234](https://github.com/hashicorp/envconsul/issues/234)]
* Supports fetching service information [[GH-220](https://github.com/hashicorp/envconsul/pull/220)]

## v0.11.0 (Nov 30, 2020)

IMPROVEMENTS:

* Add support for Vault Agent token files [[GH-249](https://github.com/hashicorp/envconsul/pull/249)]
* Update whitelist/blacklist config options to allowlist/denylist with backward compatibility [[GH-246](https://github.com/hashicorp/envconsul/pull/246)]
* Update Consul Template dependency to v0.25.1 [[GH-245](https://github.com/hashicorp/envconsul/pull/245)]

BUG FIXES:

* Remove code/logic for working with (long deprecated) Vault grace [[GH-245](https://github.com/hashicorp/envconsul/pull/245)]

## v0.10.0 (Aug 10, 2020)

IMPROVEMENTS:

* Allow users to use ENV vars in prefix paths using Go templates [[GH-167](https://github.com/hashicorp/envconsul/pull/167)]

## v0.9.3 (Apr 27, 2020)

BUG FIXES:

* Fix renewing wrapped tokens, renew using unwrapped token [[GH-239](https://github.com/hashicorp/envconsul/pull/239), [GH-222](https://github.com/hashicorp/envconsul/issues/222)]

* Fix issue with secret/prefix entries with no path [[GH-240](https://github.com/hashicorp/envconsul/pull/240), [GH-165](https://github.com/hashicorp/envconsul/issues/165)]

DOCUMENTATION:

* Remove unused -exec-reload-signal from docs [[GH-238](https://github.com/hashicorp/envconsul/pull/238), [GH-237](https://github.com/hashicorp/envconsul/issues/237)]

## v0.9.2 (Jan 08, 2020)

SECURITY:

* Don't log values/secrets pulled from vault [[GH-226](https://github.com/hashicorp/envconsul/pull/226)]

BUG FIXES:

* Build Arm binaries with CGO enabled [[GH-227](https://github.com/hashicorp/envconsul/issues/227), [GH-228](https://github.com/hashicorp/envconsul/pull/228)]

## v0.9.1 (Nov 08, 2019)

SECURITY:

* curl and musl vulnerabilities in latest alpine docker image [[GH-223](https://github.com/hashicorp/envconsul/issues/223)]

## v0.9.0 (Aug 13, 2019)

IMPROVEMENTS:

* Migrated to use Go modules [[GH-216](https://github.com/hashicorp/envconsul/pull/216), [GH-217](https://github.com/hashicorp/envconsul/issues/217)]

## v0.8.0 (June 10, 2019)

BREAKING CHANGES:

  * When passing unevaluated, 'single quoted text' to -exec or as the command
    the shell variable interpolation and backtick evaluation will no longer be
    done. This was an undocumented 'feature' and was responsible for GH-180.

IMPROVEMENTS:

  * Added support for Vault KV v2 [GH-186 GH-211]
  * Respect no_prefix for consul keys [GH-190]
  * Respect exec.env config block [GH-155]

BUG FIXES:
  * Fix issue with shell interpolation [GH-180]
  * Fix issue with PID file not being created [GH-194]
  * Fix issue with Vault renew [GH-196]
  * Don't panic if secret value isn't a string [GH-166]

## v0.7.3 (January 22, 2018)

SECURITY:

  * Fixed an issue where the parent's environment could get supplied to the child
    process if `envconsul` is given an empty prefix, even when using `-pristine`
    [GH-159]

IMPROVEMENTS:

  * Compile using Go 1.9.2 [GH-158]

BUG FIXES:

  * Fixed Makefile to use Go version from the environment [GH-157]

## v0.7.2 (August 25, 2017)

IMPROVEMENTS:

  * Compile using Go 1.9.0

## v0.7.1 (August 7, 2017)

BUG FIXES:

  * Remove dynamic linking due to a missing underscore in CGO_ENABLED during
      compilation [GH-147]

## v0.7.0 (August 1, 2017)

BREAKING CHANGES:

  * `kill_signal` (configuration) and `-kill-signal` (CLI) now refer to the
    signal that _Envconsul_ should listen to for termination, _not_ the signal
    that Envconsul should send to the child process. Use `exec.kill_signal` or
    `-exec-kill-signal` to specify the command to send to the child process.

DEPRECATIONS:

  * (configuration) `consul` is now `consul { address = "..." }`.
  * (configuration) `auth` is now `consul { auth { ... } }`.
  * (configuration) `path` is deprecated and there is no configuration file
    replacement. Use the CLI option instead.
  * (configuration) `splay = "..."` is now `exec { splay = "..." }`.
  * (configuration) `retry = "..."` is now separately controlled via both the
    `consul` and `vault` stanzas to allow for additional configuration.
  * (configuration) `ssl {}` is now separately controlled via both the
    `consul` and `vault` stanzas to allow for additional configuration.
  * (configuration) `timeout = "..."` is now `exec { kill_timeout = "..." }`.
  * (configuration) `token = "..."` is now `consul { token = "..." }`.

  * (cli) `-auth` is now `-consul-auth`.
  * (cli) `-addr` is now `-consul-addr`.
  * (cli) `-splay` is now `-exec-splay`.
  * (cli) `-retry` is now `-consul-retry-*` and `-vault-retry-*`.
  * (cli) `-ssl-*` is now `-consul-ssl-*` and `-vault-ssl-*`.
  * (cli) `-timeout` is now `-exec-kill-timeout`.
  * (cli) `-token` is now `-consul-token`.

## v0.6.2 (Jan 13, 2017)

BREAKING CHANGES:

  * Remove deprecated way of specifying `prefixes` as an array in configuration.

IMPROVEMENTS:

  * Allow stripping parts of a secret path [GH-84, GH-113]

BUG FIXES:

  * Overwriting existing environment variables [GH-122]
  * Update Consul API library to stop logging secret data [GH-120]
  * Ensure returned value is not nil [GH-124]

## v0.6.1 (June 9, 2016)

IMPROVEMENTS:

  * Accept `Key` option in SSL config
  * Accept stdin - this allows the user to pipe and use interactively
    [GH-80, GH-75]

BUG FIXES:

  * Use gatedio in tests to avoid races
  * Document `pristine` flag
  * Load `VAULT_TOKEN` from the environment into the config [GH-99, GH-100]
  * Add `timeout` CLI flag parsing [GH-73]
  * Do not overwrite previous process when no data is returned [GH-85, GH-107]

## v0.6.0 (October 12, 2015)

FEATURES:

  * Add a configurable kill switch [GH-48]
  * Add `pristine` option to completely replace the environment for the command
    [GH-58]
  * Add support for Vault configuration and prefixes
  * Add support for custom key formatting
  * Add `splay` option for sleeping a random amount of time before re-spawing
    the child process [GH-53]

IMPROVEMENTS:

  * Improve documentation around command line vs configuration file parameters
    [GH-41]
  * Update to new Consul Template APIs which are more efficient
  * Match Makefile and semantics for other HashiCorp projects
  * Set a default max-stale value of 1s
  * Support reloading configuration on SIGHUP (but the signal will also be
    sent to the child process!)

BUG FIXES:

  * Fix config merging [GH-49]
  * Trim leading and trailing slashes from prefixes [GH-59]
  * Fix ignored `-ssl` flag [GH-51]
  * Remove noisy debug line [GH-55]
  * Properly handle the case where a command is missing [GH-61]

## v0.5.0 (February 19, 2015)

DEPRECATIONS:

  * Specifying the prefix before the command is deprecated, please use the
    `-prefix` key instead

FEATURES:

  * Add support for logging to syslog
  * Add `log_level` as a configuration file option and CLI option
  * Add support for basic HTTP authentication when connecting to Consul
  * Add support for connecting to Consul via SSL
  * Add support for specifying a custom retry interval when Consul is not
    available
  * Add support for specifying multiple prefixes using the new `-prefix` command
    line and configuration option (GH-27)
  * Add support for propagating select signals to the child process (GH-31)


IMPROVEMENTS:

  * Improve test coverage, specifically around command-line flag parsing
  * Use Consul Template's logging library for consistency (and get syslog
    logging for free)

BUG FIXES:

  * Fix a bug in the documentation where the environment would be reset
  * Raise an error when specifying a non-existent option in the configuration
    file

## v0.4.0 (February 5, 2015)

IMPROVEMENTS:

  * Allow `envconsul` to run when Consul is unavailable (GH-28)
  * Add `-max-stale` to specify envconsul may talk to non-leader Consul nodes
    if they are less than the maximum stale value (GH-36)

BUG FIXES:

  * Remove deprecated CLI and config options

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
