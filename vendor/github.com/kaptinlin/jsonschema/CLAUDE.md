# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a high-performance JSON Schema validator for Go that implements the JSON Schema Draft 2020-12 specification. The library provides direct struct validation, smart unmarshaling with defaults, and a separated validation workflow.

## Key Architecture Components

### Core Components
- **Compiler** (`compiler.go`): Central component for schema compilation and management. Handles schema caching, references, custom formats, and default functions.
- **Schema** (`schema.go`): Represents compiled JSON Schema instances with all validation properties and metadata.
- **Validator methods**: Type-specific validation methods (`ValidateJSON`, `ValidateStruct`, `ValidateMap`, `Validate`)
- **Unmarshaler** (`unmarshal.go`): Handles data unmarshaling with default value application

### Code Generation Tool
- **schemagen** (`cmd/schemagen/`): Command-line tool for generating Schema methods from Go structs with jsonschema tags
- **Generator** (`cmd/schemagen/generator.go`): Core generation logic
- **Analyzer** (`cmd/schemagen/analyzer.go`): Struct analysis and tag parsing

### Validation Keywords
Individual files implement JSON Schema validation keywords (e.g., `properties.go`, `required.go`, `type.go`, `format.go`, etc.)

### Supporting Components
- **Struct Tags** (`struct_tags.go`): Parse jsonschema struct tags for validation rules
- **Internationalization** (`i18n.go`, `locales/`): Multi-language error messages  
- **Custom Formats** (`format.go`, `formats.go`): Built-in and custom format validators
- **Default Functions** (`default_funcs.go`): Dynamic default value generation

## Build and Development Commands

### Basic Commands
- `make test` - Run all tests with race detection
- `make lint` - Run all linters (golangci-lint + go mod tidy check)  
- `make bench` - Run benchmarks
- `make verify` - Run complete verification (deps + fmt + vet + lint + test)

### Test Commands
- `make test-unit` - Run unit tests only
- `make test-coverage` - Generate coverage report (creates coverage.html)
- `make test-verbose` - Run tests with verbose output

### Code Quality
- `make fmt` - Format Go code
- `make vet` - Run go vet
- `make clean` - Clean build artifacts and caches
- `make deps` - Download and tidy Go module dependencies

### Code Generation
- `go install github.com/kaptinlin/jsonschema/cmd/schemagen@latest` - Install schemagen tool
- `schemagen` - Generate schema methods for structs in current package
- `schemagen [packages...]` - Generate for specific packages

## Testing Structure

- **Unit Tests**: Each validation keyword has corresponding test files (e.g., `required_test.go`)
- **Integration Tests**: Located in `tests/` directory with comprehensive test cases
- **Official Test Suite**: Uses JSON Schema Test Suite in `testdata/JSON-Schema-Test-Suite/`
- **Benchmarks**: Performance tests using `go test -bench=.`

## Key Patterns

### Schema Compilation Pattern
1. Create Compiler instance with `NewCompiler()`
2. Register custom formats/functions if needed
3. Compile schema with `compiler.Compile(schemaJSON)`
4. Use compiled schema for validation/unmarshaling

### Validation Workflow  
1. Validate data first using appropriate method (`ValidateJSON`, `ValidateStruct`, etc.)
2. Check `result.IsValid()`
3. If valid, unmarshal with `schema.Unmarshal()` to apply defaults

### Error Handling
- All validation errors implement detailed error information with field paths
- Errors support internationalization through locale files
- Use `result.Errors` map or `result.ToList()` for error processing

## Configuration Files

- `go.mod` - Go module definition (requires Go 1.25)
- `Makefile` - Build automation and development commands
- `.golangci.version` - Required golangci-lint version for consistent linting
- `locales/*.json` - Translation files for error messages

## Dependencies

- `github.com/go-json-experiment/json` - Experimental JSON library for Go
- `github.com/kaptinlin/go-i18n` - Internationalization support
- `github.com/stretchr/testify` - Testing utilities
- `github.com/goccy/go-yaml` - YAML support

## Common Development Tasks

Run tests before committing:
```bash
make verify
```

Generate schemas from struct tags:
```bash
schemagen
```

Run specific test suites:
```bash
go test ./tests/ -v
go test ./cmd/schemagen/ -v
```