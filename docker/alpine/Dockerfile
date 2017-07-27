#
# Builder
#
FROM golang:1.8 AS builder
LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"

ARG LD_FLAGS=""
ENV LD_FLAGS="${LD_FLAGS}"

WORKDIR "/go/src/github.com/hashicorp/envconsul"

COPY . .

RUN \
  CGO_ENABLED="0" \
  go build -a -o "/envconsul" -ldflags "${LD_FLAGS}"

#
# Final
#
FROM alpine:latest
LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"

RUN \
  apk --no-cache add ca-certificates && \
  update-ca-certificates

COPY --from=builder "/envconsul" "/envconsul"
ENTRYPOINT ["/envconsul"]
