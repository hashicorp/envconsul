### Builder
#
ARG GOVERSION=latest
FROM docker.mirror.hashicorp.services/golang:${GOVERSION} AS builder
LABEL maintainer "John Eikenberry <jae@zhar.net>"

ARG LD_FLAGS
ARG GOTAGS

WORKDIR "/go/src/github.com/hashicorp/envconsul"

COPY . .

RUN \
  CGO_ENABLED="0" \
  GO111MODULE=on \
  go build \
    -mod vendor \
    -a -o "/envconsul" \
    -ldflags "${LD_FLAGS}" -tags "${GOTAGS}"

### Final
#
FROM docker.mirror.hashicorp.services/alpine:latest
LABEL maintainer "John Eikenberry <jae@zhar.net>"

# UID and GID of envconsul user and group.
# These are the defaults, this makes them explicit and overridable.
ARG UID=100
ARG GID=1000

# Create a envconsul user and group first so the IDs get set the same way, even
# as the rest of this may change over time.
RUN addgroup -g ${GID} envconsul && \
    adduser -u ${UID} -S -G envconsul envconsul

# Set up certificates and base tools.
RUN apk add --no-cache ca-certificates dumb-init

# Copy the compiled binary from the builder
COPY --from=builder "/envconsul" "/bin/envconsul"

# The agent will be started with /envconsul/config as the configuration
# directory so you can add additional config files in that location.
RUN mkdir -p "/envconsul/data" && \
    mkdir -p "/envconsul/config" && \
    chown -R "envconsul:envconsul" "/envconsul"

# Expose the envconsul data directory as a volume since that's where shared
# results should be rendered.
VOLUME /envconsul/data

# The entry point script uses dumb-init as the top-level process to reap any
# zombie processes created by our app.
COPY "docker/alpine/docker-entrypoint.sh" "/bin/docker-entrypoint.sh"
RUN chmod +x "/bin/docker-entrypoint.sh"
ENTRYPOINT ["/bin/docker-entrypoint.sh"]
USER envconsul:envconsul

# Run our app by default
CMD ["/bin/envconsul"]
