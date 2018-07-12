test-unit:
	@(go list ./... | grep -v "vendor/" | grep -v "e2e" | xargs -n1 go test -v -cover)

test-e2e:
	@(go build -o e2e/tailor-test)
	@(go test -v -cover github.com/opendevstack/tailor/e2e)
	@(cd e2e && rm tailor-test)

test: test-unit test-e2e

fmt:
	@(gofmt -w .)

build: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o tailor_linux_amd64 -v github.com/opendevstack/tailor

build-darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o tailor_darwin_amd64 -v github.com/opendevstack/tailor

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o tailor_windows_amd64.exe -v github.com/opendevstack/tailor
