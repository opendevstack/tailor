test-unit:
	@(go list ./... | grep -v "vendor/" | grep -v "e2e" | xargs -n1 go test -v -cover)

test-e2e:
	@(go build -o e2e/ocdiff-test)
	@(go test -v -cover github.com/michaelsauter/ocdiff/e2e)
	@(cd e2e && rm ocdiff-test)

test: test-unit test-e2e

fmt:
	@(gofmt -w .)
