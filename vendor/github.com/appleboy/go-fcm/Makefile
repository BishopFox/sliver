GO ?= go

.PHONY: test
test:
	@$(GO) test -v -cover -coverprofile coverage.out ./... && echo "\n==>\033[32m Ok\033[m\n" || exit 1

clean:
	go clean -x -i ./...
