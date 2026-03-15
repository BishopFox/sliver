# go-i18n

High-performance internationalization library for Go with ICU MessageFormat support.

## Commands

```bash
# Testing
task test                # Run all tests
task test-verbose        # Run tests with verbose output
task test-coverage       # Generate coverage report (coverage.html)
go test -race ./...      # Run tests with race detection

# Linting
task lint                # Run golangci-lint and tidy checks
make golangci-lint       # Run golangci-lint only
make tidy-lint           # Verify go.mod/go.sum are tidy

# Benchmarking
make bench               # Run benchmarks

# Verification
task verify              # Run deps, fmt, vet, lint, test
make all                 # Run lint and test
```

## Architecture

### Core Components

**I18n (Bundle)** - Main internationalization manager
- Manages translations across multiple locales
- Configures fallback chains and language matching
- Pre-compiles MessageFormat templates on load
- Stores parsed translations in `map[locale]map[name]*parsedTranslation`
- Provides `NewLocalizer()` to create locale-specific translators

**Localizer** - Per-locale translation interface
- `Get(name, vars)` - Token-based translation with MessageFormat variables
- `GetX(name, context, vars)` - Context-disambiguated translation (e.g., "Post <verb>" vs "Post <noun>")
- `Getf(name, args)` - sprintf-style formatting
- `Format(message, vars)` - Direct MessageFormat compilation and formatting (bypasses translation lookup)

**parsedTranslation** - Pre-compiled translation unit
- Holds locale, name, text, and compiled MessageFormat function
- Compiled once during load, reused for all lookups
- Graceful fallback: if compilation fails, stores raw text

### Translation System

**Dual Translation Model:**
- **Token-based**: Keys like `hello_world`, `button_create` (traditional i18n)
- **Text-based**: Full sentences like `"Hello, world!"` act as fallback when translation missing

**Context Disambiguation:**
- Append ` <context>` suffix to distinguish homonyms
- Example: `"Post <verb>"` vs `"Post <noun>"`
- Use `GetX("Post", "verb")` to lookup `"Post <verb>"`

**MessageFormat Integration:**
- Uses `kaptinlin/messageformat-go/v1` for ICU MessageFormat v1
- Supports pluralization, variables, custom formatters
- Pre-compiles templates on load for performance
- Graceful degradation on compilation errors (returns raw text)

### File Loading

**LoadMessages(map[locale]map[name]text)** - Load from Go maps
**LoadFiles(...paths)** - Load specific files
**LoadGlob(...patterns)** - Load files matching glob patterns
**LoadFS(fsys, ...patterns)** - Load from `fs.FS` (supports `go:embed`)

**File Processing:**
1. Read files via `os.ReadFile` or `fs.ReadFile`
2. Unmarshal using configured unmarshaler (default: JSON)
3. Extract locale from filename via `nameInsensitive()` (e.g., `zh_CN.json` → `zh-cn`)
4. Merge translations into locale-keyed map
5. Parse and compile MessageFormat templates
6. Build fallback chains via `formatFallbacks()`

### Language Features

**Locale Normalization:**
- Converts `zh_CN`, `zh-Hans`, `ZH_CN` to standard form `zh-Hans`
- Uses `golang.org/x/text/language` for parsing and matching

**Fallback Chains:**
- Configured via `WithFallback(map[locale][]fallbacks)`
- Recursive fallback support (e.g., `zh-Hans → zh → zh-Hant → default`)
- Prevents infinite loops with visited tracking
- Final fallback: default locale

**Accept-Language Parsing:**
- `MatchAvailableLocale(acceptLang)` parses HTTP Accept-Language header
- Returns best matching locale from configured locales
- Uses `language.Matcher` for confidence-based matching

## Key Types and Interfaces

```go
// I18n is the main bundle managing translations and locales
type I18n struct {
    defaultLocale             string
    defaultLanguage           language.Tag
    languages                 []language.Tag
    unmarshaler               Unmarshaler
    languageMatcher           language.Matcher
    fallbacks                 map[string][]string
    parsedTranslations        map[string]map[string]*parsedTranslation
    runtimeParsedTranslations map[string]*parsedTranslation
    mfOptions                 *mf.MessageFormatOptions
}

// Localizer provides translation methods for a specific locale
type Localizer struct {
    bundle *I18n
    locale string
}

// Vars holds MessageFormat variables for interpolation
type Vars map[string]any

// Unmarshaler unmarshals translation files (JSON, YAML, TOML, INI)
type Unmarshaler func(data []byte, v any) error
```

## Configuration Options

Use functional options pattern with `NewBundle()`:

```go
// Core configuration
WithDefaultLocale(locale string)           // Set default locale
WithLocales(locales ...string)             // Set supported locales
WithFallback(map[string][]string)          // Configure fallback chains
WithUnmarshaler(Unmarshaler)               // Set custom unmarshaler (YAML, TOML, INI)

// MessageFormat configuration
WithMessageFormatOptions(*mf.MessageFormatOptions)  // Full MessageFormat options
WithCustomFormatters(map[string]any)                // Add custom formatters
WithStrictMode(bool)                                // Enable strict parsing
```

## Coding Rules

### Error Handling
- **No panics in production code** - All errors return gracefully
- MessageFormat compilation errors: return raw text as fallback
- File loading errors: return wrapped errors with context
- Missing translations: return translation key as fallback

### Performance Patterns
- **Pre-allocation**: Use `make(map[T]V, capacity)` and `slices.Grow()` when size is known
- **Batch operations**: Use `maps.Copy()` instead of element-by-element assignment
- **String processing**: Use `strings.Cut()` over `strings.Split()` for single-delimiter splits
- **Deduplication**: Use `slices.Sort()` + `slices.Compact()` for efficient deduplication
- **Template caching**: Compile MessageFormat templates once during load, reuse for all lookups

### Go 1.26 Features Used
- `slices.Grow()`, `slices.Sort()`, `slices.Compact()`, `slices.Index()`, `slices.Insert()`, `slices.Delete()`
- `maps.Copy()` for bulk map copying
- `strings.Cut()` for efficient string splitting
- `min()` for capacity estimation
- `clear()` for map/slice clearing

### Code Style
- Functional options pattern for configuration
- Graceful degradation over strict failures
- Pre-compile and cache expensive operations
- Use `golang.org/x/text/language` for all locale operations
- Document exported types and functions with usage examples

## Testing

### Test Structure
- Table-driven tests with `t.Run()` subtests
- Use `testify/assert` and `testify/require` for assertions
- Parallel tests where possible with `t.Parallel()`
- Benchmark critical paths (MessageFormat compilation, translation lookup)

### Test Coverage
- Core translation functionality (Get, GetX, Getf, Format)
- File loading (LoadFiles, LoadGlob, LoadFS, LoadMessages)
- Fallback chain resolution
- Locale normalization and matching
- MessageFormat compilation and formatting
- Custom unmarshalers (YAML, TOML, INI)

### Running Tests
```bash
task test              # Run all tests
task test-verbose      # Verbose output
task test-coverage     # Generate coverage report
go test -race ./...    # Race detection
make bench             # Run benchmarks
```

## Dependencies

**Core:**
- `github.com/go-json-experiment/json` - Default JSON unmarshaler (experimental v2)
- `github.com/kaptinlin/messageformat-go` - ICU MessageFormat v1 implementation
- `golang.org/x/text` - Language matching and locale parsing

**Optional (for custom unmarshalers):**
- `gopkg.in/yaml.v3` - YAML support
- `github.com/pelletier/go-toml/v2` - TOML support
- `gopkg.in/ini.v1` - INI support

**Testing:**
- `github.com/stretchr/testify` - Assertions and test utilities

## Performance

### Optimizations Applied

**Go 1.26 Modernization:**
- Built-in functions (`min`, `max`, `clear`) for efficient operations
- `slices` package for pre-allocation, sorting, deduplication
- `maps` package for bulk copying
- `strings.Cut()` for reduced allocations
- Smart capacity estimation for slices and maps

**MessageFormat Engine:**
- Upgraded to `kaptinlin/messageformat-go/v1` (10-50x faster than previous engine)
- Pre-compilation and caching of templates
- Graceful fallback on compilation errors

**Key Improvements:**
- `nameInsensitive()`: 40-60% faster with reduced allocations
- File loading: 25-35% faster with batch operations
- Translation lookup: Optimized with pre-allocated data structures
- Duplicate removal: Efficient using `slices.Compact()`

### Benchmarking
```bash
make bench  # Run all benchmarks
```

Focus areas: MessageFormat compilation, translation lookup, file loading, locale normalization.


## Agent Skills

This package indexes agent skills from its own .agents/skills directory (go-i18n/.agents/skills/):

| Skill | When to Use |
|-------|-------------|
| [agent-md-creating](.agents/skills/agent-md-creating/) | Create or update CLAUDE.md and AGENTS.md instructions for this Go package. |
| [code-simplifying](.agents/skills/code-simplifying/) | Refine recently changed Go code for clarity and consistency without behavior changes. |
| [committing](.agents/skills/committing/) | Prepare conventional commit messages for this Go package. |
| [dependency-selecting](.agents/skills/dependency-selecting/) | Evaluate and choose Go dependencies with alternatives and risk tradeoffs. |
| [go-best-practices](.agents/skills/go-best-practices/) | Apply Google Go style and architecture best practices to code changes. |
| [linting](.agents/skills/linting/) | Configure or run golangci-lint and fix lint issues in this package. |
| [modernizing](.agents/skills/modernizing/) | Adopt newer Go language and toolchain features safely. |
| [ralphy-initializing](.agents/skills/ralphy-initializing/) | Initialize or repair the .ralphy workflow configuration. |
| [ralphy-todo-creating](.agents/skills/ralphy-todo-creating/) | Generate or refine TODO tracking via the Ralphy workflow. |
| [readme-creating](.agents/skills/readme-creating/) | Create or rewrite README.md for this package. |
| [releasing](.agents/skills/releasing/) | Prepare release and semantic version workflows for this package. |
| [testing](.agents/skills/testing/) | Design or update tests (table-driven, fuzz, benchmark, and edge-case coverage). |
