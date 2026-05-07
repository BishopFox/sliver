# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a high-performance Go implementation of JSON Pointer (RFC 6901), ported from the TypeScript [jsonjoy-com/json-pointer](https://github.com/jsonjoy-com/json-pointer) library.

**Primary Goals:**
1. **100% Functional Compatibility**: Maintain identical behavior with the TypeScript reference implementation
2. **TypeScript API Parity**: 1:1 mapping of all operations (parse, format, escape/unescape, get, find, validate)
3. **Performance Optimization**: Zero-allocation hot paths for common operations
4. **RFC 6901 Compliance**: Strict adherence to JSON Pointer specification semantics

**Reference Implementation**: https://github.com/jsonjoy-com/json-pointer

## Build Commands

```bash
# Run all tests with race detection
make test

# Run tests with coverage report (generates coverage.html)
make test-coverage

# Run tests with verbose output
make test-verbose

# Run benchmarks
make bench

# Run linters (golangci-lint + go mod tidy check)
make lint

# Format code
make fmt

# Run go vet
make vet

# Run full verification pipeline (deps, fmt, vet, lint, test)
make verify

# Clean build artifacts and caches
make clean

# Download and tidy dependencies
make deps
```

### Running Individual Tests

```bash
# Run specific test by name
go test -race -run TestFind

# Run tests in a specific file
go test -race -run TestStruct

# Run benchmarks for specific function
go test -bench=BenchmarkGet -benchmem
```

## Core Principles

### 1. TypeScript Compatibility (Critical)
- **Maintain 100% functional compatibility** with the TypeScript json-pointer implementation
- **Preserve exact error semantics**: Each Go error maps to a TypeScript error
- **Mirror API behavior**: All edge cases and special handling must match TypeScript
- **Port all test cases**: Reference TypeScript test suite for comprehensive coverage

### 2. Reference Implementation Alignment
- Follow algorithms from `reference/json-pointer/src/` implementation
- Each function should reference the corresponding TypeScript code in comments
- Maintain validation logic consistency with TypeScript version
- Consult TypeScript source when updating logic: https://github.com/jsonjoy-com/json-pointer

### 3. Modern Go Best Practices
- Implement **zero-allocation hot paths** for critical operations
- Use **progressive optimization**: correctness first, then performance
- Leverage **Go generics** for type safety (ArrayReference, ObjectReference)
- Follow **Go idioms** while maintaining TypeScript compatibility

### 4. Code Quality Standards
- **All comments in English** (no exceptions)
- **Include TypeScript references** in function comments
- **Document algorithm complexity** for performance-critical code
- **Consistent comment format** for all exported functions
- **Clean, readable code** over clever optimizations

### 5. Error Handling Philosophy
- Functions that **never fail in TypeScript** should not return errors in Go
- Functions that **throw in TypeScript** should return appropriate Go errors
- Use **simple error types** matching TypeScript error messages (see errors.go)
- Avoid complex error hierarchies - keep errors straightforward

## Core Architecture

### Design Philosophy

1. **TypeScript Compatibility**: Maintain identical behavior with the TypeScript reference implementation
2. **Zero-Allocation Hot Paths**: Critical operations like `Get` are optimized for zero allocations
3. **Layered Performance**: Progressive fallback from fast paths to reflection-based generic handling
4. **RFC 6901 Compliance**: Strict adherence to JSON Pointer specification semantics

### API Design Pattern: Two Entry Points

The library provides two ways to access JSON data, matching TypeScript API design:

1. **Variadic Path Components** - Direct function arguments (convenient, ergonomic):
   ```go
   value, err := jsonpointer.Get(doc, "users", "0", "name")
   ref, err := jsonpointer.Find(doc, "users", "0", "name")
   ```

2. **JSON Pointer Strings** - RFC 6901 format (standards-compliant):
   ```go
   value, err := jsonpointer.GetByPointer(doc, "/users/0/name")
   ref, err := jsonpointer.FindByPointer(doc, "/users/0/name")
   ```

### Public API (jsonpointer.go)

All public functions maintain 1:1 mapping with TypeScript API:

#### Navigation Functions
- **`Get(doc any, path ...string) (any, error)`** - Retrieve value using path components
- **`GetByPointer(doc any, pointer string) (any, error)`** - Retrieve value using JSON Pointer string
- **`Find(doc any, path ...string) (*Reference, error)`** - Find reference using path components
- **`FindByPointer(doc any, pointer string) (*Reference, error)`** - Find reference using JSON Pointer string

#### Parsing and Formatting
- **`Parse(pointer string) Path`** - Parse JSON Pointer string → Path array
- **`Format(path ...string) string`** - Format path components → JSON Pointer string
- **`Escape(component string) string`** - Escape special characters (~, /)
- **`Unescape(component string) string`** - Unescape encoded characters (~0, ~1)

#### Validation
- **`Validate(pointer any) error`** - Validate JSON Pointer string or Path
- **`ValidatePath(path any) error`** - Validate Path array structure

### Core Operations

- **Get**: Retrieve value at path (zero-allocation for common cases)
- **Find**: Retrieve value with parent context (`Reference` with `Val`, `Obj`, `Key`)
- **Parse**: JSON Pointer string → Path array (no escaping needed)
- **Format**: Path array → JSON Pointer string (automatic escaping)
- **Escape/Unescape**: Component encoding for `~` and `/` characters
- **Validate**: Validate JSON Pointer string or Path structure

### Performance Architecture: Layered Fallback Strategy

Both `Get` and `Find` use a **three-tier optimization strategy**:

#### Tier 1: Ultra-Fast Path (Zero Allocations)
- Direct type assertions for `map[string]any` and `[]any`
- Inline string-to-int conversion with `fastAtoi`
- No intermediate token objects created
- Handles 90%+ of real-world JSON data

#### Tier 2: Optimized Type Assertions
- Fast paths for common types: `[]string`, `[]int`, `[]float64`, `map[string]string`, etc.
- Pointer dereferencing for `*map[string]any`, `*[]any`
- Minimal allocations through on-demand token computation
- Handles specialized but common Go types

#### Tier 3: Reflection Fallback
- Generic handling via `reflect` package for arbitrary types
- Struct field access with JSON tag support
- Custom slice/array types
- Cached struct field mappings via `sync.Map`

This layered approach ensures **optimal performance for common cases** while maintaining **full compatibility for all Go types**.

### File Organization

#### Core Implementation Files
- **jsonpointer.go**: Public API surface - wrapper functions for all operations
- **types.go**: Core type definitions (`Path`, `Reference`, `ArrayReference`, `ObjectReference`, `internalToken`)
- **errors.go**: Sentinel errors matching TypeScript semantics exactly

#### Operation Files
- **get.go**: `Get` operation with zero-allocation optimization (fast paths for common types)
- **find.go**: `Find` operation with context tracking (returns `Reference` with parent)
- **findbypointer.go**: JSON Pointer string entry points (`GetByPointer`, `FindByPointer`)

#### Utility Files
- **util.go**: Parsing, formatting, and escape/unescape utilities
- **validate.go**: JSON Pointer and Path validation functions
- **struct.go**: Struct field access with reflection and caching (`sync.Map`)

#### Testing Files
- **\*_test.go**: Comprehensive unit tests mirroring TypeScript test suite
- **fuzz_test.go**: Fuzz testing for robustness against arbitrary inputs
- **benchmarks/**: Performance benchmarks and comparisons with other libraries

### Type System

#### Core Types (types.go)

**Path** - JSON Pointer path as array of string tokens:
```go
type Path []string
```

**Reference** - Generic reference with value, parent object, and key:
```go
type Reference struct {
    Val any    // The value at the pointer location
    Obj any    // The parent container (map, slice, struct)
    Key string // The key/index as string
}
```

**ArrayReference[T]** - Type-safe array element reference:
```go
type ArrayReference[T any] struct {
    Val *T  // Pointer for undefined | T semantics (nil = undefined)
    Obj []T // Parent array
    Key int // Numeric index
}
```

**ObjectReference[T]** - Type-safe object property reference:
```go
type ObjectReference[T any] struct {
    Val T            // The value at the key
    Obj map[string]T // Parent object
    Key string       // Property name
}
```

**internalToken** - Internal optimization structure (not exposed):
```go
type internalToken struct {
    key   string // Original key string
    index int    // Precomputed array index (-1 if invalid)
}
```

#### Helper Functions
- `IsArrayReference(ref Reference) bool` - Check if reference points to array element
- `IsArrayEnd[T](ref ArrayReference[T]) bool` - Check if reference is array end marker
- `IsObjectReference(ref Reference) bool` - Check if reference points to object property

### Error Types

All errors are defined in **errors.go** and map directly to TypeScript errors:

#### TypeScript-Compatible Errors
- **ErrInvalidIndex**: Invalid array index encountered (TypeScript: `'INVALID_INDEX'`)
- **ErrNotFound**: Path cannot be traversed (TypeScript: `'NOT_FOUND'`)
- **ErrNoParent**: Trying to get parent of root path (TypeScript: `'NO_PARENT'`)
- **ErrPointerInvalid**: JSON Pointer string is invalid (TypeScript: `'POINTER_INVALID'`)
- **ErrPointerTooLong**: JSON Pointer exceeds max length (TypeScript: `'POINTER_TOO_LONG'`)
- **ErrInvalidPath**: Path is not an array (TypeScript: `'Invalid path.'`)
- **ErrPathTooLong**: Path array exceeds max length (TypeScript: `'Path too long.'`)
- **ErrInvalidPathStep**: Path step is not string or number (TypeScript: `'Invalid path step.'`)

#### Go-Specific Errors
- **ErrIndexOutOfBounds**: Array index out of bounds (Go-specific for clarity)
- **ErrNilPointer**: Cannot traverse through nil pointer (Go-specific)
- **ErrFieldNotFound**: Struct field not found (Go-specific for struct support)
- **ErrKeyNotFound**: Map key not found (Go-specific for detailed error reporting)

## Key Implementation Patterns

### 1. Fast String-to-Integer Conversion

The `fastAtoi` function avoids `strconv.Atoi` allocations in hot paths:
- Inline digit validation
- Leading zero detection (RFC 6901 requirement)
- Overflow protection
- Returns -1 for invalid input (no error allocation)

### 2. Struct Field Caching

Struct field lookups use a global `sync.Map` cache:
- Key: `reflect.Type` of struct
- Value: `map[string]int` mapping field names to field indices
- Supports JSON tags: `json:"name"`, `json:"name,omitempty"`, `json:"-"`
- Unexported fields automatically skipped

### 3. On-Demand Token Computation

Unlike TypeScript's array-based approach, Go implementation computes tokens lazily:
```go
// Only created when fast path fails
token := internalToken{
    key:   step,
    index: fastAtoi(step),
}
```

This avoids allocating token slices for simple paths that resolve in the fast path.

### 4. RFC 6901 Array Semantics

Special handling for array end marker and bounds:
- `"-"` token refers to position *after* last element (non-existent)
- `index == len(array)` is valid for insertion context but returns error for access
- `index > len(array)` returns `ErrIndexOutOfBounds`
- Leading zeros rejected (except "0")

### 5. Error Semantics Matching TypeScript

**Critical**: Each error must map precisely to its TypeScript equivalent to maintain compatibility.

#### TypeScript Error Mappings
- `ErrNotFound` ← `throw new Error('NOT_FOUND')` - Path cannot be traversed
- `ErrInvalidIndex` ← `throw new Error('INVALID_INDEX')` - Invalid array index
- `ErrNoParent` ← `throw new Error('NO_PARENT')` - Root has no parent
- `ErrPointerInvalid` ← `throw new Error('POINTER_INVALID')` - Malformed JSON Pointer
- `ErrPointerTooLong` ← `throw new Error('POINTER_TOO_LONG')` - Exceeds max length
- `ErrInvalidPath` ← `throw new Error('Invalid path.')` - Path is not array
- `ErrPathTooLong` ← `throw new Error('Path too long.')` - Path exceeds max length
- `ErrInvalidPathStep` ← `throw new Error('Invalid path step.')` - Invalid path component

#### Go-Specific Errors
These errors provide more detailed information for Go use cases:
- `ErrIndexOutOfBounds` - Array index out of bounds (more specific than ErrNotFound)
- `ErrKeyNotFound` - Map key not found (Go map semantics)
- `ErrFieldNotFound` - Struct field not found (Go struct support)
- `ErrNilPointer` - Cannot traverse through nil pointer (Go pointer semantics)

## Testing Strategy

### Test Coverage Requirements
- All public functions must have comprehensive tests
- Edge cases: empty paths, root access, nil pointers, invalid indices
- Type coverage: maps, slices, structs, pointers, nested combinations
- Error paths: invalid pointers, missing keys, out-of-bounds access

### Benchmark Focus
- Compare against other Go JSON Pointer libraries
- Measure allocation counts (`-benchmem`)
- Profile hot paths to identify optimization opportunities
- Regression testing for performance improvements

### Fuzz Testing
See `fuzz_test.go` - validates robustness against arbitrary inputs.

## Development Process

### Implementation Priority

When working on new features or improvements, follow this order:

1. **Core Types and Errors**: Define or update type definitions and error constants first
2. **Utility Functions**: Implement parsing, formatting, validation helpers
3. **Get Operations**: Implement or optimize `Get` and `GetByPointer` functions
4. **Find Operations**: Implement or optimize `Find` and `FindByPointer` functions
5. **Validation**: Ensure comprehensive input validation
6. **Main API Integration**: Update public API wrappers in jsonpointer.go
7. **Performance Optimization**: Add fast paths and optimize hot code
8. **Comprehensive Testing**: Write tests mirroring TypeScript test cases

### Code Review Checklist

Before submitting changes, verify:

#### TypeScript Compatibility
- [ ] Behavior matches TypeScript reference implementation exactly
- [ ] All TypeScript test cases have been ported or referenced
- [ ] Error messages and error types match TypeScript errors
- [ ] Edge cases handled identically to TypeScript version
- [ ] Function comments reference TypeScript original code

#### Code Quality
- [ ] All comments are in English
- [ ] Algorithm complexity documented for performance-critical code
- [ ] Code is clean, readable, and maintainable
- [ ] No over-engineering - prefer simplicity
- [ ] Follows consistent comment format for exported functions

#### Performance
- [ ] No unnecessary allocations in hot paths
- [ ] Fast paths added for common type combinations
- [ ] Benchmarks run and no performance regressions
- [ ] Type assertions optimized before falling back to reflection

#### Testing
- [ ] All tests passing with `-race` flag
- [ ] New functionality has comprehensive test coverage
- [ ] Edge cases and error conditions tested
- [ ] Fuzz tests updated if relevant

### Common Development Tasks

#### Adding Support for a New Type

1. **Add fast path** in `get.go` `fastGet` function for the new type
2. **Add corresponding case** in `find.go` main switch statement
3. **Add optimized path** in `tryArrayAccess` or `tryObjectAccess` if applicable
4. **Add test cases** in appropriate test file with edge cases
5. **Run benchmarks** to verify performance impact: `make bench`
6. **Update documentation** if this is a new public-facing type

#### Optimizing Performance

1. **Profile first**: `go test -cpuprofile=cpu.prof -bench=.`
2. **Check allocations**: `go test -benchmem -bench=BenchmarkGet`
3. **Identify hot paths**: Use `go tool pprof cpu.prof` to find bottlenecks
4. **Focus optimization**: Most time spent in type assertions and `fastAtoi`
5. **Avoid reflection**: Add type-specific fast paths before falling back to `reflect`
6. **Benchmark comparison**: Ensure new code doesn't regress existing benchmarks
7. **Document tradeoffs**: Note any complexity added for performance gains

#### Maintaining TypeScript Compatibility

**Critical**: When updating any logic, always check the reference implementation first.

**Workflow**:
1. **Consult TypeScript source**: https://github.com/jsonjoy-com/json-pointer
2. **Read function comments**: Each function has TypeScript reference
3. **Review test cases**: Mirror TypeScript test suite structure
4. **Verify error messages**: Match TypeScript error strings exactly
5. **Test edge cases**: Ensure identical behavior for all edge cases
6. **Update references**: Keep TypeScript reference comments up to date

**Key TypeScript Files to Reference**:
- `src/find.ts` - Find operation implementation
- `src/get.ts` - Get operation implementation
- `src/util.ts` - Parsing, formatting, validation
- `__tests__/` - Comprehensive test cases

#### Adding New Error Types

Only add new errors if absolutely necessary:
1. **Check TypeScript first**: Does TypeScript have an equivalent error?
2. **Use existing errors**: Prefer existing errors if semantically correct
3. **Document mapping**: If adding Go-specific error, document why in errors.go
4. **Update CLAUDE.md**: Add error to the Error Types section
5. **Maintain simplicity**: Avoid complex error hierarchies

### Quality Assurance

#### Before Committing
```bash
# Run full verification pipeline
make verify

# This runs: deps, fmt, vet, lint, test
```

#### Performance Validation
```bash
# Run benchmarks and save results
go test -bench=. -benchmem ./... > bench-new.txt

# Compare with baseline (if you have bench-old.txt)
benchcmp bench-old.txt bench-new.txt
```

#### Test Coverage
```bash
# Generate coverage report
make test-coverage

# Open coverage.html in browser to review
```

### Implementation Guidelines

#### Simplicity Principles
- **Direct ports without over-engineering**: Port TypeScript logic directly when possible
- **Single implementation per operation**: Avoid multiple competing implementations
- **Avoid unnecessary complexity**: Don't add features not in TypeScript version
- **Focus on maintainability**: Code will be read more than written

#### Success Criteria
- All tests passing (including race detection)
- Clean, readable implementation
- Good performance characteristics
- 100% TypeScript compatibility maintained

## Comment Standards and Documentation

### Required Format for Exported Functions

Every exported function must follow this format:

```go
// FunctionName does something specific.
// Additional description if needed.
//
// TypeScript original code from <file>.ts:
//   <relevant TypeScript snippet or reference>
//
// Performance characteristics: O(n) time, O(1) space
func FunctionName(params) returnType {
    // implementation
}
```

### Comment Requirements
- **All comments in English** - No exceptions
- **Reference TypeScript source** - Include file and relevant snippet
- **Document complexity** - Note algorithm complexity for performance-critical code
- **Explain non-obvious logic** - Add inline comments for complex algorithms
- **Update on changes** - Keep TypeScript references current

### Example Comment
```go
// Parse parses a JSON Pointer string into a Path array.
// Handles unescaping of ~0 (for ~) and ~1 (for /) sequences.
//
// TypeScript original code from util.ts:
//   export const parseJsonPointer = (pointer: string): Path => { ... }
//
// Performance: O(n) where n is the length of the pointer string.
func Parse(pointer string) Path {
    return parseJSONPointer(pointer)
}
```

## Package Structure Best Practices

### Separation of Concerns
- **Public API** (jsonpointer.go) - Simple wrapper functions only
- **Internal Implementation** - Complex logic in internal functions
- **Type Definitions** (types.go) - All types in one place
- **Error Definitions** (errors.go) - All sentinel errors together

### Interface Design
- Prefer **simple interfaces** over complex abstractions
- Use **minimal external dependencies** (only testify for testing)
- Follow **Go idioms** while maintaining TypeScript compatibility
- Avoid **premature abstraction** - implement what's needed now

### Dependency Management
- Prefer **standard library** when possible (`reflect`, `strings`, `strconv`)
- Keep dependencies **minimal and well-maintained**
- No framework dependencies - pure Go implementation

## Go Version and Dependencies

- **Go Version**: 1.24.7 (see `go.mod`)
- **Go Features Used**: Generics (Go 1.18+), type parameters for ArrayReference/ObjectReference
- **Dependencies**: Minimal - only `github.com/stretchr/testify v1.11.1` for testing
- **Standard Library**: Heavy use of `reflect`, `strings`, `strconv`
- **Module Path**: `github.com/kaptinlin/jsonpointer`

## golangci-lint Configuration

- **Version**: 2.4.0 (specified in `.golangci.version`)
- **Timeout**: 10 minutes (for comprehensive linting)
- **Configuration**: Managed via Makefile with automatic version checking
- **Installation**: Automatic via `make install-golangci-lint`
- **Usage**: Run `make lint` for full linting suite (golangci-lint + mod tidy check)

## Additional Resources

### TypeScript Reference Implementation
- **Source Repository**: https://github.com/jsonjoy-com/json-pointer
- **Key Files**:
  - `src/find.ts` - Find operation
  - `src/get.ts` - Get operation
  - `src/util.ts` - Parsing, formatting, validation
  - `__tests__/` - Comprehensive test suite

### RFC 6901 Specification
- **RFC Document**: https://tools.ietf.org/html/rfc6901
- **Key Concepts**: JSON Pointer format, escaping rules, array indexing

### Performance Benchmarks
- See `benchmarks/README.md` for detailed performance comparisons
- Compare against other Go JSON Pointer libraries
- Track performance regressions with benchmark suite

## Summary

This library is a **high-fidelity port** of the TypeScript json-pointer implementation with Go-specific optimizations. When in doubt:

1. **Check TypeScript first** - Consult the reference implementation
2. **Maintain compatibility** - Preserve exact behavior and error semantics
3. **Optimize carefully** - Performance improvements must not break compatibility
4. **Test thoroughly** - Mirror TypeScript test cases and add Go-specific tests
5. **Document clearly** - Reference TypeScript source and explain Go-specific choices

The goal is **100% functional compatibility** with the TypeScript version while leveraging Go's performance characteristics and type safety.
