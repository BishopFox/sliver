# AGENTS.md

This file provides AI agents (Claude, Cursor, etc.) with coding guidelines when working on this Go project.

## 🎯 Core Principles

### KISS (Keep It Simple, Stupid)
- Write simple, straightforward code that's easy to understand
- Avoid over-engineering and unnecessary abstractions
- Prefer clear logic over clever tricks
- Choose readability over brevity when they conflict

### DRY (Don't Repeat Yourself)
- Extract common logic into reusable functions
- Use generics for type-safe code reuse (Go 1.18+)
- Create utility packages for shared functionality
- Avoid copy-paste programming

### YAGNI (You Aren't Gonna Need It)
- Implement features only when actually needed
- Don't build "just in case" functionality
- Resist premature optimization
- Add complexity only when justified by real requirements

### Single Responsibility Principle
- Each function should do one thing well
- Keep functions short and focused (ideally < 50 lines)
- Split complex logic into smaller, testable units
- Separate concerns clearly

## 🚀 Golang 2025 Best Practices

### Modern Go Features (Go 1.23+)

#### Generics (Go 1.18+)
```go
// Good: Use generics for type-safe collections
func Map[T, U any](slice []T, fn func(T) U) []U {
    result := make([]U, len(slice))
    for i, v := range slice {
        result[i] = fn(v)
    }
    return result
}

// Good: Type constraints for domain logic
func Min[T constraints.Ordered](a, b T) T {
    if a < b {
        return a
    }
    return b
}
```

#### Enhanced Error Handling
```go
// Good: Wrap errors with context
if err := validateSchema(data); err != nil {
    return fmt.Errorf("schema validation failed: %w", err)
}

// Good: Use errors.Is and errors.As
if errors.Is(err, ErrInvalidSchema) {
    // handle specific error
}

var validationErr *ValidationError
if errors.As(err, &validationErr) {
    // access validation error details
}
```

#### Structured Logging (slog - Go 1.21+)
```go
import "log/slog"

// Good: Use structured logging
slog.Info("schema compiled",
    slog.String("schema_id", id),
    slog.Int("properties", len(props)),
    slog.Duration("duration", elapsed))

slog.Error("validation failed",
    slog.Any("error", err),
    slog.String("path", fieldPath))
```

#### Clear Package Functions (Go 1.22+)
```go
// Good: Use clear for better performance
clear(myMap)  // Better than making new map
clear(mySlice)  // Clear slice elements
```

### Code Style & Formatting

#### Naming Conventions
```go
// Good: Short, clear names in limited scope
for i, v := range items {
    processItem(v)
}

// Good: Descriptive names for exported APIs
type SchemaValidator interface {
    ValidateJSON(data []byte) (*ValidationResult, error)
    ValidateStruct(v any) (*ValidationResult, error)
}

// Avoid: Single letter exports or unclear abbreviations
// Bad: type Svc struct {}
// Good: type Service struct {}
```

#### Import Organization
```go
import (
    // Standard library
    "context"
    "errors"
    "fmt"

    // Third-party packages
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    // Local packages
    "github.com/kaptinlin/jsonschema/internal/validator"
)
```

#### Comment Style
```go
// Good: Document exported APIs with examples
// ValidateJSON validates JSON data against the compiled schema.
// It returns a ValidationResult containing any validation errors.
//
// Example:
//
//     result, err := schema.ValidateJSON([]byte(`{"name": "John"}`))
//     if err != nil {
//         return err
//     }
//     if !result.IsValid() {
//         // handle validation errors
//     }
func (s *Schema) ValidateJSON(data []byte) (*ValidationResult, error) {
    // Implementation
}
```

### Performance Optimization

#### Memory Management
```go
// Good: Pre-allocate with known capacity
result := make([]string, 0, len(input))

// Good: Use sync.Pool for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

func processData(data []byte) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    // use buffer
}

// Good: Use strings.Builder for concatenation
var b strings.Builder
for _, s := range strs {
    b.WriteString(s)
}
return b.String()
```

#### Avoid Common Performance Pitfalls
```go
// Bad: String concatenation in loops
result := ""
for _, s := range items {
    result += s  // Creates new string each iteration
}

// Good: Use strings.Builder
var b strings.Builder
for _, s := range items {
    b.WriteString(s)
}
result := b.String()

// Bad: Unnecessary allocations
data := []byte(string(byteSlice))

// Good: Direct conversion
data := byteSlice
```

### Error Handling Patterns

#### Domain-Specific Errors
```go
// Good: Custom error types with context
type ValidationError struct {
    Path    string
    Message string
    Code    string
    Value   any
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s (at %s)", e.Code, e.Message, e.Path)
}

// Good: Sentinel errors for common cases
var (
    ErrInvalidSchema = errors.New("invalid schema")
    ErrInvalidJSON   = errors.New("invalid JSON")
)
```

#### Context Propagation
```go
// Good: Pass context as first parameter
func (c *Compiler) CompileWithContext(ctx context.Context, schema []byte) (*Schema, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    // compilation logic
}
```

## 🧪 Testing with testify

### Test Structure
```go
import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
)

// Good: Table-driven tests
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    bool
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   `{"name": "John"}`,
            want:    true,
            wantErr: false,
        },
        {
            name:    "invalid input",
            input:   `{"age": "invalid"}`,
            want:    false,
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := validate(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, result.IsValid())
        })
    }
}
```

### Assertions vs Requirements
```go
// Good: Use require for critical checks that should stop test
func TestCritical(t *testing.T) {
    result, err := compile(schema)
    require.NoError(t, err)  // Stop if compilation fails
    require.NotNil(t, result)

    // Continue with non-critical assertions
    assert.Equal(t, expectedType, result.Type)
    assert.Len(t, result.Properties, 5)
}

// Good: Use assert for non-critical checks
func TestProperties(t *testing.T) {
    schema := loadSchema()
    assert.Equal(t, "object", schema.Type)
    assert.Greater(t, len(schema.Properties), 0)
    assert.Contains(t, schema.Required, "name")
}
```

### Test Suites
```go
// Good: Use suites for related tests with setup/teardown
type ValidatorSuite struct {
    suite.Suite
    compiler *Compiler
    schema   *Schema
}

func (s *ValidatorSuite) SetupTest() {
    s.compiler = NewCompiler()
    var err error
    s.schema, err = s.compiler.Compile(testSchema)
    s.Require().NoError(err)
}

func (s *ValidatorSuite) TestValidInput() {
    result, err := s.schema.Validate(validInput)
    s.Require().NoError(err)
    s.True(result.IsValid())
}

func TestValidatorSuite(t *testing.T) {
    suite.Run(t, new(ValidatorSuite))
}
```

### Benchmarks
```go
// Good: Write benchmarks for critical paths
func BenchmarkValidation(b *testing.B) {
    schema := compileSchema(b)
    data := loadTestData(b)

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, err := schema.Validate(data)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Good: Sub-benchmarks for comparison
func BenchmarkValidationTypes(b *testing.B) {
    schema := compileSchema(b)

    b.Run("ValidateJSON", func(b *testing.B) {
        data := []byte(`{"name": "John"}`)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            schema.ValidateJSON(data)
        }
    })

    b.Run("ValidateStruct", func(b *testing.B) {
        data := testStruct{Name: "John"}
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            schema.ValidateStruct(data)
        }
    })
}
```

### Test Helpers
```go
// Good: Create helpers to reduce boilerplate
func requireValidSchema(t *testing.T, schema string) *Schema {
    t.Helper()
    compiler := NewCompiler()
    s, err := compiler.Compile([]byte(schema))
    require.NoError(t, err)
    require.NotNil(t, s)
    return s
}

func assertValidationError(t *testing.T, result *ValidationResult, expectedCode string) {
    t.Helper()
    assert.False(t, result.IsValid())
    assert.Contains(t, result.Errors, expectedCode)
}
```

## 📚 API Design

### Public Interface Guidelines
```go
// Good: Clear, consistent method names
type Validator interface {
    Validate(data any) (*Result, error)
    ValidateJSON(data []byte) (*Result, error)
    ValidateStruct(v any) (*Result, error)
}

// Good: Options pattern for configuration
type Option func(*Compiler)

func WithCache(size int) Option {
    return func(c *Compiler) {
        c.cache = newCache(size)
    }
}

func WithLogger(logger *slog.Logger) Option {
    return func(c *Compiler) {
        c.logger = logger
    }
}

func NewCompiler(opts ...Option) *Compiler {
    c := &Compiler{
        // defaults
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}
```

### Backward Compatibility
```go
// Good: Deprecate gracefully
// Deprecated: Use ValidateJSON instead.
// This function will be removed in v2.0.0.
func Validate(data []byte) error {
    _, err := ValidateJSON(data)
    return err
}
```

## 🔍 Code Quality

### Static Analysis Tools
```bash
# Required tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run before commit
goimports -w .
go vet ./...
golangci-lint run
go test ./... -race
```

### golangci-lint Configuration
```yaml
# .golangci.yml
linters:
  enable:
    - errcheck      # Check error handling
    - gosimple      # Simplify code
    - govet         # Go vet
    - ineffassign   # Detect ineffectual assignments
    - staticcheck   # Static analysis
    - unused        # Check for unused code
    - gofmt         # Check formatting
    - goimports     # Check imports
    - misspell      # Check spelling
    - revive        # Fast linter
    - gocritic      # Opinionated linter
```

### Pre-commit Hooks
```bash
#!/bin/bash
# .git/hooks/pre-commit

# Format code
goimports -w .

# Run tests
go test ./... -race -short

# Run linters
golangci-lint run

# Check go.mod
go mod tidy
git diff --exit-code go.mod go.sum
```

## 🚫 Anti-Patterns

### Avoid Over-Engineering
```go
// Bad: Unnecessary abstraction
type StringProcessor interface {
    Process(string) string
}
type UpperCaseProcessor struct{}
func (u UpperCaseProcessor) Process(s string) string {
    return strings.ToUpper(s)
}

// Good: Keep it simple
func toUpperCase(s string) string {
    return strings.ToUpper(s)
}
```

### Avoid Premature Optimization
```go
// Bad: Optimizing before measuring
func processItems(items []Item) {
    // Complex optimization without proof it's needed
    pool := sync.Pool{...}
    // ...
}

// Good: Start simple, optimize if needed
func processItems(items []Item) []Result {
    results := make([]Result, len(items))
    for i, item := range items {
        results[i] = processItem(item)
    }
    return results
}
// Then benchmark and optimize if needed
```

### Avoid Mutable Global State
```go
// Bad: Global mutable state
var defaultCompiler *Compiler

func Validate(data []byte) error {
    return defaultCompiler.Validate(data)
}

// Good: Explicit dependencies
type Service struct {
    compiler *Compiler
}

func (s *Service) Validate(data []byte) error {
    return s.compiler.Validate(data)
}
```

## 🌍 Internationalization

### Error Messages
```go
// Good: Use error codes with i18n support
type ValidationError struct {
    Code   string
    Params map[string]any
}

func (e *ValidationError) Localize(locale string) string {
    return i18n.Translate(locale, e.Code, e.Params)
}

// Usage
err := &ValidationError{
    Code: "required_field_missing",
    Params: map[string]any{
        "field": "email",
    },
}
```

## ⚡ Validation Library Specific

### Input Type Handling
```go
// Good: Support multiple input types consistently
func (s *Schema) Validate(data any) (*Result, error) {
    switch v := data.(type) {
    case []byte:
        return s.ValidateJSON(v)
    case string:
        return s.ValidateJSON([]byte(v))
    case map[string]any:
        return s.ValidateMap(v)
    default:
        return s.ValidateStruct(v)
    }
}
```

### Schema Compilation & Caching
```go
// Good: Cache compiled schemas
type Compiler struct {
    mu    sync.RWMutex
    cache map[string]*Schema
}

func (c *Compiler) Compile(schema []byte) (*Schema, error) {
    id := computeSchemaID(schema)

    c.mu.RLock()
    if cached, ok := c.cache[id]; ok {
        c.mu.RUnlock()
        return cached, nil
    }
    c.mu.RUnlock()

    compiled := c.compile(schema)

    c.mu.Lock()
    c.cache[id] = compiled
    c.mu.Unlock()

    return compiled, nil
}
```

### Performance Hot Paths
```go
// Good: Fast path for common types
func validateType(value any, expectedType string) bool {
    switch expectedType {
    case "string":
        _, ok := value.(string)
        return ok
    case "number":
        switch value.(type) {
        case int, int64, float64:
            return true
        }
        return false
    default:
        return validateTypeReflection(value, expectedType)
    }
}
```

---

> **Focus**: Write simple, correct, maintainable code. Test thoroughly with testify. Optimize only when measured performance requires it. Follow KISS, DRY, and YAGNI principles religiously.
