TEST?=./...
NAME?=$(shell basename "${CURDIR}")
VERSION = $(shell awk -F\" '/^const Version/ { print $$2; exit }' main.go)

default: test

# bin generates the binaries for all platforms.
bin: generate
	@sh -c "'${CURDIR}/scripts/build.sh' '${NAME}'"

# dev creates binares for testing locally - they are put into ./bin and $GOPATH.
dev: generate
	@DEV=1 sh -c "'${CURDIR}/scripts/build.sh' '${NAME}'"

# dist creates the binaries for distibution.
dist:
	@sh -c "'${CURDIR}/scripts/dist.sh' '${NAME}' '${VERSION}'"

# test runs the test suite and vets the code.
test: generate
	@echo "==> Running tests..."
	@go list $(TEST) \
		| grep -v "github.com/hashicorp/${NAME}/vendor" \
		| xargs -n1 go test -timeout=60s -parallel=10 ${TESTARGS}

# testrace runs the race checker
testrace: generate
	@echo "==> Running tests (race)..."
	@go list $(TEST) \
		| grep -v "github.com/hashicorp/${NAME}/vendor" \
		| xargs -n1 go test -timeout=60s -race ${TESTARGS}

# updatedeps installs all the dependencies needed to run and build.
updatedeps:
	@echo "==> Updating dependencies..."
	@echo "    Cleaning previous dependencies..."
	@rm -rf vendor/
	@echo "    Updating to newest dependencies..."
	@go get -f -t -u ./...
	@echo "    Saving dependencies..."
	godep save

# generate runs `go generate` to build the dynamically generated source files.
generate:
	@echo "==> Generating..."
	@find . -type f -name '.DS_Store' -delete
	@go list ./... \
		| grep -v "github.com/hashicorp/${NAME}/vendor" \
		| xargs -n1 go generate

# bootstrap installs the necessary go tools for development/build.
bootstrap:
	@echo "==> Bootstrapping..."
	go get -u github.com/tools/godep
	go get -u github.com/mitchellh/gox

.PHONY: default bin dev dist test testrace updatedeps vet generate bootstrap
