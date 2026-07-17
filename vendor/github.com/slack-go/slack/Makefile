.PHONY: help deps fmt lint test test-race test-integration

help:
	@echo ""
	@echo "Welcome to slack-go/slack make."
	@echo "The following commands are available:"
	@echo ""
	@echo "    make deps              : Fetch all dependencies"
	@echo "    make fmt               : Run go fmt to fix any formatting issues"
	@echo "    make lint              : Run golangci-lint for linting issues"
	@echo "    make test              : Run all short tests"
	@echo "    make test-race         : Run all tests with race condition checking"
	@echo "    make test-integration  : Run all tests without limiting to short"
	@echo ""
	@echo "    make pr-prep           : Run this before making a PR to run fmt, lint and tests"
	@echo ""

deps:
	@go mod tidy

fmt:
	@go fmt .

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1 run ./...

test:
	@go test -v -count=1 -timeout 300s -short ./...

test-race:
	@go test -v -count=1 -timeout 300s -short -race ./...

test-integration:
	@go test -v -count=1 -timeout 600s ./...

pr-prep: fmt lint test-race test-integration
