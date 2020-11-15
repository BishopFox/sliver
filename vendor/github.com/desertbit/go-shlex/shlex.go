// Package shlex provides a simple lexical analysis like Unix shell.
package shlex

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"unicode"
)

var (
	ErrNoClosing = errors.New("no closing quotation")
	ErrNoEscaped = errors.New("no escaped character")
)

// Tokenizer is the interface that classifies a token according to
// words, whitespaces, quotations, escapes and escaped quotations.
type Tokenizer interface {
	IsWord(rune) bool
	IsWhitespace(rune) bool
	IsQuote(rune) bool
	IsEscape(rune) bool
	IsEscapedQuote(rune) bool
}

// DefaultTokenizer implements a simple tokenizer like Unix shell.
type DefaultTokenizer struct{}

func (t *DefaultTokenizer) IsWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}
func (t *DefaultTokenizer) IsQuote(r rune) bool {
	switch r {
	case '\'', '"':
		return true
	default:
		return false
	}
}
func (t *DefaultTokenizer) IsWhitespace(r rune) bool {
	return unicode.IsSpace(r)
}
func (t *DefaultTokenizer) IsEscape(r rune) bool {
	return r == '\\'
}
func (t *DefaultTokenizer) IsEscapedQuote(r rune) bool {
	return r == '"'
}

// Lexer represents a lexical analyzer.
type Lexer struct {
	reader          *bufio.Reader
	tokenizer       Tokenizer
	posix           bool
	whitespaceSplit bool
}

// NewLexer creates a new Lexer reading from io.Reader.  This Lexer
// has a DefaultTokenizer according to posix and whitespaceSplit
// rules.
func NewLexer(r io.Reader, posix, whitespaceSplit bool) *Lexer {
	return &Lexer{
		reader:          bufio.NewReader(r),
		tokenizer:       &DefaultTokenizer{},
		posix:           posix,
		whitespaceSplit: whitespaceSplit,
	}
}

// NewLexerString creates a new Lexer reading from a string.  This
// Lexer has a DefaultTokenizer according to posix and whitespaceSplit
// rules.
func NewLexerString(s string, posix, whitespaceSplit bool) *Lexer {
	return NewLexer(strings.NewReader(s), posix, whitespaceSplit)
}

// Split splits a string according to posix or non-posix rules.
func Split(s string, posix bool) ([]string, error) {
	return NewLexerString(s, posix, true).Split()
}

// SetTokenizer sets a Tokenizer.
func (l *Lexer) SetTokenizer(t Tokenizer) {
	l.tokenizer = t
}

func (l *Lexer) Split() ([]string, error) {
	result := make([]string, 0)
	for {
		token, err := l.readToken()
		if token != nil {
			result = append(result, string(token))
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (l *Lexer) readToken() (token []rune, err error) {
	t := l.tokenizer
	quoted := false
	state := ' '
	escapedState := ' '
scanning:
	for {
		next, _, err := l.reader.ReadRune()
		if err != nil {
			if t.IsQuote(state) {
				return token, ErrNoClosing
			} else if t.IsEscape(state) {
				return token, ErrNoEscaped
			}
			return token, err
		}

		switch {
		case t.IsWhitespace(state):
			switch {
			case t.IsWhitespace(next):
				break scanning
			case l.posix && t.IsEscape(next):
				escapedState = 'a'
				state = next
			case t.IsWord(next):
				token = append(token, next)
				state = 'a'
			case t.IsQuote(next):
				if !l.posix {
					token = append(token, next)
				}
				state = next
			default:
				token = []rune{next}
				if l.whitespaceSplit {
					state = 'a'
				} else if token != nil || (l.posix && quoted) {
					break scanning
				}
			}
		case t.IsQuote(state):
			quoted = true
			switch {
			case next == state:
				if !l.posix {
					token = append(token, next)
					break scanning
				} else {
					if token == nil {
						token = []rune{}
					}
					state = 'a'
				}
			case l.posix && t.IsEscape(next) && t.IsEscapedQuote(state):
				escapedState = state
				state = next
			default:
				token = append(token, next)
			}
		case t.IsEscape(state):
			if t.IsQuote(escapedState) && next != state && next != escapedState {
				token = append(token, state)
			}
			token = append(token, next)
			state = escapedState
		case t.IsWord(state):
			switch {
			case t.IsWhitespace(next):
				if token != nil || (l.posix && quoted) {
					break scanning
				}
			case l.posix && t.IsQuote(next):
				state = next
			case l.posix && t.IsEscape(next):
				escapedState = 'a'
				state = next
			case t.IsWord(next) || t.IsQuote(next):
				token = append(token, next)
			default:
				if l.whitespaceSplit {
					token = append(token, next)
				} else if token != nil {
					l.reader.UnreadRune()
					break scanning
				}
			}
		}
	}
	return token, nil
}
