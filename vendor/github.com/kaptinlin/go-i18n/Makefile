# go-i18n: Internationalization library for Go
# Set up GOBIN so that our binaries are installed to ./bin instead of $GOPATH/bin.
PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
export GOBIN = $(PROJECT_ROOT)/bin

GOLANGCI_LINT_BINARY := $(GOBIN)/golangci-lint
GOLANGCI_LINT_VERSION := $(shell $(GOLANGCI_LINT_BINARY) version --format short 2>/dev/null || $(GOLANGCI_LINT_BINARY) version --short 2>/dev/null || echo "not-installed")
REQUIRED_GOLANGCI_LINT_VERSION := $(shell cat .golangci.version 2>/dev/null || echo "2.4.0")

# Directories containing independent Go modules.
MODULE_DIRS = .

.PHONY: all
all: lint test

.PHONY: help
help: ## Show this help message
	@echo "go-i18n: Internationalization library for Go"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: clean
clean: ## Clean build artifacts and caches
	@echo "[clean] Cleaning build artifacts..."
	@rm -rf $(GOBIN)
	@go clean -cache -testcache

.PHONY: deps
deps: ## Download Go module dependencies
	@echo "[deps] Downloading dependencies..."
	@go mod download
	@go mod tidy

.PHONY: test
test: ## Run all tests
	@echo "[test] Running all tests..."
	@$(foreach mod,$(MODULE_DIRS),(cd $(mod) && go test ./...) &&) true

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "[test] Running unit tests..."
	@go test ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "[test] Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "[test] Coverage report generated: coverage.html"

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	@echo "[test] Running tests with verbose output..."
	@go test -race -v ./...

.PHONY: bench
bench: ## Run benchmarks
	@echo "[bench] Running benchmarks..."
	@go test -bench=. -benchmem ./...

.PHONY: lint
lint: golangci-lint tidy-lint ## Run all linters

.PHONY: install-golangci-lint
install-golangci-lint:
	@mkdir -p $(GOBIN)
	@if [ "$(GOLANGCI_LINT_VERSION)" != "$(REQUIRED_GOLANGCI_LINT_VERSION)" ]; then \
		echo "[lint] Installing golangci-lint v$(REQUIRED_GOLANGCI_LINT_VERSION) (current: $(GOLANGCI_LINT_VERSION))"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v$(REQUIRED_GOLANGCI_LINT_VERSION); \
		echo "[lint] golangci-lint v$(REQUIRED_GOLANGCI_LINT_VERSION) installed successfully"; \
	else \
		echo "[lint] golangci-lint v$(REQUIRED_GOLANGCI_LINT_VERSION) already installed"; \
	fi

.PHONY: golangci-lint
golangci-lint: install-golangci-lint ## Run golangci-lint
	@echo "[lint] Running $(shell $(GOLANGCI_LINT_BINARY) version)"
	@$(foreach mod,$(MODULE_DIRS), \
		(cd $(mod) && \
		echo "[lint] golangci-lint: $(mod)" && \
		$(GOLANGCI_LINT_BINARY) run --timeout=10m --path-prefix $(mod)) &&) true

.PHONY: tidy-lint
tidy-lint: ## Check if go.mod and go.sum are tidy
	@$(foreach mod,$(MODULE_DIRS), \
		(cd $(mod) && \
		echo "[lint] mod tidy: $(mod)" && \
		go mod tidy && \
		git diff --exit-code -- go.mod go.sum) &&) true

.PHONY: fmt
fmt: ## Format Go code
	@echo "[fmt] Formatting Go code..."
	@go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "[vet] Running go vet..."
	@go vet ./...


.PHONY: verify
verify: deps fmt vet lint test ## Run all verification steps (deps, format, vet, lint, test)
	@echo "[verify] All verification steps completed successfully ✅"