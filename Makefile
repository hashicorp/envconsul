alldirs=$(shell find . \( -path ./Godeps -o -path ./.git \) -prune -o -type d -print)
GODIRS=$(foreach dir, $(alldirs), $(if $(wildcard $(dir)/*.go),$(dir)))
GOFILES=$(foreach dir, $(alldirs), $(wildcard $(dir)/*.go))

TEST_TAGS=test
BUILD_TAGS=

ifeq ("$(WERCKER)", "true")
TEST_TAGS  += wercker
BUILD_TAGS += wercker production
endif

EXECUTABLE ?= envetcd
ETCD_HOST ?= 127.0.0.1
ETCD_PORT ?= 4001
ETCD_PEER_PORT ?= 7001

all: build

lint:
	golint ./...
	go vet ./...

test:
	godep go test -tags "$(TEST_TAGS)" -v ./...

coverage:
	@echo "mode: set" > acc.out
	@for dir in $(GODIRS); do \
		cmd="godep go test -tags '$(TEST_TAGS)' -v -coverprofile=profile.out $$dir"; \
		eval $$cmd; \
		if test $$? -ne 0; then \
			exit 1; \
		fi; \
		if test -f profile.out; then \
			cat profile.out | grep -v "mode: set" >> acc.out; \
		fi; \
	done
	@rm -f ./profile.out

build: $(EXECUTABLE)

$(EXECUTABLE): $(GOFILES)
	godep go build -tags "$(BUILD_TAGS)" -v -o $(EXECUTABLE)

clean:
	@rm -f ./$(EXECUTABLE)

save:
	godep save ./...

etcd:
	@etcd -data-dir .cache/etcd -name $(EXECUTABLE) -addr=$(ETCD_HOST):$(ETCD_PORT) -peer-addr=$(ETCD_HOST):$(ETCD_PEER_PORT)
