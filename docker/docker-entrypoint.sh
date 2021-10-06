#!/bin/sh

# Don't use dumb-init as it isn't required and the end-user has the option
# to set it via the `--init` option.

set -e

# If the user is trying to run envconsul directly with some arguments,
# then pass them to envconsul.
# On alpine /bin/sh is busybox which supports this bashism.
if [ "${1:0:1}" = '-' ]
then
    set -- /bin/envconsul "$@"
fi

# MUST exec here for envconsul to replace the shell as PID 1 in order
# to properly propagate signals from the OS to the envconsul process.
exec "$@"
