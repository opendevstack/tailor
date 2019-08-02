
.PHONY: test test-unit test-e2e imports fmt lint install build build-darwin build-linux build-windows

test-unit: imports
	@(go list ./... | grep -v "vendor/" | grep -v "e2e" | xargs -n1 go test -cover)

test-e2e: imports internal/test/e2e/tailor-test
	@(go test -v -cover github.com/opendevstack/tailor/internal/test/e2e)

test: test-unit test-e2e

imports:
	@(goimports -w .)

fmt:
	@(gofmt -w .)

lint:
	@(go mod download && golangci-lint run)

install: imports
	@(cd cmd/tailor && go install)

build: imports build-linux build-darwin build-windows

build-linux: imports
	cd cmd/tailor && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o tailor-linux-amd64

build-darwin: imports
	cd cmd/tailor && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o tailor-darwin-amd64

build-windows: imports
	cd cmd/tailor && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o tailor-windows-amd64.exe

internal/test/e2e/tailor-test: cmd/tailor/main.go go.mod go.sum pkg/*
	@(echo "Generating E2E test binary")
	@(cd cmd/tailor && go build -o ../../internal/test/e2e/tailor-test)
