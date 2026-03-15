# Taskfile Migration Plan for Golang Monorepo

## Overview

This document outlines the plan to migrate all packages in the golang monorepo from Makefile to Taskfile.yml. The migration has been successfully tested on the `jsonschema` package.

## Migration Strategy

### Approach
- **Manual migration** for each package to ensure accuracy and maintain all functionality
- **Incremental rollout** - migrate one package at a time
- **Verification** - test all tasks before removing Makefile

### Benefits of Taskfile over Makefile
1. **Better cross-platform support** - Works consistently on Windows, macOS, and Linux
2. **Cleaner syntax** - YAML-based configuration is more readable
3. **Built-in features** - Dependencies, status checks, file watching, and more
4. **Better task discovery** - `task --list` provides formatted help
5. **Parallel execution** - Tasks can run in parallel using `deps`
6. **Smart caching** - Skip tasks when sources haven't changed

## Successful Test Case: jsonschema

### Migration Results
✅ All Makefile targets successfully converted to Taskfile tasks
✅ All tests pass (`task test`)
✅ Linting works correctly (`task lint`)
✅ Benchmarks run successfully (`task bench`)
✅ All verification steps complete (`task verify`)
✅ Makefile removed after verification

### Key Conversions

#### 1. Variables
**Makefile:**
```makefile
PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
export GOBIN = $(PROJECT_ROOT)/bin
REQUIRED_GOLANGCI_LINT_VERSION := $(shell cat .golangci.version 2>/dev/null || echo "2.9.0")
```

**Taskfile:**
```yaml
vars:
  PROJECT_ROOT:
    sh: pwd
  GOBIN: '{{.PROJECT_ROOT}}/bin'
  REQUIRED_GOLANGCI_LINT_VERSION:
    sh: cat .golangci.version 2>/dev/null || echo "2.9.0"

env:
  GOBIN: '{{.GOBIN}}'
```

#### 2. Simple Tasks
**Makefile:**
```makefile
.PHONY: test
test:
	@echo "[test] Running all tests..."
	@go test -race ./...
```

**Taskfile:**
```yaml
test:
  desc: Run all tests with race detection
  cmds:
    - echo "Running all tests..."
    - go test -race ./...
```

#### 3. Dependencies
**Makefile:**
```makefile
.PHONY: lint
lint: golangci-lint tidy-lint
```

**Taskfile:**
```yaml
lint:
  desc: Run all linters
  deps:
    - golangci-lint
    - tidy-lint
```

#### 4. Conditional Execution
**Makefile:**
```makefile
.PHONY: install-golangci-lint
install-golangci-lint:
	@if [ "$(GOLANGCI_LINT_VERSION)" != "$(REQUIRED_GOLANGCI_LINT_VERSION)" ]; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL ... | sh -s -- -b $(GOBIN) v$(REQUIRED_GOLANGCI_LINT_VERSION); \
	fi
```

**Taskfile:**
```yaml
install-golangci-lint:
  desc: Install golangci-lint with the required version
  cmds:
    - mkdir -p {{.GOBIN}}
    - |
      if [ "{{.GOLANGCI_LINT_VERSION}}" != "{{.REQUIRED_GOLANGCI_LINT_VERSION}}" ]; then
        echo "Installing golangci-lint v{{.REQUIRED_GOLANGCI_LINT_VERSION}}..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b {{.GOBIN}} v{{.REQUIRED_GOLANGCI_LINT_VERSION}}
      fi
  status:
    - test "{{.GOLANGCI_LINT_VERSION}}" = "{{.REQUIRED_GOLANGCI_LINT_VERSION}}"
```

## Standard Taskfile Template

Based on the jsonschema migration, here's the standard template for Go packages:

```yaml
version: '3'

vars:
  PROJECT_ROOT:
    sh: pwd
  GOBIN: '{{.PROJECT_ROOT}}/bin'
  GOLANGCI_LINT_BINARY: '{{.GOBIN}}/golangci-lint'
  REQUIRED_GOLANGCI_LINT_VERSION:
    sh: cat .golangci.version 2>/dev/null || echo "2.9.0"
  GOLANGCI_LINT_VERSION:
    sh: '{{.GOLANGCI_LINT_BINARY}} version --format short 2>/dev/null || {{.GOLANGCI_LINT_BINARY}} version --short 2>/dev/null || echo "not-installed"'

env:
  GOBIN: '{{.GOBIN}}'

tasks:
  default:
    desc: Run lint and test
    cmds:
      - task: lint
      - task: test

  help:
    desc: Show this help message
    cmds:
      - echo "Package Name"
      - echo "Available targets:"
      - task --list
    silent: true

  clean:
    desc: Clean build artifacts and caches
    cmds:
      - echo "Cleaning build artifacts..."
      - rm -rf {{.GOBIN}}
      - go clean -cache -testcache || true

  deps:
    desc: Download Go module dependencies
    cmds:
      - echo "Downloading dependencies..."
      - go mod download
      - go mod tidy

  test:
    desc: Run all tests with race detection
    cmds:
      - echo "Running all tests..."
      - go test -race ./...

  test-coverage:
    desc: Run tests with coverage report
    cmds:
      - echo "Running tests with coverage..."
      - go test -race -coverprofile=coverage.out ./...
      - go tool cover -html=coverage.out -o coverage.html
      - echo "Coverage report generated at coverage.html"

  test-verbose:
    desc: Run tests with verbose output
    cmds:
      - echo "Running tests with verbose output..."
      - go test -race -v ./...

  bench:
    desc: Run benchmarks
    cmds:
      - echo "Running benchmarks..."
      - go test -bench=. -benchmem ./...

  lint:
    desc: Run all linters
    deps:
      - golangci-lint
      - tidy-lint

  install-golangci-lint:
    desc: Install golangci-lint with the required version
    cmds:
      - mkdir -p {{.GOBIN}}
      - |
        if [ "{{.GOLANGCI_LINT_VERSION}}" != "{{.REQUIRED_GOLANGCI_LINT_VERSION}}" ]; then
          echo "Installing golangci-lint v{{.REQUIRED_GOLANGCI_LINT_VERSION}} (current: {{.GOLANGCI_LINT_VERSION}})..."
          curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b {{.GOBIN}} v{{.REQUIRED_GOLANGCI_LINT_VERSION}}
          echo "golangci-lint v{{.REQUIRED_GOLANGCI_LINT_VERSION}} installed successfully"
        fi
    status:
      - test "{{.GOLANGCI_LINT_VERSION}}" = "{{.REQUIRED_GOLANGCI_LINT_VERSION}}"

  golangci-lint:
    desc: Run golangci-lint
    deps:
      - install-golangci-lint
    cmds:
      - '{{.GOLANGCI_LINT_BINARY}} version'
      - echo "Running golangci-lint..."
      - '{{.GOLANGCI_LINT_BINARY}} run --timeout=10m --path-prefix .'

  tidy-lint:
    desc: Check if go.mod and go.sum are tidy
    cmds:
      - echo "Checking go.mod and go.sum are tidy..."
      - go mod tidy
      - git diff --exit-code -- go.mod go.sum

  fmt:
    desc: Format Go code
    cmds:
      - echo "Formatting Go code..."
      - go fmt ./...

  vet:
    desc: Run go vet
    cmds:
      - echo "Running go vet..."
      - go vet ./...

  verify:
    desc: Run all verification steps (deps, format, vet, lint, test)
    cmds:
      - task: deps
      - task: fmt
      - task: vet
      - task: lint
      - task: test
      - echo "All verification steps completed successfully ✅"
```

## Migration Checklist for Each Package

### Pre-Migration
- [ ] Read the existing Makefile
- [ ] Identify all targets and their dependencies
- [ ] Note any package-specific customizations
- [ ] Check for any special variables or environment settings

### Migration Steps
1. [ ] Create `Taskfile.yml` based on the standard template
2. [ ] Convert all Makefile targets to Task tasks
3. [ ] Preserve all custom logic and package-specific features
4. [ ] Test each task individually:
   - [ ] `task test`
   - [ ] `task lint`
   - [ ] `task bench`
   - [ ] `task clean`
   - [ ] `task deps`
   - [ ] `task fmt`
   - [ ] `task vet`
   - [ ] `task verify`
5. [ ] Verify `task --list` shows all tasks with descriptions
6. [ ] Run `task default` to ensure the default workflow works
7. [ ] Delete the Makefile
8. [ ] Commit changes with descriptive message

### Post-Migration
- [ ] Update any CI/CD pipelines that reference `make` commands
- [ ] Update documentation (README.md, CLAUDE.md) if needed
- [ ] Notify team members of the change

## Common Patterns and Solutions

### 1. Looping Over Directories
**Makefile:**
```makefile
MODULE_DIRS = . subdir1 subdir2
test:
	@$(foreach mod,$(MODULE_DIRS),(cd $(mod) && go test ./...) &&) true
```

**Taskfile:**
```yaml
vars:
  MODULE_DIRS: ['.', 'subdir1', 'subdir2']

tasks:
  test:
    cmds:
      - for: { var: MODULE_DIRS }
        cmd: cd {{.ITEM}} && go test ./...
```

### 2. Conditional Commands
Use `if` for conditional execution:
```yaml
tasks:
  deploy:
    cmds:
      - cmd: echo "Deploying to production"
        if: '[ "$ENV" = "production" ]'
```

### 3. File Watching
```yaml
tasks:
  dev:
    desc: Watch and rebuild on changes
    watch: true
    sources:
      - '**/*.go'
    cmds:
      - go build -o ./bin/app
```

### 4. Cleanup with Defer
```yaml
tasks:
  test-with-db:
    cmds:
      - docker run -d --name test-db postgres:15
      - defer: docker rm -f test-db
      - go test -v ./...
```

## Known Issues and Solutions

### Issue 1: Echo with Brackets
**Problem:** Echo statements with brackets like `echo "[test] Running..."` cause parsing errors.

**Solution:** Remove brackets or use simple quotes:
```yaml
cmds:
  - echo "Running tests..."
```

### Issue 2: Go Cache Cleanup
**Problem:** `go clean -cache -testcache` may fail with "directory not empty" errors.

**Solution:** Add `|| true` to ignore errors:
```yaml
cmds:
  - go clean -cache -testcache || true
```

### Issue 3: Parallel golangci-lint
**Problem:** Multiple golangci-lint instances may conflict.

**Solution:** Task's dependency system handles this automatically with the `status` check.

## Packages to Migrate

Based on the monorepo structure, the following packages need migration:

1. ✅ jsonschema (completed - test case)
2. [ ] agentstack
3. [ ] aster
4. [ ] condeval
5. [ ] defuddle-go
6. [ ] emitter
7. [ ] filter
8. [ ] gendog
9. [ ] go-i18n
10. [ ] gozod
11. [ ] jsoncrdt
12. [ ] jsonpatch
13. [ ] knora
14. [ ] markconv
15. [ ] messageformat-go
16. [ ] openapi-request
17. [ ] pdfkit
18. [ ] polyparse
19. [ ] polytrans
20. [ ] requests
21. [ ] template
22. [ ] unifai
23. [ ] unifmsg
24. [ ] vfs
25. [ ] ... (and remaining packages)

## Timeline Estimate

- **Per package:** 15-30 minutes (depending on complexity)
- **Total packages:** ~30+
- **Estimated total time:** 8-15 hours
- **Recommended approach:** Migrate 3-5 packages per day

## Rollback Plan

If issues arise during migration:
1. Keep the original Makefile in git history
2. Can revert individual package migrations with `git revert`
3. Both Makefile and Taskfile can coexist temporarily if needed

## Success Criteria

A migration is considered successful when:
- ✅ All original Makefile targets are available as Task tasks
- ✅ `task --list` shows all tasks with descriptions
- ✅ All tests pass (`task test`)
- ✅ Linting works (`task lint`)
- ✅ Benchmarks run (`task bench`)
- ✅ Full verification passes (`task verify`)
- ✅ Makefile is removed
- ✅ Changes are committed to git

## References

- Task documentation: https://taskfile.dev/
- Golang Taskfile skill: `/Users/lincheng/work/skills/golang-skills/golang-taskfile`
- Test case: `jsonschema` package (commit: 4acacb9)

## Next Steps

1. Review this plan with the team
2. Begin migrating packages in priority order
3. Update CI/CD pipelines as packages are migrated
4. Document any package-specific customizations
5. Create a tracking issue to monitor progress
