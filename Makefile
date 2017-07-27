# Metadata about this makefile and position
MKFILE_PATH := $(lastword $(MAKEFILE_LIST))
CURRENT_DIR := $(patsubst %/,%,$(dir $(realpath $(MKFILE_PATH))))

# Ensure GOPATH
GOPATH ?= $(HOME)/go

# List all our actual files, excluding vendor
GOFILES ?= $(shell go list $(TEST) | grep -v /vendor/)

# Tags specific for building
GOTAGS ?=

# Number of procs to use
GOMAXPROCS ?= 4

# Get the project metadata
GOVERSION := 1.8.1
PROJECT := $(CURRENT_DIR:$(GOPATH)/src/%=%)
OWNER := $(notdir $(patsubst %/,%,$(dir $(PROJECT))))
NAME := $(notdir $(PROJECT))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
VERSION := $(shell awk -F\" '/Version/ { print $$2; exit }' "${CURRENT_DIR}/version/version.go")
EXTERNAL_TOOLS = \
	github.com/golang/dep/cmd/dep

# Current system information
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Default os-arch combination to build
XC_OS ?= darwin freebsd linux netbsd openbsd solaris windows
XC_ARCH ?= 386 amd64 arm
XC_EXCLUDE ?= darwin/arm solaris/386 solaris/arm windows/arm

# GPG Signing key (blank by default, means no GPG signing)
GPG_KEY ?=

# List of ldflags
LD_FLAGS ?= \
	-s \
	-w \
	-X ${PROJECT}/version.Name=${NAME} \
	-X ${PROJECT}/version.GitCommit=${GIT_COMMIT}

# List of Docker targets to build
DOCKER_TARGETS ?= alpine scratch

# List of tests to run
TEST ?= ./...

# Create a cross-compile target for every os-arch pairing. This will generate
# a make target for each os/arch like "make linux/amd64" as well as generate a
# meta target for compiling everything.
define make-xc-target
  $1/$2:
  ifneq (,$(findstring ${1}/${2},$(XC_EXCLUDE)))
		@printf "%s%20s %s\n" "-->" "${1}/${2}:" "${PROJECT} (excluded)"
  else
		@printf "%s%20s %s\n" "-->" "${1}/${2}:" "${PROJECT}"
		@docker run \
			--interactive \
			--rm \
			--dns="8.8.8.8" \
			--volume="${CURRENT_DIR}:/go/src/${PROJECT}" \
			--workdir="/go/src/${PROJECT}" \
			"golang:1.8" \
			env \
				CGOENABLED=0 \
				GOOS=${1} \
				GOARCH=${2} \
				go build \
				  -a \
					-o="pkg/${1}_${2}/${NAME}" \
					-ldflags "${LD_FLAGS}"
  endif
  .PHONY: $1/$2

  $1:: $1/$2
  .PHONY: $1

  all:: $1/$2
  .PHONY: all
endef
$(foreach goarch,$(XC_ARCH),$(foreach goos,$(XC_OS),$(eval $(call make-xc-target,$(goos),$(goarch)))))

# bootstrap installs the necessary go tools for development or build.
bootstrap:
	@echo "==> Bootstrapping ${PROJECT}"
	@for t in ${EXTERNAL_TOOLS}; do \
		echo "--> Installing $$t" ; \
		go get -u "$$t"; \
	done
.PHONY: bootstrap

# deps updates all dependencies for this project.
deps:
	@echo "==> Updating deps for ${PROJECT}"
	@dep ensure -update
	@dep prune
.PHONY: deps

# dev builds and installs the project locally.
dev:
	@echo "==> Installing ${NAME} for ${GOOS}/${GOARCH}"
	@env \
		-i \
		PATH="${PATH}" \
		CGO_ENABLED="0" \
		GOOS="${GOOS}" \
		GOARCH="${GOARCH}" \
		GOPATH="${GOPATH}" \
		go install -ldflags "${LD_FLAGS}"
.PHONY: dev

# Create a docker compile target for each container. This will create
# docker/scratch, etc.
define make-docker-target
  docker/$1:
		@echo "==> Building ${1} Docker container for ${PROJECT}"
		@docker build \
			--rm \
			--force-rm \
			--no-cache \
			--squash \
			--compress \
			--file="docker/${1}/Dockerfile" \
			--build-arg="LD_FLAGS=${LD_FLAGS}" \
			--tag="${OWNER}/${NAME}:${1}" \
			--tag="${OWNER}/${NAME}:${VERSION}-${1}" \
			"${CURRENT_DIR}"
  .PHONY: docker/$1

  docker:: docker/$1
  .PHONY: docker
endef
$(foreach target,$(DOCKER_TARGETS),$(eval $(call make-docker-target,$(target))))

# Create a docker push target for each container. This will create
# docker-push/scratch, etc.
define make-docker-push-target
  docker-push/$1:
		@echo "==> Pushing ${1} to Docker registry"
		@docker push "${OWNER}/${NAME}:${1}"
		@docker push "${OWNER}/${NAME}:${VERSION}-${1}"
  .PHONY: docker-push/$1

  docker-push:: docker-push/$1
  .PHONY: docker-push
endef
$(foreach target,$(DOCKER_TARGETS),$(eval $(call make-docker-push-target,$(target))))

# test runs the test suite.
test:
	@echo "==> Testing ${NAME}"
	@go test -timeout=30s -parallel=20 -tags="${GOTAGS}" ${GOFILES} ${TESTARGS}
.PHONY: test

# test-race runs the test suite.
test-race:
	@echo "==> Testing ${NAME} (race)"
	@go test -timeout=60s -race -tags="${GOTAGS}" ${GOFILES} ${TESTARGS}
.PHONY: test-race
