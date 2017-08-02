#
# Builder
#
FROM golang:1.8 AS builder
LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"

ARG LD_FLAGS

WORKDIR "/go/src/github.com/hashicorp/envconsul"

COPY . .

RUN \
  CGO_ENABLED="0" \
  go build -a -o "/envconsul" -ldflags "${LD_FLAGS}"

#
# Final
#
FROM scratch
LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"

ADD "https://curl.haxx.se/ca/cacert.pem" "/etc/ssl/certs/ca-certificates.crt"
COPY --from=builder "/envconsul" "/bin/envconsul"
ENTRYPOINT ["/bin/envconsul"]
