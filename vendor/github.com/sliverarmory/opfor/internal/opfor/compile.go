package opfor

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/bytecode"
	"github.com/sliverarmory/opfor/internal/compiler"
	"github.com/sliverarmory/opfor/internal/lexer"
	"github.com/sliverarmory/opfor/internal/parser"
)

// CompileOption configures source compilation. It is a closed functional-option
// type; external packages should use the exported With* constructors.
type CompileOption func(*compileConfig)

type compileConfig struct {
	parser parser.Options
	err    error
}

// WithStrictSyntax requires documented comma and semicolon separators instead
// of accepting the reference runtime's legacy omissions.
func WithStrictSyntax() CompileOption {
	return func(config *compileConfig) {
		environments := config.parser.Environments
		config.parser = parser.StrictOptions()
		config.parser.Environments = environments
	}
}

// WithCompatibilityWarnings reports accepted omitted commas and semicolons as
// warnings while retaining compatibility-mode parsing.
func WithCompatibilityWarnings() CompileOption {
	return func(config *compileConfig) {
		config.parser.ReportCompatibilityWarnings = true
	}
}

// WithCompileEnvironment registers an importer-defined environment keyword
// for one standalone Compile call. Importers that own a Runtime will normally
// use WithEnvironment with Runtime.Compile instead.
func WithCompileEnvironment(keyword string, kind EnvironmentKind) CompileOption {
	return func(config *compileConfig) {
		normalized, err := normalizeEnvironment(keyword, kind)
		if err != nil {
			config.err = err
			return
		}
		if config.parser.Environments == nil {
			config.parser.Environments = make(map[string]ast.EnvironmentForm)
		}
		config.parser.Environments[normalized] = environmentASTForm(kind)
	}
}

// Program is immutable compiled Sleep/Aggressor code. A Program may be loaded
// by multiple independent runtimes.
type Program struct {
	source         Source
	tree           *ast.Script
	function       *bytecode.Function
	diagnostics    []Diagnostic
	numberLiterals map[*ast.NumberExpr]compiledNumberLiteral
	stringLiterals map[*ast.StringExpr]compiledStringLiteral
	// sourceAccount identifies the Runtime family which paid for this
	// Program's source bytes during compilation. A standalone Program leaves
	// it nil; loading that Program into a Runtime performs admission instead.
	// The field is set before publication and is never mutated afterwards.
	sourceAccount *runtimeResourceAccount
}

// Compile parses and compiles a named source unit.
func Compile(source Source, options ...CompileOption) (*Program, error) {
	config := compileConfig{parser: parser.CompatibilityOptions()}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	if config.err != nil {
		return nil, config.err
	}

	// Program must remain immutable even if an embedding caller reuses its
	// input buffer after Compile returns.
	source.Data = append([]byte(nil), source.Data...)
	parsed := parser.ParseWithOptions(lexer.Source(source), config.parser)
	diagnostics := append([]Diagnostic(nil), parsed.Diagnostics...)
	if hasErrorDiagnostic(diagnostics) {
		return nil, &CompileError{Diagnostics: diagnostics}
	}

	compiled := compiler.Compile(parsed.Script)
	diagnostics = append(diagnostics, compiled.Diagnostics...)
	diagnostics = append(diagnostics, validateStaticImports(source, parsed.Script)...)
	if hasErrorDiagnostic(diagnostics) {
		return nil, &CompileError{Diagnostics: diagnostics}
	}
	program := &Program{
		source: source, tree: parsed.Script, function: compiled.Function,
		diagnostics: append([]Diagnostic(nil), diagnostics...),
	}
	program.numberLiterals, program.stringLiterals = compileProgramLiterals(parsed.Script)
	return program, nil
}

type compiledNumberLiteral struct {
	value Value
	err   error
}

type compiledStringLiteral struct {
	decoded decodedSleepLiteral
	value   Value
	static  bool
	err     error
}

// compileProgramLiterals performs immutable parsing work once per Program.
// Errors remain attached to their node and are surfaced at the same runtime
// point as before, preserving Sleep's observable execution boundary.
func compileProgramLiterals(tree *ast.Script) (map[*ast.NumberExpr]compiledNumberLiteral, map[*ast.StringExpr]compiledStringLiteral) {
	numbers := make(map[*ast.NumberExpr]compiledNumberLiteral)
	stringsByNode := make(map[*ast.StringExpr]compiledStringLiteral)
	ast.Inspect(tree, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.NumberExpr:
			value, err := numberLiteral(node)
			numbers[node] = compiledNumberLiteral{value: value, err: err}
		case *ast.StringExpr:
			template := compiledStringLiteral{}
			if node.Kind == ast.SingleQuotedString {
				template.value = String(decodeSleepSingleQuoted(node.Text))
				template.static = true
				stringsByNode[node] = template
				break
			}
			template.decoded, template.err = decodeSleepEscapesAt(node.Text, node.TextRange)
			if template.err == nil && node.Kind == ast.DoubleQuotedString && !strings.Contains(template.decoded.text, "$") {
				template.value = template.decoded.valueRange(0, len(template.decoded.text))
				if strings.Contains(template.decoded.text, escapedDollarSentinel) {
					template.value = sleepStringReplaceAll(
						template.value,
						String(escapedDollarSentinel),
						String("$"),
					)
				}
				template.static = true
			}
			stringsByNode[node] = template
		}
		return true
	})
	return numbers, stringsByNode
}

// CompileString is a convenience wrapper around Compile.
func CompileString(name, code string, options ...CompileOption) (*Program, error) {
	return Compile(NewSource(name, []byte(code)), options...)
}

// Compile parses and compiles source using this Runtime's registered
// environment keywords.
func (r *Runtime) Compile(source Source, options ...CompileOption) (*Program, error) {
	if r == nil {
		return nil, fmt.Errorf("opfor: runtime is nil")
	}
	if _, err := r.reserveSourceLength(len(source.Data), 0); err != nil {
		return nil, err
	}
	return r.compileReservedSource(source, options...)
}

func (r *Runtime) reserveSourceLength(length int, alreadyReserved uint64) (uint64, error) {
	wanted := uint64(length)
	if wanted <= alreadyReserved {
		return alreadyReserved, nil
	}
	if err := r.reserveResource(resourceSourceBytes, wanted-alreadyReserved); err != nil {
		return alreadyReserved, err
	}
	return wanted, nil
}

// compileReservedSource compiles source whose bytes have already been charged
// to this Runtime family. Source-loading paths use it after reserving bytes
// before growing their input buffers.
func (r *Runtime) compileReservedSource(source Source, options ...CompileOption) (*Program, error) {
	if r == nil {
		return nil, fmt.Errorf("opfor: runtime is nil")
	}
	r.mu.RLock()
	environments := make(map[string]EnvironmentKind, len(r.environments))
	for keyword, kind := range r.environments {
		environments[keyword] = kind
	}
	r.mu.RUnlock()
	registered := func(config *compileConfig) {
		if config.parser.Environments == nil {
			config.parser.Environments = make(map[string]ast.EnvironmentForm, len(environments))
		}
		for keyword, kind := range environments {
			config.parser.Environments[keyword] = environmentASTForm(kind)
		}
	}
	combined := append([]CompileOption(nil), options...)
	combined = append(combined, registered)
	program, err := Compile(source, combined...)
	if err != nil {
		return nil, err
	}
	program.sourceAccount = r.resources
	return program, nil
}

// CompileString is the string-source counterpart to Runtime.Compile.
func (r *Runtime) CompileString(name, code string, options ...CompileOption) (*Program, error) {
	if r == nil {
		return nil, fmt.Errorf("opfor: runtime is nil")
	}
	// Reserve before converting the string to a byte slice so a rejected source
	// cannot trigger the compiler-owned allocation first.
	if err := r.reserveResource(resourceSourceBytes, uint64(len(code))); err != nil {
		return nil, err
	}
	return r.compileReservedSource(NewSource(name, []byte(code)), options...)
}

func normalizeEnvironment(keyword string, kind EnvironmentKind) (string, error) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return "", fmt.Errorf("opfor: environment keyword is empty")
	}
	if kind > EnvironmentPredicate {
		return "", fmt.Errorf("opfor: environment %q has invalid kind %d", keyword, kind)
	}
	for index, character := range keyword {
		if unicode.IsLetter(character) || character == '_' || (index > 0 && unicode.IsDigit(character)) {
			continue
		}
		return "", fmt.Errorf("opfor: invalid environment keyword %q", keyword)
	}
	return keyword, nil
}

func environmentASTForm(kind EnvironmentKind) ast.EnvironmentForm {
	switch kind {
	case EnvironmentFilter:
		return ast.FilterEnvironment
	case EnvironmentPredicate:
		return ast.PredicateEnvironment
	default:
		return ast.OrdinaryEnvironment
	}
}

// Source returns a copy of the program's original source.
func (p *Program) Source() Source {
	if p == nil {
		return Source{}
	}
	source := p.source
	source.Data = append([]byte(nil), source.Data...)
	return source
}

// Diagnostics returns non-fatal compatibility warnings produced while
// compiling the program. Fatal diagnostics are returned in CompileError and
// do not produce a Program.
func (p *Program) Diagnostics() []Diagnostic {
	if p == nil {
		return nil
	}
	return append([]Diagnostic(nil), p.diagnostics...)
}

func hasErrorDiagnostic(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}
