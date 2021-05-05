#!/usr/bin/dumb-init /bin/sh
set -e

# Note above that we run dumb-init as PID 1 in order to reap zombie processes
# as well as forward signals to all processes in its session. Normally, sh
# wouldn't do either of these functions so we'd leak zombies as well as do
# unclean termination of all our sub-processes.

# ENVCONSUL_DATA_DIR is exposed as a volume for possible persistent storage.
# ENVCONSUL_CONFIG_DIR isn't exposed as a volume but you can compose additional
# config files in there if you use this image as a base, or use
# ENVCONSUL_LOCAL_CONFIG below.
ENVCONSUL_DATA_DIR=/envconsul/data
ENVCONSUL_CONFIG_DIR=/envconsul/config

# You can also set the ENVCONSUL_LOCAL_CONFIG environemnt variable to pass some
# configuration JSON without having to bind any volumes.
if [ -n "$ENVCONSUL_LOCAL_CONFIG" ]; then
  echo "$ENVCONSUL_LOCAL_CONFIG" > "$ENVCONSUL_CONFIG_DIR/local-config.hcl"
fi

# If the user is trying to run envconsul directly with some arguments, then
# pass them to envconsul.
if [ "${1:0:1}" = '-' ]; then
  set -- /bin/envconsul "$@"
fi

# Set the configuration directory
if [ "$1" = '/bin/envconsul' ]; then
  shift
  set -- /bin/envconsul \
    -config="$ENVCONSUL_CONFIG_DIR" \
    "$@"
fi

exec "$@"
