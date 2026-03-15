// lexer.go - ICU MessageFormat lexer implementation
// TypeScript original code:
// import moo, { Rules } from 'moo';
//
// export const states: { [state: string]: Rules } = {
//   body: {
//     doubleapos: { match: "''", value: () => "'" },
//     quoted: {
//       lineBreaks: true,
//       match: /'[{}#](?:[^']|'')*'(?!')/u,
//       value: src => src.slice(1, -1).replace(/''/g, "'")
//     },
//     argument: {
//       lineBreaks: true,
//       match: /\{\s*[^\p{Pat_Syn}\p{Pat_WS}]+\s*/u,
//       push: 'arg',
//       value: src => src.substring(1).trim()
//     },
//     octothorpe: '#',
//     end: { match: '}', pop: 1 },
//     content: { lineBreaks: true, match: /[^][^{}#']*/u }
//   },
//   arg: {
//     select: {
//       lineBreaks: true,
//       match: /,\s*(?:plural|select|selectordinal)\s*,\s*/u,
//       next: 'select',
//       value: src => src.split(',')[1].trim()
//     },
//     'func-args': {
//       lineBreaks: true,
//       match: /,\s*[^\p{Pat_Syn}\p{Pat_WS}]+\s*,/u,
//       next: 'body',
//       value: src => src.split(',')[1].trim()
//     },
//     'func-simple': {
//       lineBreaks: true,
//       match: /,\s*[^\p{Pat_Syn}\p{Pat_WS}]+\s*/u,
//       value: src => src.substring(1).trim()
//     },
//     end: { match: '}', pop: 1 }
//   },
//   select: {
//     offset: {
//       lineBreaks: true,
//       match: /\s*offset\s*:\s*\d+\s*/u,
//       value: src => src.split(':')[1].trim()
//     },
//     case: {
//       lineBreaks: true,
//       match: /\s*(?:=\d+|[^\p{Pat_Syn}\p{Pat_WS}]+)\s*\{/u,
//       push: 'body',
//       value: src => src.substring(0, src.indexOf('{')).trim()
//     },
//     end: { match: /\s*\}/u, pop: 1 }
//   }
// };
//
// export const lexer = moo.states(states);

package v1

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// LexerToken represents a token from the lexer with position information
type LexerToken struct {
	Type       string // Token type
	Value      string // Token value (processed)
	Text       string // Original raw text
	Offset     int    // Start position in source
	Line       int    // Line number (1-based)
	Col        int    // Column number (1-based)
	LineBreaks int    // Number of line breaks consumed
}

// LexerState represents the current state of the lexer state machine
type LexerState string

const (
	LexerStateBody   LexerState = "body"
	LexerStateArg    LexerState = "arg"
	LexerStateSelect LexerState = "select"
)

// TokenType constants matching TypeScript lexer
const (
	TokenDoubleApos = "doubleapos"
	TokenQuoted     = "quoted"
	TokenArgument   = "argument"
	TokenOctothorpe = "octothorpe"
	TokenEnd        = "end"
	TokenContent    = "content"
	TokenSelect     = "select"
	TokenFuncArgs   = "func-args" // #nosec G101 -- This is a token type constant, not credentials
	TokenFuncSimple = "func-simple"
	TokenOffset     = "offset"
	TokenCase       = "case"
)

// Pattern definitions matching TypeScript regex patterns
var (
	// Body state patterns
	patternDoubleApos = regexp.MustCompile(`^''`)
	patternQuoted     = regexp.MustCompile(`^'[{}#](?:[^']|'')*'`)
	patternArgument   = regexp.MustCompile(`^\{\s*[^{},\s]+\s*`)
	patternContent    = regexp.MustCompile(`^[^{}#']*`)

	// Arg state patterns
	patternSelect     = regexp.MustCompile(`^,\s*(?:plural|select|selectordinal)\s*,\s*`)
	patternFuncArgs   = regexp.MustCompile(`^,\s*[^{},\s]+\s*,`)
	patternFuncSimple = regexp.MustCompile(`^,\s*[^{},\s]+\s*`)

	// Select state patterns
	patternOffset    = regexp.MustCompile(`^\s*offset\s*:\s*-?\d+\s*`)
	patternCase      = regexp.MustCompile(`^\s*(?:=\d+|[^{}\s]+)\s*\{`)
	patternSelectEnd = regexp.MustCompile(`^\s*\}`)
)

// Lexer implements a stateful lexer following TypeScript's moo.states pattern
type Lexer struct {
	source     string
	pos        int
	line       int
	col        int
	stateStack []LexerState
	tokens     []LexerToken
	tokenIndex int
}

// NewLexer creates a new lexer instance
func NewLexer(source string) *Lexer {
	return &Lexer{
		source:     source,
		pos:        0,
		line:       1,
		col:        1,
		stateStack: []LexerState{LexerStateBody},
		tokens:     nil,
		tokenIndex: 0,
	}
}

// Reset resets the lexer with new source
func (l *Lexer) Reset(source string) {
	l.source = source
	l.pos = 0
	l.line = 1
	l.col = 1
	l.stateStack = []LexerState{LexerStateBody}
	l.tokens = nil
	l.tokenIndex = 0
}

// getCurrentState returns the current lexer state
func (l *Lexer) getCurrentState() LexerState {
	if len(l.stateStack) == 0 {
		return LexerStateBody
	}
	return l.stateStack[len(l.stateStack)-1]
}

// pushState pushes a new state onto the stack
func (l *Lexer) pushState(state LexerState) {
	l.stateStack = append(l.stateStack, state)
}

// popState pops a state from the stack
func (l *Lexer) popState() {
	if len(l.stateStack) > 1 {
		l.stateStack = l.stateStack[:len(l.stateStack)-1]
	}
}

// nextState changes to a new state (replacing current)
func (l *Lexer) nextState(state LexerState) {
	if len(l.stateStack) > 0 {
		l.stateStack[len(l.stateStack)-1] = state
	} else {
		l.stateStack = []LexerState{state}
	}
}

// atEnd returns true if we're at the end of input
func (l *Lexer) atEnd() bool {
	return l.pos >= len(l.source)
}

// remaining returns the remaining source from current position
func (l *Lexer) remaining() string {
	if l.atEnd() {
		return ""
	}
	return l.source[l.pos:]
}

// createToken creates a token with position information
func (l *Lexer) createToken(tokenType, value, text string, lineBreaks int) LexerToken {
	token := LexerToken{
		Type:       tokenType,
		Value:      value,
		Text:       text,
		Offset:     l.pos,
		Line:       l.line,
		Col:        l.col,
		LineBreaks: lineBreaks,
	}
	return token
}

// advance advances the lexer position and updates line/col tracking
func (l *Lexer) advance(length int) {
	text := l.source[l.pos : l.pos+length]
	for _, ch := range text {
		if ch == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
	}
	l.pos += length
}

// countLineBreaks counts line breaks in text
func (l *Lexer) countLineBreaks(text string) int {
	count := 0
	for _, ch := range text {
		if ch == '\n' {
			count++
		}
	}
	return count
}

// matchBodyState matches patterns in body state
func (l *Lexer) matchBodyState() *LexerToken {
	remaining := l.remaining()
	if remaining == "" {
		return nil
	}

	// doubleapos: { match: "''", value: () => "'" }
	if match := patternDoubleApos.FindString(remaining); match != "" {
		token := l.createToken(TokenDoubleApos, "'", match, 0)
		l.advance(len(match))
		return &token
	}

	// quoted: {
	//   lineBreaks: true,
	//   match: /'[{}#](?:[^']|'')*'(?!')/u,
	//   value: src => src.slice(1, -1).replace(/''/g, "'")
	// }
	if match := patternQuoted.FindString(remaining); match != "" {
		// Implement negative lookahead (?!') manually
		endPos := len(match)
		if endPos < len(remaining) && remaining[endPos] == '\'' {
			// Skip this match because it's followed by another quote
		} else {
			// Process value: slice(1, -1).replace(/''/g, "'")
			value := match[1 : len(match)-1]             // Remove outer quotes
			value = strings.ReplaceAll(value, "''", "'") // Replace double quotes
			lineBreaks := l.countLineBreaks(match)
			token := l.createToken(TokenQuoted, value, match, lineBreaks)
			l.advance(len(match))
			return &token
		}
	}

	// argument: {
	//   lineBreaks: true,
	//   match: /\{\s*[^\p{Pat_Syn}\p{Pat_WS}]+\s*/u,
	//   push: 'arg',
	//   value: src => src.substring(1).trim()
	// }
	if match := patternArgument.FindString(remaining); match != "" {
		// Process value: substring(1).trim()
		value := strings.TrimSpace(match[1:])
		lineBreaks := l.countLineBreaks(match)
		token := l.createToken(TokenArgument, value, match, lineBreaks)
		l.advance(len(match))
		l.pushState(LexerStateArg)
		return &token
	}

	// octothorpe: '#'
	if remaining[0] == '#' {
		token := l.createToken(TokenOctothorpe, "#", "#", 0)
		l.advance(1)
		return &token
	}

	// end: { match: '}', pop: 1 }
	if remaining[0] == '}' {
		token := l.createToken(TokenEnd, "}", "}", 0)
		l.advance(1)
		l.popState()
		return &token
	}

	// content: { lineBreaks: true, match: /[^][^{}#']*/u }
	// Note: [^] matches any character including newline
	if match := patternContent.FindString(remaining); match != "" {
		lineBreaks := l.countLineBreaks(match)
		token := l.createToken(TokenContent, match, match, lineBreaks)
		l.advance(len(match))
		return &token
	}

	// If no pattern matches, consume one character as content
	if remaining != "" {
		r, size := utf8.DecodeRuneInString(remaining)
		char := string(r)
		lineBreaks := 0
		if r == '\n' {
			lineBreaks = 1
		}
		token := l.createToken(TokenContent, char, char, lineBreaks)
		l.advance(size)
		return &token
	}

	return nil
}

// matchArgState matches patterns in arg state
func (l *Lexer) matchArgState() *LexerToken {
	remaining := l.remaining()
	if remaining == "" {
		return nil
	}

	// select: {
	//   lineBreaks: true,
	//   match: /,\s*(?:plural|select|selectordinal)\s*,\s*/u,
	//   next: 'select',
	//   value: src => src.split(',')[1].trim()
	// }
	if match := patternSelect.FindString(remaining); match != "" {
		// Process value: split(',')[1].trim()
		parts := strings.Split(match, ",")
		value := strings.TrimSpace(parts[1])
		lineBreaks := l.countLineBreaks(match)
		token := l.createToken(TokenSelect, value, match, lineBreaks)
		l.advance(len(match))
		l.nextState(LexerStateSelect)
		return &token
	}

	// 'func-args': {
	//   lineBreaks: true,
	//   match: /,\s*[^\p{Pat_Syn}\p{Pat_WS}]+\s*,/u,
	//   next: 'body',
	//   value: src => src.split(',')[1].trim()
	// }
	if match := patternFuncArgs.FindString(remaining); match != "" {
		// Process value: split(',')[1].trim()
		parts := strings.Split(match, ",")
		value := strings.TrimSpace(parts[1])
		lineBreaks := l.countLineBreaks(match)
		token := l.createToken(TokenFuncArgs, value, match, lineBreaks)
		l.advance(len(match))
		l.nextState(LexerStateBody)
		return &token
	}

	// 'func-simple': {
	//   lineBreaks: true,
	//   match: /,\s*[^\p{Pat_Syn}\p{Pat_WS}]+\s*/u,
	//   value: src => src.substring(1).trim()
	// }
	if match := patternFuncSimple.FindString(remaining); match != "" {
		// Process value: substring(1).trim()
		value := strings.TrimSpace(match[1:])
		lineBreaks := l.countLineBreaks(match)
		token := l.createToken(TokenFuncSimple, value, match, lineBreaks)
		l.advance(len(match))
		return &token
	}

	// end: { match: '}', pop: 1 }
	if remaining[0] == '}' {
		token := l.createToken(TokenEnd, "}", "}", 0)
		l.advance(1)
		l.popState()
		return &token
	}

	return nil
}

// matchSelectState matches patterns in select state
func (l *Lexer) matchSelectState() *LexerToken {
	remaining := l.remaining()
	if remaining == "" {
		return nil
	}

	// offset: {
	//   lineBreaks: true,
	//   match: /\s*offset\s*:\s*\d+\s*/u,
	//   value: src => src.split(':')[1].trim()
	// }
	if match := patternOffset.FindString(remaining); match != "" {
		// Process value: split(':')[1].trim()
		parts := strings.Split(match, ":")
		value := strings.TrimSpace(parts[1])
		lineBreaks := l.countLineBreaks(match)
		token := l.createToken(TokenOffset, value, match, lineBreaks)
		l.advance(len(match))
		return &token
	}

	// case: {
	//   lineBreaks: true,
	//   match: /\s*(?:=\d+|[^\p{Pat_Syn}\p{Pat_WS}]+)\s*\{/u,
	//   push: 'body',
	//   value: src => src.substring(0, src.indexOf('{')).trim()
	// }
	if match := patternCase.FindString(remaining); match != "" {
		// Process value: substring(0, indexOf('{')).trim()
		before, _, _ := strings.Cut(match, "{")
		value := strings.TrimSpace(before)
		lineBreaks := l.countLineBreaks(match)
		token := l.createToken(TokenCase, value, match, lineBreaks)
		l.advance(len(match))
		l.pushState(LexerStateBody)
		return &token
	}

	// end: { match: /\s*\}/u, pop: 1 }
	if match := patternSelectEnd.FindString(remaining); match != "" {
		token := l.createToken(TokenEnd, "}", match, 0)
		l.advance(len(match))
		l.popState()
		return &token
	}

	return nil
}

// NextToken returns the next token from the lexer
func (l *Lexer) NextToken() *LexerToken {
	for !l.atEnd() {
		state := l.getCurrentState()

		var token *LexerToken
		switch state {
		case LexerStateBody:
			token = l.matchBodyState()
		case LexerStateArg:
			token = l.matchArgState()
		case LexerStateSelect:
			token = l.matchSelectState()
		}

		if token != nil {
			return token
		}

		// If no token was matched but we're not at end, something is wrong
		// Skip one character and continue
		if !l.atEnd() {
			l.advance(1)
		}
	}

	return nil
}

// Tokenize tokenizes the entire source and returns all tokens
func (l *Lexer) Tokenize() ([]LexerToken, error) {
	var tokens []LexerToken

	for {
		token := l.NextToken()
		if token == nil {
			break
		}
		tokens = append(tokens, *token)
	}

	l.tokens = tokens
	return tokens, nil
}

// Iterator interface for compatibility with TypeScript's for...of pattern
func (l *Lexer) Iterator() func() *LexerToken {
	if l.tokens == nil {
		_, _ = l.Tokenize() // Explicitly ignore both return values in iterator context
	}

	index := 0
	return func() *LexerToken {
		if index >= len(l.tokens) {
			return nil
		}
		token := &l.tokens[index]
		index++
		return token
	}
}

// FormatError formats an error message with position information like TypeScript
func (l *Lexer) FormatError(token *LexerToken, message string) string {
	if token == nil {
		return fmt.Sprintf("ParseError: %s", message)
	}

	// Find the line in source
	lines := strings.Split(l.source, "\n")
	if token.Line <= 0 || token.Line > len(lines) {
		return fmt.Sprintf("ParseError: %s at line %d col %d", message, token.Line, token.Col)
	}

	line := lines[token.Line-1]
	pointer := strings.Repeat(" ", token.Col-1) + "^"

	return fmt.Sprintf("ParseError: %s at line %d col %d:\n\n%s\n%s",
		message, token.Line, token.Col, line, pointer)
}

// Global lexer instance for compatibility
var globalLexer = NewLexer("")

// Reset resets the global lexer with new source
func ResetLexer(source string) *Lexer {
	globalLexer.Reset(source)
	return globalLexer
}
