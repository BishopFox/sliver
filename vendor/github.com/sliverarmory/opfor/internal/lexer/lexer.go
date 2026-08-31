package lexer

import (
	"sort"
	"unicode"
	"unicode/utf8"

	"github.com/sliverarmory/opfor/internal/envspec"
)

const (
	diagnosticInvalidCharacter  = "LEX001"
	diagnosticUnterminatedQuote = "LEX002"
	diagnosticMalformedSigil    = "LEX003"
	diagnosticMalformedNumber   = "LEX004"
	diagnosticMalformedEscape   = "LEX005"
)

var syntaxKeywords = map[string]struct{}{
	"assert":    {},
	"break":     {},
	"callcc":    {},
	"catch":     {},
	"continue":  {},
	"done":      {},
	"else":      {},
	"false":     {},
	"for":       {},
	"foreach":   {},
	"from":      {},
	"halt":      {},
	"if":        {},
	"import":    {},
	"new":       {},
	"report":    {},
	"return":    {},
	"separator": {},
	"throw":     {},
	"true":      {},
	"try":       {},
	"use":       {},
	"while":     {},
	"yield":     {},
}

var keywords = func() map[string]struct{} {
	result := make(map[string]struct{}, len(syntaxKeywords)+len(envspec.Builtins()))
	for keyword := range syntaxKeywords {
		result[keyword] = struct{}{}
	}
	for _, spec := range envspec.Builtins() {
		if spec.LexicalKeyword {
			result[spec.Keyword] = struct{}{}
		}
	}
	return result
}()

var wordOperators = map[string]struct{}{
	"cmp":      {},
	"eq":       {},
	"ge":       {},
	"gt":       {},
	"hasmatch": {},
	"in":       {},
	"is":       {},
	"isa":      {},
	"isin":     {},
	"ismatch":  {},
	"iswm":     {},
	"le":       {},
	"lt":       {},
	"ne":       {},
	"not":      {},
	"notin":    {},
}

var symbolicOperators = []string{
	"!iswm", "!isa", "!is", "!in",
	"<<=", ">>=", "**=", "<=>", "===", "!==", "!=~",
	"=>", "==", "!=", "<=", ">=", "&&", "||", "++", "--",
	"+=", "-=", "*=", "/=", "%=", ".=", "&=", "|=", "^=",
	"<<", ">>", "**", "=~", "!~", "::", "->",
	"=", "+", "-", "*", "/", "%", ".", "<", ">", "!", "~", "|", "^", "?", "&",
}

func init() {
	// scanOperator relies on longest-match ordering. Keep that property even
	// when the table grows.
	sort.SliceStable(symbolicOperators, func(i, j int) bool {
		return len(symbolicOperators[i]) > len(symbolicOperators[j])
	})
}

// Result is the complete output of a lexical scan. Tokens always ends in EOF.
type Result struct {
	Tokens              []Token
	Diagnostics         []Diagnostic
	HasStructuralErrors bool
}

// Lexer tokenizes one Source.
type Lexer struct {
	source Source
	data   []byte

	offset int
	line   int
	column int

	tokens      []Token
	diagnostics []Diagnostic

	structuralDiagnostics []Diagnostic
}

// New creates a lexer for source.
func New(source Source) *Lexer {
	return &Lexer{
		source:                source,
		data:                  source.Data,
		line:                  1,
		column:                1,
		structuralDiagnostics: sleepStructuralDiagnostics(source),
	}
}

// Lex tokenizes source and returns all tokens and diagnostics.
func Lex(source Source) Result {
	return New(source).Lex()
}

// Lex tokenizes the lexer's source. A Lexer is single use.
func (l *Lexer) Lex() Result {
	for !l.atEnd() {
		leadingWhitespace := l.skipWhitespace()
		if l.atEnd() {
			l.emitEOF(leadingWhitespace)
			return l.result()
		}

		start := l.position()
		startOffset := l.offset
		r, _ := l.peekRune()

		var token Token
		switch {
		case r == '#':
			token = l.scanComment(start, startOffset)
		case r == '\'', r == '"', r == '`':
			token = l.scanString(start, startOffset, r)
		case r == '\\':
			token = l.scanReference(start, startOffset)
		case r == '$':
			token = l.scanSigil(start, startOffset, Scalar, '$')
		case r == '@':
			token = l.scanSigil(start, startOffset, Array, '@')
		case r == '%':
			if l.sigilFollows() || l.nextRuneIs('(') {
				token = l.scanSigil(start, startOffset, Hash, '%')
			} else {
				token = l.scanOperator(start, startOffset)
			}
		case r == '&':
			if l.sigilFollows() {
				token = l.scanSigil(start, startOffset, Function, '&')
			} else {
				token = l.scanOperator(start, startOffset)
			}
		case r == '^':
			if l.sigilFollows() {
				token = l.scanClass(start, startOffset)
			} else {
				token = l.scanOperator(start, startOffset)
			}
		case (r == '+' || r == '-') && l.signedSpecialDoubleFollows():
			token = l.scanSignedSpecialDouble(start, startOffset)
		case r == '-' && l.namedPredicateFollows(1):
			token = l.scanNamedPredicate(start, startOffset, false)
		case r == '!' && l.hasPrefix("!-") && l.namedPredicateFollows(2):
			token = l.scanNamedPredicate(start, startOffset, true)
		case unicode.IsDigit(r):
			token = l.scanNumber(start, startOffset)
		case isIdentifierStart(r):
			token = l.scanIdentifier(start, startOffset)
		default:
			token = l.scanPunctuationOrOperator(start, startOffset, r)
		}

		token.LeadingWhitespace = leadingWhitespace
		token.TrailingWhitespace = l.nextIsWhitespace()
		l.tokens = append(l.tokens, token)
	}

	l.emitEOF(false)
	return l.result()
}

func (l *Lexer) result() Result {
	if len(l.structuralDiagnostics) != 0 {
		// The reference lexer reports its fixed-order delimiter audit instead of
		// the later token scanner's inevitable unterminated-string cascade. Keep
		// producing best-effort tokens for lexer callers, while allowing the
		// parser to stop before those tokens create secondary diagnostics.
		return Result{
			Tokens:              l.tokens,
			Diagnostics:         append([]Diagnostic(nil), l.structuralDiagnostics...),
			HasStructuralErrors: true,
		}
	}
	return Result{Tokens: l.tokens, Diagnostics: l.diagnostics}
}

func (l *Lexer) emitEOF(leadingWhitespace bool) {
	position := l.position()
	l.tokens = append(l.tokens, Token{
		Kind:              EOF,
		Span:              l.span(position, position),
		LeadingWhitespace: leadingWhitespace,
	})
}

func (l *Lexer) scanComment(start Position, startOffset int) Token {
	l.advance() // #
	textOffset := l.offset
	for !l.atEnd() {
		r, _ := l.peekRune()
		// Sleep recognizes LF and CRLF as line endings, but a bare CR is
		// ordinary whitespace. Keep scanning a comment across a lone CR for
		// the same reason.
		if r == '\n' || r == '\r' && l.hasPrefix("\r\n") {
			break
		}
		l.advance()
	}
	return l.token(Comment, start, startOffset, textOffset, l.offset)
}

func (l *Lexer) scanString(start Position, startOffset int, quote rune) Token {
	l.advance()
	textOffset := l.offset
	textStart := l.position()
	escaped := false
	for !l.atEnd() {
		r, _ := l.peekRune()
		if !escaped && r == quote {
			textEnd := l.offset
			textEndPosition := l.position()
			l.advance()
			kind := SingleString
			switch quote {
			case '"':
				kind = DoubleString
			case '`':
				kind = BacktickString
			}
			token := l.token(kind, start, startOffset, textOffset, textEnd)
			token.TextSpan = l.span(textStart, textEndPosition)
			if quote != '\'' {
				l.validateParsedLiteralEscapes(token)
			}
			return token
		}

		l.advance()
		if r == '\\' {
			escaped = !escaped
		} else {
			escaped = false
		}
	}

	token := l.token(Invalid, start, startOffset, textOffset, l.offset)
	l.addDiagnostic(diagnosticUnterminatedQuote, "unterminated quoted string", token.Span)
	return token
}

func (l *Lexer) validateParsedLiteralEscapes(token Token) {
	text := token.Text
	for index := 0; index < len(text); index++ {
		if text[index] != '\\' || index+1 >= len(text) {
			continue
		}
		kind := text[index+1]
		index++
		width := 0
		description := ""
		switch kind {
		case 'u':
			width = 4
			description = "\\uXXXX"
		case 'x':
			width = 2
			description = "\\xXX"
		default:
			continue
		}
		if index+width >= len(text) {
			l.addDiagnostic(diagnosticMalformedEscape, "not enough remaining characters for "+description, token.Span)
			continue
		}
		digits := text[index+1 : index+1+width]
		valid := true
		for _, digit := range digits {
			if !isHexDigit(digit) {
				valid = false
				break
			}
		}
		if !valid {
			l.addDiagnostic(diagnosticMalformedEscape, "invalid unicode escape \\"+string(kind)+digits+" - must be hex digits", token.Span)
		}
		index += width
	}
}

func (l *Lexer) scanReference(start Position, startOffset int) Token {
	l.advance() // backslash
	if l.atEnd() {
		return l.invalid(start, startOffset, diagnosticMalformedSigil, "backslash must precede a variable or function reference")
	}

	r, _ := l.peekRune()
	if r != '$' && r != '@' && r != '%' && r != '&' {
		l.advance()
		return l.invalid(start, startOffset, diagnosticMalformedSigil, "backslash must precede a variable or function reference")
	}

	referenceStart := l.offset
	l.advance()
	if r == '$' && !l.atEnd() {
		next, _ := l.peekRune()
		if next == '+' {
			l.advance()
			return l.token(Reference, start, startOffset, referenceStart, l.offset)
		}
	}
	if !l.scanVariableName(r == '&') {
		return l.invalid(start, startOffset, diagnosticMalformedSigil, "reference sigil must be followed by a name")
	}
	return l.token(Reference, start, startOffset, referenceStart, l.offset)
}

func (l *Lexer) scanSigil(start Position, startOffset int, kind Kind, sigil rune) Token {
	l.advance()
	textOffset := l.offset

	// @() and %() are collection constructors. Their empty Text distinguishes
	// them from named variables without inventing parser-only token kinds.
	if (sigil == '@' || sigil == '%') && !l.atEnd() {
		r, _ := l.peekRune()
		if r == '(' {
			return l.token(kind, start, startOffset, textOffset, textOffset)
		}
	}

	if sigil == '$' && !l.atEnd() {
		r, _ := l.peekRune()
		if r == '+' {
			l.advance()
			return l.token(kind, start, startOffset, textOffset, l.offset)
		}
	}

	if !l.scanVariableName(kind == Function) {
		return l.invalid(start, startOffset, diagnosticMalformedSigil, "variable sigil must be followed by a name")
	}
	return l.token(kind, start, startOffset, textOffset, l.offset)
}

func (l *Lexer) scanVariableName(allowConnectors bool) bool {
	if l.atEnd() {
		return false
	}
	r, _ := l.peekRune()
	if !(isIdentifierStart(r) || unicode.IsDigit(r)) {
		return false
	}
	l.advance()
	for !l.atEnd() {
		r, _ = l.peekRune()
		if isIdentifierContinue(r) {
			l.advance()
			continue
		}
		if allowConnectors && (r == '-' || r == '+') && l.connectorContinuesIdentifier() {
			l.advance()
			continue
		}
		break
	}
	return true
}

func (l *Lexer) scanClass(start Position, startOffset int) Token {
	l.advance() // ^
	textOffset := l.offset
	for !l.atEnd() {
		r, _ := l.peekRune()
		if isIdentifierContinue(r) || r == '$' {
			l.advance()
			continue
		}
		// A dot belongs to a qualified Java class name only when another
		// identifier segment follows it. Sleep also permits its concatenation
		// operator to touch either operand, including ^Class."text" and
		// ^Class.$scalar, so consuming the trailing dot here changes the parse.
		if r == '.' && l.connectorContinuesIdentifier() {
			l.advance()
			continue
		}
		if r == '[' && l.nextRuneIs(']') {
			l.advance()
			l.advance()
			continue
		}
		break
	}
	return l.token(Class, start, startOffset, textOffset, l.offset)
}

func (l *Lexer) scanNumber(start Position, startOffset int) Token {
	kind := Integer
	malformed := false

	if l.hasPrefix("0x") || l.hasPrefix("0X") {
		l.advance()
		l.advance()
		digitCount := 0
		lastDigit := rune(0)
		for !l.atEnd() {
			r, _ := l.peekRune()
			if !isHexTermDigit(r) {
				break
			}
			digitCount++
			lastDigit = r
			l.advance()
		}
		if digitCount == 0 {
			malformed = true
		}
		if digitCount != 0 && !l.atEnd() {
			r, _ := l.peekRune()
			// LexicalAnalyzer keeps a dot inside a term only when the
			// immediately preceding UTF-16 char is a decimal digit. Thus
			// 0x9.Ap1 is one double term, while 0x1F."x" is integer
			// concatenation and 0xA.8p1 remains an unknown expression.
			if r == 'p' || r == 'P' || r == '.' && IsJavaDecimalDigit(lastDigit) {
				return l.scanHexadecimalDouble(start, startOffset)
			}
		}
		if !l.atEnd() {
			r, _ := l.peekRune()
			// Checkers.isNumber recognizes only the uppercase suffix before
			// delegating to Long.decode.
			if r == 'L' {
				kind = Long
				l.advance()
			}
		}
	} else {
		digitsBeforeDot := 0
		for !l.atEnd() {
			r, _ := l.peekRune()
			if !unicode.IsDigit(r) {
				break
			}
			digitsBeforeDot++
			l.advance()
		}

		if !l.atEnd() {
			r, _ := l.peekRune()
			if r == '.' {
				kind = Double
				l.advance()
				for !l.atEnd() {
					r, _ = l.peekRune()
					if !unicode.IsDigit(r) {
						break
					}
					l.advance()
				}
			}
		}

		if !l.atEnd() {
			r, _ := l.peekRune()
			if r == 'e' || r == 'E' {
				kind = Double
				l.advance()
				if !l.atEnd() {
					r, _ = l.peekRune()
					if r == '+' || r == '-' {
						l.advance()
					}
				}
				exponentDigits := 0
				for !l.atEnd() {
					r, _ = l.peekRune()
					if !unicode.IsDigit(r) {
						break
					}
					exponentDigits++
					l.advance()
				}
				if exponentDigits == 0 {
					malformed = true
				}
			}
		}

		if !l.atEnd() {
			r, _ := l.peekRune()
			switch r {
			case 'L':
				if kind == Double {
					malformed = true
				}
				kind = Long
				l.advance()
			// Checkers.isDouble delegates to Double.parseDouble, which accepts
			// either Java floating-point suffix on decimal forms.
			case 'D', 'd', 'F', 'f':
				kind = Double
				l.advance()
			}
		}

		if digitsBeforeDot == 0 && kind != Double {
			malformed = true
		}
	}

	token := l.token(kind, start, startOffset, startOffset, l.offset)
	if malformed {
		token.Kind = Invalid
		l.addDiagnostic(diagnosticMalformedNumber, "malformed numeric literal", token.Span)
	}
	return token
}

func (l *Lexer) scanIdentifier(start Position, startOffset int) Token {
	l.advance()
	for !l.atEnd() {
		r, _ := l.peekRune()
		if isIdentifierContinue(r) {
			l.advance()
			continue
		}
		if (r == '-' || r == '+') && l.connectorContinuesIdentifier() {
			l.advance()
			continue
		}
		break
	}

	kind := Identifier
	lexeme := string(l.data[startOffset:l.offset])
	if _, ok := keywords[lexeme]; ok {
		kind = Keyword
	} else if _, ok := wordOperators[lexeme]; ok {
		kind = Operator
	}
	return l.token(kind, start, startOffset, startOffset, l.offset)
}

// scanHexadecimalDouble continues after scanNumber has consumed a 0x/0X
// prefix and at least one hexadecimal digit. Sleep passes the complete term
// to Double.parseDouble: a binary p/P exponent is mandatory, fraction digits
// are optional after a dot, and D/d/F/f suffixes are accepted. Invalid terms
// remain ordinary unknown expressions rather than lexer diagnostics.
func (l *Lexer) scanHexadecimalDouble(start Position, startOffset int) Token {
	if !l.atEnd() {
		r, _ := l.peekRune()
		if r == '.' {
			l.advance()
			for !l.atEnd() {
				r, _ = l.peekRune()
				if !isHexTermDigit(r) {
					break
				}
				l.advance()
			}
		}
	}

	valid := false
	if !l.atEnd() {
		r, _ := l.peekRune()
		if r == 'p' || r == 'P' {
			l.advance()
			if !l.atEnd() {
				r, _ = l.peekRune()
				if r == '+' || r == '-' {
					l.advance()
				}
			}
			exponentDigits := 0
			for !l.atEnd() {
				r, _ = l.peekRune()
				if r < '0' || r > '9' {
					break
				}
				exponentDigits++
				l.advance()
			}
			valid = exponentDigits != 0
		}
	}

	if valid && !l.atEnd() {
		r, _ := l.peekRune()
		if r == 'D' || r == 'd' || r == 'F' || r == 'f' {
			l.advance()
		}
	}
	kind := Identifier
	if valid {
		kind = Double
	}
	return l.token(kind, start, startOffset, startOffset, l.offset)
}

func (l *Lexer) signedSpecialDoubleFollows() bool {
	for _, literal := range []string{"+NaN", "-NaN", "+Infinity", "-Infinity"} {
		if !l.hasPrefix(literal) {
			continue
		}
		after := l.offset + len(literal)
		if after >= len(l.data) {
			return true
		}
		r, _ := utf8.DecodeRune(l.data[after:])
		return !isIdentifierContinue(r)
	}
	return false
}

func (l *Lexer) scanSignedSpecialDouble(start Position, startOffset int) Token {
	l.advance() // sign
	for !l.atEnd() {
		r, _ := l.peekRune()
		if !isIdentifierContinue(r) {
			break
		}
		l.advance()
	}
	return l.token(Double, start, startOffset, startOffset, l.offset)
}

func (l *Lexer) scanNamedPredicate(start Position, startOffset int, negated bool) Token {
	if negated {
		l.advance() // !
	}
	l.advance() // -
	l.advance() // first name rune, checked by namedPredicateFollows
	for !l.atEnd() {
		r, _ := l.peekRune()
		if isIdentifierContinue(r) {
			l.advance()
			continue
		}
		if (r == '-' || r == '+') && l.connectorContinuesIdentifier() {
			l.advance()
			continue
		}
		break
	}
	return l.token(Operator, start, startOffset, startOffset, l.offset)
}

func (l *Lexer) scanPunctuationOrOperator(start Position, startOffset int, r rune) Token {
	if r == ':' && l.hasPrefix("::") {
		return l.scanOperator(start, startOffset)
	}

	kind := Invalid
	switch r {
	case ';':
		kind = Semicolon
	case ',':
		kind = Comma
	case ':':
		kind = Colon
	case '{':
		kind = LeftBrace
	case '}':
		kind = RightBrace
	case '(':
		kind = LeftParen
	case ')':
		kind = RightParen
	case '[':
		kind = LeftBracket
	case ']':
		kind = RightBracket
	default:
		if l.operatorFollows() {
			return l.scanOperator(start, startOffset)
		}
	}

	l.advance()
	if kind == Invalid {
		return l.invalid(start, startOffset, diagnosticInvalidCharacter, "invalid character in source")
	}
	return l.token(kind, start, startOffset, startOffset, l.offset)
}

func (l *Lexer) scanOperator(start Position, startOffset int) Token {
	for _, operator := range symbolicOperators {
		if !l.hasPrefix(operator) {
			continue
		}
		if operator[0] == '!' && len(operator) > 1 && isASCIIAlpha(operator[1]) {
			after := l.offset + len(operator)
			if after < len(l.data) {
				r, _ := utf8.DecodeRune(l.data[after:])
				if isIdentifierContinue(r) {
					continue
				}
			}
		}
		for range operator {
			l.advance()
		}
		return l.token(Operator, start, startOffset, startOffset, l.offset)
	}

	l.advance()
	return l.invalid(start, startOffset, diagnosticInvalidCharacter, "invalid operator")
}

func (l *Lexer) token(kind Kind, start Position, startOffset, textStart, textEnd int) Token {
	return Token{
		Kind:   kind,
		Lexeme: string(l.data[startOffset:l.offset]),
		Text:   string(l.data[textStart:textEnd]),
		Span:   l.span(start, l.position()),
	}
}

func (l *Lexer) invalid(start Position, startOffset int, code, message string) Token {
	token := l.token(Invalid, start, startOffset, startOffset, l.offset)
	l.addDiagnostic(code, message, token.Span)
	return token
}

func (l *Lexer) addDiagnostic(code, message string, span Span) {
	l.diagnostics = append(l.diagnostics, Diagnostic{
		Severity: SeverityError,
		Code:     code,
		Message:  message,
		Span:     span,
	})
}

func (l *Lexer) skipWhitespace() bool {
	skipped := false
	for !l.atEnd() {
		r, _ := l.peekRune()
		if !unicode.IsSpace(r) {
			break
		}
		skipped = true
		l.advance()
	}
	return skipped
}

func (l *Lexer) nextIsWhitespace() bool {
	if l.atEnd() {
		return false
	}
	r, _ := l.peekRune()
	return unicode.IsSpace(r)
}

func (l *Lexer) operatorFollows() bool {
	for _, operator := range symbolicOperators {
		if l.hasPrefix(operator) {
			return true
		}
	}
	return false
}

func (l *Lexer) sigilFollows() bool {
	if l.atEnd() {
		return false
	}
	_, size := l.peekRune()
	if l.offset+size >= len(l.data) {
		return false
	}
	r, _ := utf8.DecodeRune(l.data[l.offset+size:])
	return isIdentifierStart(r) || unicode.IsDigit(r)
}

func (l *Lexer) connectorContinuesIdentifier() bool {
	_, size := l.peekRune()
	after := l.offset + size
	if after >= len(l.data) {
		return false
	}
	r, _ := utf8.DecodeRune(l.data[after:])
	return isIdentifierStart(r)
}

func (l *Lexer) namedPredicateFollows(prefixBytes int) bool {
	after := l.offset + prefixBytes
	if after >= len(l.data) {
		return false
	}
	r, _ := utf8.DecodeRune(l.data[after:])
	return isIdentifierStart(r)
}

func (l *Lexer) nextRuneIs(expected rune) bool {
	if l.atEnd() {
		return false
	}
	_, size := l.peekRune()
	if l.offset+size >= len(l.data) {
		return false
	}
	r, _ := utf8.DecodeRune(l.data[l.offset+size:])
	return r == expected
}

func (l *Lexer) nextRuneIsDigit() bool {
	if l.atEnd() {
		return false
	}
	_, size := l.peekRune()
	if l.offset+size >= len(l.data) {
		return false
	}
	r, _ := utf8.DecodeRune(l.data[l.offset+size:])
	return unicode.IsDigit(r)
}

func (l *Lexer) hasPrefix(prefix string) bool {
	if len(l.data)-l.offset < len(prefix) {
		return false
	}
	for i := range prefix {
		if l.data[l.offset+i] != prefix[i] {
			return false
		}
	}
	return true
}

func (l *Lexer) atEnd() bool {
	return l.offset >= len(l.data)
}

func (l *Lexer) peekRune() (rune, int) {
	return utf8.DecodeRune(l.data[l.offset:])
}

func (l *Lexer) advance() rune {
	r, size := l.peekRune()
	if r == '\r' {
		l.offset += size
		if l.offset < len(l.data) && l.data[l.offset] == '\n' {
			l.offset++
			l.line++
			l.column = 1
			return '\n'
		}
		// The reference parser deliberately does not treat a lone carriage
		// return as a newline. It still separates tokens as whitespace, but
		// diagnostics for CR-only dynamic source remain on the same line.
		l.column++
		return r
	}

	l.offset += size
	if r == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return r
}

func (l *Lexer) position() Position {
	return Position{Offset: l.offset, Line: l.line, Column: l.column}
}

func (l *Lexer) span(start, end Position) Span {
	return Span{Source: sourceName(l.source), Start: start, End: end}
}

func isIdentifierStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentifierContinue(r rune) bool {
	return isIdentifierStart(r) || unicode.IsDigit(r)
}

func isHexDigit(r rune) bool {
	return JavaDigit(r, 16) >= 0
}

func isHexTermDigit(r rune) bool {
	// A supplementary digit is not a Java hexadecimal digit because Sleep's
	// decoder sees its two UTF-16 surrogates. Keep it in the lexical term so
	// classification reports Unknown expression instead of splitting 0x off
	// as a malformed prefix.
	return isHexDigit(r) || unicode.IsDigit(r)
}

func isASCIIAlpha(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}
