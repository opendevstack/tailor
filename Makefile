SHELL = /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

prepare-test:
	@(oc whoami &> /dev/null || oc cluster up)
.PHONY: prepare-test

test-unit: imports
	@(go list ./... | grep -v "vendor/" | grep -v "e2e" | xargs -n1 go test -cover)
.PHONY: test-unit

test-e2e: imports internal/test/e2e/tailor-test
	@(go test -v -cover -timeout 20m github.com/opendevstack/tailor/internal/test/e2e)
.PHONY: test-e2e

test: test-unit test-e2e
.PHONY: test

imports:
	@(goimports -w .)
.PHONY: imports

fmt:
	@(gofmt -w .)
.PHONY: fmt

lint:
	@(go mod download && golangci-lint run)
.PHONY: lint

install: imports
	@(cd cmd/tailor && go install -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)")
.PHONY: install

build: imports build-linux build-darwin build-windows
.PHONY: build

build-linux: imports
	cd cmd/tailor && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)" -o tailor-linux-amd64
.PHONY: build-linux

build-darwin: imports
	cd cmd/tailor && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)" -o tailor-darwin-amd64
.PHONY: build-darwin

build-windows: imports
	cd cmd/tailor && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)" -o tailor-windows-amd64.exe
.PHONY: build-windows

internal/test/e2e/tailor-test: cmd/tailor/main.go go.mod go.sum pkg/cli/* pkg/commands/* pkg/openshift/* pkg/utils/*
	@(echo "Generating E2E test binary ...")
	@(cd cmd/tailor && go build -gcflags "all=-trimpath=$(CURDIR);$(shell go env GOPATH)" -o ../../internal/test/e2e/tailor-test)
