#
# Builder
#
FROM golang:1.12 AS builder
LABEL maintainer "John Eikenberry <jae@zhar.net>"

ARG LD_FLAGS
ARG GOTAGS

WORKDIR "/go/src/github.com/hashicorp/envconsul"

COPY . .

RUN \
  CGO_ENABLED="0" \
  go build -a -o "/envconsul" -ldflags "${LD_FLAGS}" -tags "${GOTAGS}"

#
# Final
#
FROM scratch
LABEL maintainer "John Eikenberry <jae@zhar.net>"

ADD "https://curl.haxx.se/ca/cacert.pem" "/etc/ssl/certs/ca-certificates.crt"
COPY --from=builder "/envconsul" "/bin/envconsul"
ENTRYPOINT ["/bin/envconsul"]
