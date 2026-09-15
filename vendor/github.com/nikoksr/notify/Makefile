export GO111MODULE := on
export GOPROXY = https://proxy.golang.org,direct

###############################################################################
# DEPENDENCIES
###############################################################################

# Install all the build and lint dependencies
tools:
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/daixiang0/gci@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/vektra/mockery/v3@v3.6.1
	@go install github.com/segmentio/golines@latest
.PHONY: tools

###############################################################################
# TESTS
###############################################################################

test:
	@go build ./...
	@go test -failfast -race ./...
.PHONY: test

gen-coverage:
	@go test -race -covermode=atomic -coverprofile=coverage.out ./... > /dev/null
.PHONY: gen-coverage

coverage: gen-coverage
	@go tool cover -func coverage.out
.PHONY: coverage

coverage-html: gen-coverage
	@go tool cover -html=coverage.out -o cover.html
.PHONY: coverage-html

mock:
	@mockery
	@rm -f mock_notifier.go mock_option.go
.PHONY: mock

###############################################################################
# CODE HEALTH
###############################################################################

fmt:
	@goimports -w .
	@golines --shorten-comments -m 120 -w .
	@gofumpt -w -l .
	@gci write -s standard -s default -s "prefix(github.com/nikoksr/notify)" .
.PHONY: fmt

lint:
	@golangci-lint run ./...
.PHONY: lint

clean:
	@find . -name "mock_*" -type f -delete
.PHONY: clean

ci: lint test
.PHONY: ci

###############################################################################

.DEFAULT_GOAL := ci
