#
# Builder
#
FROM golang:1.8 AS builder
LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"

ARG GOOS=""
ARG GOARCH=""
ARG PROJECT=""
ARG OWNER=""
ARG NAME=""
ARG LD_FLAGS=""

ENV GOOS="${GOOS}"
ENV GOARCH="${GOARCH}"
ENV PROJECT="${PROJECT}"
ENV OWNER="${OWNER}"
ENV NAME="${NAME}"
ENV LD_FLAGS="${LD_FLAGS}"

WORKDIR "/go/src/${PROJECT}"

COPY . .

RUN \
  CGO_ENABLED="0" \
  GOOS="${GOOS}" \
  GOARCH="${GOARCH}" \
  go build -a -o "/vault-service-broker" -ldflags "${LD_FLAGS}"
#
# #
# # Final
# #
# FROM scratch
# LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"
#
# ADD "https://curl.haxx.se/ca/cacert.pem" "/etc/ssl/certs/ca-certificates.crt"
# COPY --from=builder /vault-service-broker /vault-service-broker
# ENTRYPOINT ["/vault-service-broker"]
