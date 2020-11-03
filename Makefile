SHELL = /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

## Run unit tests.
test-unit: imports
	@(go list ./... | grep -v "vendor/" | grep -v "e2e" | xargs -n1 go test -cover)
.PHONY: test-unit

## Run E2E tests (real binary against temporal, unique project in real cluster).
test-e2e: imports internal/test/e2e/tailor-test
	@(go test -v -cover -timeout 20m github.com/opendevstack/tailor/internal/test/e2e)
.PHONY: test-e2e

## Run all tests.
test: test-unit test-e2e
.PHONY: test

## Run goimports.
imports:
	@(goimports -w .)
.PHONY: imports

## Run gofmt.
fmt:
	@(gofmt -w .)
.PHONY: fmt

## Run golangci-lint.
lint:
	@(go mod download && golangci-lint run)
.PHONY: lint

## Install binary on current platform.
install: imports
	@(cd cmd/tailor && go install -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)")
.PHONY: install

## Build binaries for all supported platforms.
build: imports build-linux build-darwin build-windows
.PHONY: build

## Build Linux binary.
build-linux: imports
	cd cmd/tailor && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)" -o tailor-linux-amd64
.PHONY: build-linux

## Build macOS binary.
build-darwin: imports
	cd cmd/tailor && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)" -o tailor-darwin-amd64
.PHONY: build-darwin

## Build Windows binary.
build-windows: imports
	cd cmd/tailor && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)" -o tailor-windows-amd64.exe
.PHONY: build-windows

internal/test/e2e/tailor-test: cmd/tailor/main.go go.mod go.sum pkg/cli/* pkg/commands/* pkg/openshift/* pkg/utils/*
	@(echo "Generating E2E test binary ...")
	@(cd cmd/tailor && go build -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)" -o ../../internal/test/e2e/tailor-test)

### HELP
### Based on https://gist.github.com/prwhite/8168133#gistcomment-2278355.
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:|^# .*/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  %-35s %s\n", helpCommand, helpMessage; \
		} else { \
			printf "\n"; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)
.PHONY: help
