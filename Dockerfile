ARG BUILD_IMAGE="golang:stretch"
FROM ${BUILD_IMAGE} AS builder
ARG OS="darwin"
ARG ARCH="amd64"
ARG VERSION="local"
WORKDIR /go/src/github.com/hashicorp/envconsul
COPY . ./
RUN CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} go build -o envconsul \
      -ldflags "-s -w -X github.com/hashicorp/envconsul/version.Name=envconsul -X github.com/hashicorp/envconsul/version.GitCommit=${VERSION}" \
      -tags ""
RUN mv envconsul /envconsul
