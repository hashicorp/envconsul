### Builder
#
ARG GOVERSION=latest
FROM golang:${GOVERSION} AS builder
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
FROM scratch
LABEL maintainer "John Eikenberry <jae@zhar.net>"

ADD "https://curl.haxx.se/ca/cacert.pem" "/etc/ssl/certs/ca-certificates.crt"
COPY --from=builder "/envconsul" "/bin/envconsul"
ENTRYPOINT ["/bin/envconsul"]
