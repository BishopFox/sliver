package kong

import (
	"fmt"
	"strings"
)

// TokenType is the type of a token.
type TokenType int

// Token types.
const (
	UntypedToken TokenType = iota
	EOLToken
	FlagToken               // --<flag>
	FlagValueToken          // =<value>
	ShortFlagToken          // -<short>[<tail]
	ShortFlagTailToken      // <tail>
	PositionalArgumentToken // <arg>
)

func (t TokenType) String() string {
	switch t {
	case UntypedToken:
		return "untyped"
	case EOLToken:
		return "<EOL>"
	case FlagToken: // --<flag>
		return "long flag"
	case FlagValueToken: // =<value>
		return "flag value"
	case ShortFlagToken: // -<short>[<tail]
		return "short flag"
	case ShortFlagTailToken: // <tail>
		return "short flag remainder"
	case PositionalArgumentToken: // <arg>
		return "positional argument"
	}
	panic("unsupported type")
}

// Token created by Scanner.
type Token struct {
	Value interface{}
	Type  TokenType
}

func (t Token) String() string {
	switch t.Type {
	case FlagToken:
		return fmt.Sprintf("--%v", t.Value)

	case ShortFlagToken:
		return fmt.Sprintf("-%v", t.Value)

	case EOLToken:
		return "EOL"

	default:
		return fmt.Sprintf("%v", t.Value)
	}
}

// IsEOL returns true if this Token is past the end of the line.
func (t Token) IsEOL() bool {
	return t.Type == EOLToken
}

// IsAny returns true if the token's type is any of those provided.
func (t TokenType) IsAny(types ...TokenType) bool {
	for _, typ := range types {
		if t == typ {
			return true
		}
	}
	return false
}

// InferredType tries to infer the type of a token.
func (t Token) InferredType() TokenType {
	if t.Type != UntypedToken {
		return t.Type
	}
	if v, ok := t.Value.(string); ok {
		if strings.HasPrefix(v, "--") { // nolint: gocritic
			return FlagToken
		} else if v == "-" {
			return PositionalArgumentToken
		} else if strings.HasPrefix(v, "-") {
			return ShortFlagToken
		}
	}
	return t.Type
}

// IsValue returns true if token is usable as a parseable value.
//
// A parseable value is either a value typed token, or an untyped token NOT starting with a hyphen.
func (t Token) IsValue() bool {
	tt := t.InferredType()
	return tt.IsAny(FlagValueToken, ShortFlagTailToken, PositionalArgumentToken) ||
		(tt == UntypedToken && !strings.HasPrefix(t.String(), "-"))
}

// Scanner is a stack-based scanner over command-line tokens.
//
// Initially all tokens are untyped. As the parser consumes tokens it assigns types, splits tokens, and pushes them back
// onto the stream.
//
// For example, the token "--foo=bar" will be split into the following by the parser:
//
// 		[{FlagToken, "foo"}, {FlagValueToken, "bar"}]
type Scanner struct {
	args []Token
}

// ScanAsType creates a new Scanner from args with the given type.
func ScanAsType(ttype TokenType, args ...string) *Scanner {
	s := &Scanner{}
	for _, arg := range args {
		s.args = append(s.args, Token{Value: arg, Type: ttype})
	}
	return s
}

// Scan creates a new Scanner from args with untyped tokens.
func Scan(args ...string) *Scanner {
	return ScanAsType(UntypedToken, args...)
}

// ScanFromTokens creates a new Scanner from a slice of tokens.
func ScanFromTokens(tokens ...Token) *Scanner {
	return &Scanner{args: tokens}
}

// Len returns the number of input arguments.
func (s *Scanner) Len() int {
	return len(s.args)
}

// Pop the front token off the Scanner.
func (s *Scanner) Pop() Token {
	if len(s.args) == 0 {
		return Token{Type: EOLToken}
	}
	arg := s.args[0]
	s.args = s.args[1:]
	return arg
}

type expectedError struct {
	context string
	token   Token
}

func (e *expectedError) Error() string {
	return fmt.Sprintf("expected %s value but got %q (%s)", e.context, e.token, e.token.InferredType())
}

// PopValue pops a value token, or returns an error.
//
// "context" is used to assist the user if the value can not be popped, eg. "expected <context> value but got <type>"
func (s *Scanner) PopValue(context string) (Token, error) {
	t := s.Pop()
	if !t.IsValue() {
		return t, &expectedError{context, t}
	}
	return t, nil
}

// PopValueInto pops a value token into target or returns an error.
//
// "context" is used to assist the user if the value can not be popped, eg. "expected <context> value but got <type>"
func (s *Scanner) PopValueInto(context string, target interface{}) error {
	t, err := s.PopValue(context)
	if err != nil {
		return err
	}
	return jsonTranscode(t.Value, target)
}

// PopWhile predicate returns true.
func (s *Scanner) PopWhile(predicate func(Token) bool) (values []Token) {
	for predicate(s.Peek()) {
		values = append(values, s.Pop())
	}
	return
}

// PopUntil predicate returns true.
func (s *Scanner) PopUntil(predicate func(Token) bool) (values []Token) {
	for !predicate(s.Peek()) {
		values = append(values, s.Pop())
	}
	return
}

// Peek at the next Token or return an EOLToken.
func (s *Scanner) Peek() Token {
	if len(s.args) == 0 {
		return Token{Type: EOLToken}
	}
	return s.args[0]
}

// Push an untyped Token onto the front of the Scanner.
func (s *Scanner) Push(arg interface{}) *Scanner {
	s.PushToken(Token{Value: arg})
	return s
}

// PushTyped pushes a typed token onto the front of the Scanner.
func (s *Scanner) PushTyped(arg interface{}, typ TokenType) *Scanner {
	s.PushToken(Token{Value: arg, Type: typ})
	return s
}

// PushToken pushes a preconstructed Token onto the front of the Scanner.
func (s *Scanner) PushToken(token Token) *Scanner {
	s.args = append([]Token{token}, s.args...)
	return s
}
