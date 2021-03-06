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

# This is the release of https://github.com/hashicorp/docker-base to pull in
# order to provide HashiCorp-built versions of basic utilities like dumb-init
ARG DOCKER_BASE_VERSION="0.0.4"

# This is the location of the releases.
ARG HASHICORP_RELEASES="https://releases.hashicorp.com"

# UID and GID of envconsul user and group.
# These are the defaults, this makes them explicit and overridable.
ARG UID=100
ARG GID=1000

# Create a envconsul user and group first so the IDs get set the same way, even
# as the rest of this may change over time.
RUN addgroup -g ${GID} envconsul && \
    adduser -u ${UID} -S -G envconsul envconsul

# Set up certificates and base tools.
RUN apk add --no-cache ca-certificates curl gnupg libcap openssl && \
    mkdir -p /etc/ssl/certs/ && \
    update-ca-certificates --fresh && \
    gpg --keyserver keys.gnupg.net --recv-keys 91A6E7F85D05C65630BEF18951852D87348FFC4C && \
    mkdir -p /tmp/build && \
    cd /tmp/build && \
    curl -sO ${HASHICORP_RELEASES}/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_linux_amd64.zip && \
    curl -sO ${HASHICORP_RELEASES}/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS && \
    curl -sO ${HASHICORP_RELEASES}/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS.sig && \
    gpg --batch --verify docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS.sig docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS && \
    grep ${DOCKER_BASE_VERSION}_linux_amd64.zip docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS | sha256sum -c && \
    unzip docker-base_${DOCKER_BASE_VERSION}_linux_amd64.zip && \
    cp bin/dumb-init /bin && \
    cd /tmp && \
    rm -rf /tmp/build && \
    apk del gnupg openssl && \
    rm -rf /root/.gnupg

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
