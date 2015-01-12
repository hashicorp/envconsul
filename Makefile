alldirs=$(shell find . \( -path ./Godeps -o -path ./.git \) -prune -o -type d -print)
GODIRS=$(foreach dir, $(alldirs), $(if $(wildcard $(dir)/*.go),$(dir)))
GOFILES=$(foreach dir, $(alldirs), $(wildcard $(dir)/*.go))

ifeq ("$(CIRCLECI)", "true")
export GIT_BRANCH = $(CIRCLE_BRANCH)
endif

EXECUTABLE ?= envetcd

all: build

lint:
	golint ./...
	go vet ./...

test: $(GOFILES)
	godep go test -tags "$(TEST_TAGS)" -v ./...

coverage: .acc.out

.acc.out: $(GOFILES)
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

.coveralls-stamp: .acc.out
	@if [ -n "$(COVERALLS_REPO_TOKEN)" ]; then \
		goveralls -v -coverprofile=.acc.out -service circle-ci -repotoken $(COVERALLS_REPO_TOKEN); \
	fi
	@touch .coveralls-stamp

build: $(EXECUTABLE)

$(EXECUTABLE): $(GOFILES)
	godep go build -v -o $(EXECUTABLE)

$(EXECUTABLE)-linux-amd64: $(GO_FILES)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 godep go build -a -v -o $(EXECUTABLE)-linux-amd64

release: release-linux-amd64

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

clean:
	@rm -rf \
		./$(EXECUTABLE) \
		./$(EXECUTABLE)-linux-amd64 \
		./.acc.out \
		./.coveralls-stamp \
		./$(EXECUTABLE)-*.tgz \
		./release

save:
	@rm -rf ./Godeps
	godep save ./...
