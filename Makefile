ALL_DIRS=$(shell find . \( -path ./Godeps -o -path ./.git \) -prune -o -type d -print)
GO_FILES=$(foreach dir, $(ALL_DIRS), $(wildcard $(dir)/*.go))
GO_PKGS=$(shell go list ./...)

ifeq ("$(CIRCLECI)", "true")
export GIT_BRANCH = $(CIRCLE_BRANCH)
endif

EXECUTABLE ?= envetcd

all: build

lint:
	golint ./...
	go vet ./...

test: $(GO_FILES)
	godep go test -v ./...

coverage: .acc.out

.acc.out: $(GO_FILES)
	@echo "mode: set" > .acc.out
	@for pkg in $(GO_PKGS); do \
		cmd="godep go test -v -coverprofile=profile.out $$pkg"; \
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

.coveralls-stamp: .acc.out
	@if [ -n "$(COVERALLS_REPO_TOKEN)" ]; then \
		goveralls -v -coverprofile=.acc.out -service circle-ci -repotoken $(COVERALLS_REPO_TOKEN); \
	fi
	@touch .coveralls-stamp

build: $(EXECUTABLE)

$(EXECUTABLE): $(GO_FILES)
	@godep go build -v -o $(EXECUTABLE) ./cmd/envetcd

$(EXECUTABLE)-linux-amd64: $(GO_FILES)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 godep go build -a -v -tags netgo -installsuffix netgo -o $(EXECUTABLE)-linux-amd64 ./cmd/envetcd

$(EXECUTABLE)-darwin-amd64: $(GO_FILES)
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 godep go build -a -v -tags netgo -installsuffix netgo -o $(EXECUTABLE)-darwin-amd64 ./cmd/envetcd

release: release-linux-amd64 release-darwin-amd64

release-linux-amd64: $(EXECUTABLE) $(EXECUTABLE)-linux-amd64
	$(eval VERSION=$(shell ./$(EXECUTABLE) -v | awk '{print $$3}'))
	@mkdir -p $(EXECUTABLE)-$(VERSION)-linux-amd64
	@cp ./README.md \
		./LICENSE \
		$(EXECUTABLE)-$(VERSION)-linux-amd64
	@cp \
		./$(EXECUTABLE)-linux-amd64 \
		$(EXECUTABLE)-$(VERSION)-linux-amd64/$(EXECUTABLE)
	@mkdir -p release
	@tar czf release/$(EXECUTABLE)-$(VERSION)-linux-amd64.tgz $(EXECUTABLE)-$(VERSION)-linux-amd64
	@rm -rf $(EXECUTABLE)-$(VERSION)-linux-amd64
	@cp $(EXECUTABLE)-linux-amd64 release/

release-darwin-amd64: $(EXECUTABLE) $(EXECUTABLE)-darwin-amd64
	$(eval VERSION=$(shell ./$(EXECUTABLE) -v | awk '{print $$3}'))
	@mkdir -p $(EXECUTABLE)-$(VERSION)-darwin-amd64
	@cp ./README.md \
		./LICENSE \
		$(EXECUTABLE)-$(VERSION)-darwin-amd64
	@cp \
		./$(EXECUTABLE)-darwin-amd64 \
		$(EXECUTABLE)-$(VERSION)-darwin-amd64/$(EXECUTABLE)
	@mkdir -p release
	@tar czf release/$(EXECUTABLE)-$(VERSION)-darwin-amd64.tgz $(EXECUTABLE)-$(VERSION)-darwin-amd64
	@rm -rf $(EXECUTABLE)-$(VERSION)-darwin-amd64
	@cp $(EXECUTABLE)-darwin-amd64 release/

clean:
	@rm -rf \
		./$(EXECUTABLE) \
		./$(EXECUTABLE)-linux-amd64 \
		./$(EXECUTABLE)-darwin-amd64 \
		./.acc.out \
		./.coveralls-stamp \
		./$(EXECUTABLE)-*.tgz \
		./release

save:
	@rm -rf ./Godeps
	godep save ./...
