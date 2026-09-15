// Package parser parses Sleep and Aggressor Script tokens into an AST.
package parser

import (
	"fmt"
	"strings"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/lexer"
)

const (
	diagnosticExpectedToken        = "PAR001"
	diagnosticExpectedExpression   = "PAR002"
	diagnosticExpectedTerminator   = "PAR003"
	diagnosticInvalidAssignment    = "PAR004"
	diagnosticUnexpectedToken      = "PAR005"
	diagnosticMissingComma         = "PAR006"
	diagnosticInvalidEnvironment   = "PAR007"
	diagnosticInvalidObject        = "PAR008"
	diagnosticInvalidImport        = "PAR009"
	diagnosticInvalidControlFlow   = "PAR010"
	diagnosticOperatorWhitespace   = "PAR011"
	diagnosticInvalidParsedLiteral = "PAR012"
	diagnosticTooManyDiagnostics   = "PAR999"
	defaultMaximumDiagnostics      = 100
)

// Options controls compatibility extensions accepted by the parser.
type Options struct {
	// AllowOmittedSemicolons accepts a statement boundary at a line break,
	// closing brace, or end of input. The reference Sleep parser accepts this
	// in several contexts even though documented syntax uses semicolons.
	AllowOmittedSemicolons bool

	// AllowOmittedCommas accepts adjacent values in argument, tuple, and loop
	// lists. The reference parser is similarly permissive for legacy scripts.
	AllowOmittedCommas bool

	// ReportCompatibilityWarnings emits warnings for accepted missing
	// separators. Leave this false when loading legacy .cna files quietly.
	ReportCompatibilityWarnings bool

	// MaximumDiagnostics limits cascading errors. Values <= 0 use 100.
	MaximumDiagnostics int

	// Environments declares importer-defined parser keywords and their Sleep
	// environment bridge forms. A predicate environment must be known before
	// parsing so `keyword (condition) { ... }` is not interpreted as a function
	// call followed by an unconditional block.
	Environments map[string]ast.EnvironmentForm
}

// CompatibilityOptions returns the defaults used by Parse. They follow the
// permissive behavior of the reference Sleep parser.
func CompatibilityOptions() Options {
	return Options{
		AllowOmittedSemicolons: true,
		AllowOmittedCommas:     true,
		MaximumDiagnostics:     defaultMaximumDiagnostics,
	}
}

// StrictOptions requires documented comma and semicolon separators.
func StrictOptions() Options {
	return Options{MaximumDiagnostics: defaultMaximumDiagnostics}
}

// Result contains a best-effort tree and every lexer/parser diagnostic.
type Result struct {
	Script      *ast.Script
	Diagnostics []lexer.Diagnostic
}

// HasErrors reports whether parsing or lexing produced an error diagnostic.
func (r Result) HasErrors() bool {
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == lexer.SeverityError {
			return true
		}
	}
	return false
}

// Parse lexes and parses source using compatibility options.
func Parse(source lexer.Source) Result {
	return ParseWithOptions(source, CompatibilityOptions())
}

// ParseWithOptions lexes and parses source with options.
func ParseWithOptions(source lexer.Source, options Options) Result {
	lexed := lexer.Lex(source)
	if lexed.HasStructuralErrors {
		return Result{Diagnostics: append([]lexer.Diagnostic(nil), lexed.Diagnostics...)}
	}
	parsed := parseTokensWithOptions(lexed.Tokens, options, source.Data)
	diagnostics := make([]lexer.Diagnostic, 0, len(lexed.Diagnostics)+len(parsed.Diagnostics))
	diagnostics = append(diagnostics, lexed.Diagnostics...)
	diagnostics = append(diagnostics, parsed.Diagnostics...)
	parsed.Diagnostics = diagnostics
	return parsed
}

// ParseTokens parses an already-tokenized source using compatibility options.
func ParseTokens(tokens []lexer.Token) Result {
	return ParseTokensWithOptions(tokens, CompatibilityOptions())
}

// ParseTokensWithOptions parses an already-tokenized source with options.
func ParseTokensWithOptions(tokens []lexer.Token, options Options) Result {
	return parseTokensWithOptions(tokens, options, nil)
}

func parseTokensWithOptions(tokens []lexer.Token, options Options, source []byte) Result {
	if options.MaximumDiagnostics <= 0 {
		options.MaximumDiagnostics = defaultMaximumDiagnostics
	}

	// Comments never participate in the grammar. Their removal does not lose
	// statement-boundary information because token spans retain line numbers.
	filtered := make([]lexer.Token, 0, len(tokens)+1)
	for _, token := range tokens {
		if token.Kind != lexer.Comment {
			filtered = append(filtered, token)
		}
	}
	if len(filtered) == 0 || filtered[len(filtered)-1].Kind != lexer.EOF {
		var span lexer.Span
		if len(filtered) != 0 {
			span = filtered[len(filtered)-1].Span
			span.Start = span.End
		}
		filtered = append(filtered, lexer.Token{Kind: lexer.EOF, Span: span})
	}

	p := &parser{tokens: filtered, options: options, source: source}
	script := p.parseScript(false)
	return Result{Script: script, Diagnostics: p.diagnostics}
}

type parser struct {
	tokens      []lexer.Token
	source      []byte
	position    int
	options     Options
	diagnostics []lexer.Diagnostic
	aborted     bool
}

func (p *parser) parseScript(stopAtBrace bool) *ast.Script {
	start := p.current().Span
	statements := make([]ast.Stmt, 0)
	for !p.atEnd() {
		if stopAtBrace && p.checkKind(lexer.RightBrace) {
			break
		}
		if p.matchKind(lexer.Semicolon) {
			continue
		}

		before := p.position
		statement := p.parseStatement()
		if statement != nil {
			statements = append(statements, statement)
		}
		if p.position == before {
			p.errorAt(p.current(), diagnosticUnexpectedToken, "unexpected token %q", p.current().Lexeme)
			p.advance()
			p.synchronize()
		}
		if p.aborted {
			break
		}
	}

	end := p.current().Span
	if len(statements) != 0 {
		start = statements[0].Span()
		end = statements[len(statements)-1].Span()
	}
	return &ast.Script{
		Base:       ast.Base{Range: joinSpans(start, end)},
		Statements: statements,
	}
}

func (p *parser) current() lexer.Token {
	if p.position >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.position]
}

func (p *parser) peek(distance int) lexer.Token {
	index := p.position + distance
	if index < 0 {
		index = 0
	}
	if index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[index]
}

func (p *parser) previous() lexer.Token {
	return p.peek(-1)
}

func (p *parser) atEnd() bool {
	return p.current().Kind == lexer.EOF
}

func (p *parser) advance() lexer.Token {
	token := p.current()
	if !p.atEnd() {
		p.position++
	}
	return token
}

func (p *parser) checkKind(kind lexer.Kind) bool {
	return p.current().Kind == kind
}

func (p *parser) matchKind(kinds ...lexer.Kind) bool {
	for _, kind := range kinds {
		if p.checkKind(kind) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *parser) checkLexeme(lexemes ...string) bool {
	for _, lexeme := range lexemes {
		if p.current().Lexeme == lexeme {
			return true
		}
	}
	return false
}

func (p *parser) matchLexeme(lexemes ...string) bool {
	if p.checkLexeme(lexemes...) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) checkWord(words ...string) bool {
	word := strings.ToLower(p.current().Text)
	if word == "" {
		word = strings.ToLower(p.current().Lexeme)
	}
	for _, candidate := range words {
		if word == candidate {
			return true
		}
	}
	return false
}

func (p *parser) matchWord(words ...string) bool {
	if p.checkWord(words...) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) expectKind(kind lexer.Kind, description string) (lexer.Token, bool) {
	if p.checkKind(kind) {
		return p.advance(), true
	}
	p.errorAt(p.current(), diagnosticExpectedToken, "expected %s, found %s", description, tokenDescription(p.current()))
	return p.current(), false
}

func (p *parser) errorAt(token lexer.Token, code, format string, args ...any) {
	p.addDiagnostic(lexer.SeverityError, code, token.Span, format, args...)
}

func (p *parser) warnAt(token lexer.Token, code, format string, args ...any) {
	p.addDiagnostic(lexer.SeverityWarning, code, token.Span, format, args...)
}

func (p *parser) addDiagnostic(severity lexer.Severity, code string, span lexer.Span, format string, args ...any) {
	if p.aborted {
		return
	}
	if len(p.diagnostics) >= p.options.MaximumDiagnostics {
		p.diagnostics = append(p.diagnostics, lexer.Diagnostic{
			Severity: lexer.SeverityError,
			Code:     diagnosticTooManyDiagnostics,
			Message:  "too many parser diagnostics; parsing stopped",
			Span:     span,
		})
		p.aborted = true
		return
	}
	p.diagnostics = append(p.diagnostics, lexer.Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
		Span:     span,
	})
}

func (p *parser) synchronize() {
	for !p.atEnd() && !p.checkKind(lexer.RightBrace) {
		if p.matchKind(lexer.Semicolon) {
			return
		}
		if isStatementKeyword(p.current()) {
			return
		}
		p.advance()
	}
}

func tokenDescription(token lexer.Token) string {
	if token.Kind == lexer.EOF {
		return "end of input"
	}
	if token.Lexeme == "" {
		return token.Kind.String()
	}
	return fmt.Sprintf("%s %q", token.Kind, token.Lexeme)
}

func joinSpans(first, last lexer.Span) lexer.Span {
	if first.Source == "" {
		return last
	}
	if last.Source == "" {
		return first
	}
	return lexer.Span{Source: first.Source, Start: first.Start, End: last.End}
}

func isStatementKeyword(token lexer.Token) bool {
	word := strings.ToLower(token.Text)
	if word == "" {
		word = strings.ToLower(token.Lexeme)
	}
	switch word {
	case "assert", "break", "callcc", "continue", "done", "for", "foreach", "halt", "if", "import", "return", "throw", "try", "while", "yield":
		return true
	default:
		return false
	}
}
