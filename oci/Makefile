# TODO: add version
ifneq (,$(wildcard ./VERSION))
LDFLAGS := -ldflags "-X github.com/tektoncd/experimental/oci/pkg/cmd/version.clientVersion=`cat VERSION`"
endif

ifneq ($(RELEASE_VERSION),)
LDFLAGS := -ldflags "-X github.com/tektoncd/experimental/oci/pkg/cmd/version.clientVersion=$(RELEASE_VERSION)"
endif

all: bin/oci

FORCE:

vendor:
	@go mod vendor

.PHONY: cross
cross: amd64 386 arm arm64 ## build cross platform binaries

.PHONY: amd64
amd64:
	GOOS=linux GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/oci-linux-amd64 ./cmd/oci
	GOOS=windows GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/oci-windows-amd64 ./cmd/oci
	GOOS=darwin GOARCH=amd64 go build -mod=vendor $(LDFLAGS) -o bin/oci-darwin-amd64 ./cmd/oci

.PHONY: 386
386:
	GOOS=linux GOARCH=386 go build -mod=vendor $(LDFLAGS) -o bin/oci-linux-386 ./cmd/oci
	GOOS=windows GOARCH=386 go build -mod=vendor $(LDFLAGS) -o bin/oci-windows-386 ./cmd/oci
	GOOS=darwin GOARCH=386 go build -mod=vendor $(LDFLAGS) -o bin/oci-darwin-386 ./cmd/oci

.PHONY: arm
arm:
	GOOS=linux GOARCH=arm go build -mod=vendor $(LDFLAGS) -o bin/oci-linux-arm ./cmd/oci

.PHONY: arm64
arm64:
	GOOS=linux GOARCH=arm64 go build -mod=vendor $(LDFLAGS) -o bin/oci-linux-arm64 ./cmd/oci

bin/%: cmd/% vendor FORCE
	go build -mod=vendor $(LDFLAGS) -v -o $@ ./$<

check: lint test

.PHONY: test
test: test-unit ## run all tests

.PHONY: lint
lint: ## run linter(s)
	@echo "Linting..."
	@golangci-lint run ./... --timeout 5m

.PHONY: clean
clean: ## clean build artifacts
	rm -fR bin VERSION

.PHONY: fmt ## formats teh god code(excludes vendors dir)
fmt:
	@go fmt $(go list ./... | grep -v /vendor/)

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
