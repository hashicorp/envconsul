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

test: .dep-stamp $(GOFILES)
	godep go test -tags "$(TEST_TAGS)" -v ./...

coverage: .acc.out

.acc.out: .dep-stamp $(GOFILES)
	@echo "mode: set" > .acc.out
	@for dir in $(GODIRS); do \
		cmd="godep go test -tags '$(TEST_TAGS)' -v -coverprofile=profile.out $$dir"; \
		eval $$cmd; \
		if test $$? -ne 0; then \
			exit 1; \
		fi; \
		if test -f profile.out; then \
			cat profile.out | grep -v "mode: set" >> .acc.out; \
		fi; \
	done
	@rm -f ./profile.out

coveralls: .coveralls-stamp

.coveralls-stamp: .coveralls-dep-stamp .acc.out
	if [ -n "$(WERCKER_GIT_BRANCH)" ]; then \
		export GIT_BRANCH=$$WERCKER_GIT_BRANCH; \
	fi
	if [ -n "$(COVERALLS_REPO_TOKEN)" ]; then \
		goveralls -v -coverprofile=.acc.out -service wercker -repotoken $(COVERALLS_REPO_TOKEN); \
	fi
	@touch .coveralls-stamp

build: $(EXECUTABLE)

$(EXECUTABLE): .dep-stamp $(GOFILES)
	godep go build -tags "$(BUILD_TAGS)" -v -o $(EXECUTABLE)

dist: $(EXECUTABLE)
	$(eval VERSION=$(shell ./$(EXECUTABLE) -v | awk '{print $$3}'))
	@mkdir -p $(EXECUTABLE)-$(VERSION)
	@cp ./README.md \
		./LICENSE \
		./$(EXECUTABLE) \
		$(EXECUTABLE)-$(VERSION)
	@tar czf $(EXECUTABLE)-$(VERSION).tgz $(EXECUTABLE)-$(VERSION)
	@rm -rf $(EXECUTABLE)-$(VERSION)

clean:
	@rm -f ./$(EXECUTABLE) \
		./.acc.out \
		./.dep-stamp \
		./.coveralls-stamp \
		./.coveralls-dep-stamp \
		./$(EXECUTABLE)-*.tgz

save: .dep-stamp
	godep save ./...

.dep-stamp:
	# go get should not be used except for godep
	# godep should provide all dependencies
	# missing dependencies are a legitimate build failure
	go get -v github.com/tools/godep
	@touch .dep-stamp

.coveralls-dep-stamp:
	# these are needed for coverage testing
	go get -v github.com/axw/gocov/gocov
	go get -v github.com/joshuarubin/goveralls
	@touch .coveralls-dep-stamp

etcd:
	@etcd -data-dir .cache/etcd -name $(EXECUTABLE) -addr=$(ETCD_HOST):$(ETCD_PORT) -peer-addr=$(ETCD_HOST):$(ETCD_PEER_PORT)
