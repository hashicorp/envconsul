# Metadata about this makefile and position
MKFILE_PATH := $(lastword $(MAKEFILE_LIST))
CURRENT_DIR := $(patsubst %/,%,$(dir $(realpath $(MKFILE_PATH))))

# Tags specific for building
GOTAGS ?=

# Get the project metadata
PROJECT := $(shell go list -m)
OWNER := "hashicorp"
NAME := $(notdir $(PROJECT))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
VERSION := $(shell awk -F\" '/^[ \t]+Version/ { print $$2; exit }' "${CURRENT_DIR}/version/version.go")
PRERELEASE := $(shell awk -F\" '/^[ \t]+VersionPrerelease/ { print $$2; exit }' "${CURRENT_DIR}/version/version.go")

# Current system information
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# List of ldflags
LD_FLAGS ?= \
	-s -w \
	-X ${PROJECT}/version.Name=${NAME} \
	-X ${PROJECT}/version.GitCommit=${GIT_COMMIT}

# for CRT build process
version:
	@echo ${VERSION}${PRERELEASE}
.PHONY: version

# dev builds and installs the project locally.
dev:
	@echo "==> Installing ${NAME} for ${GOOS}/${GOARCH}"
	@env \
		CGO_ENABLED="0" \
		go install \
			-ldflags "${LD_FLAGS}" \
			-tags "${GOTAGS}"
.PHONY: dev

# dev docker builds
docker:
	@env CGO_ENABLED="0" go build -ldflags "${LD_FLAGS}" -o $(NAME)
	mkdir -p dist/linux/amd64/
	cp envconsul dist/linux/amd64/
	env DOCKER_BUILDKIT=1 docker build -t envconsul .
.PHONY: docker

# test runs the test suite.
test:
	@echo "==> Testing ${NAME}"
	@go test -timeout=30s -parallel=20 -failfast -tags="${GOTAGS}" ./... ${TESTARGS}
.PHONY: test

# test-race runs the test suite.
test-race:
	@echo "==> Testing ${NAME} (race)"
	@go test -timeout=60s -race -tags="${GOTAGS}" ./... ${TESTARGS}
.PHONY: test-race

# clean removes any previous binaries
clean:
	@rm -rf "${CURRENT_DIR}/dist/"
	@rm -f "envconsul"
.PHONY: clean

# noop command to get build pipeline working
dev-tree:
	@true
.PHONY: dev-tree
