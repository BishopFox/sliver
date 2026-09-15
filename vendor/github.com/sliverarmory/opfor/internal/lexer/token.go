package lexer

import "fmt"

// Kind identifies a lexical token category.
type Kind uint16

const (
	Invalid Kind = iota
	EOF
	Comment

	Identifier
	Keyword
	Operator

	Scalar
	Array
	Hash
	Function
	Reference
	Class

	Integer
	Long
	Double
	SingleString
	DoubleString
	BacktickString

	Semicolon
	Comma
	Colon
	LeftBrace
	RightBrace
	LeftParen
	RightParen
	LeftBracket
	RightBracket
)

var kindNames = [...]string{
	Invalid:        "invalid",
	EOF:            "eof",
	Comment:        "comment",
	Identifier:     "identifier",
	Keyword:        "keyword",
	Operator:       "operator",
	Scalar:         "scalar",
	Array:          "array",
	Hash:           "hash",
	Function:       "function",
	Reference:      "reference",
	Class:          "class",
	Integer:        "integer",
	Long:           "long",
	Double:         "double",
	SingleString:   "single-quoted string",
	DoubleString:   "double-quoted string",
	BacktickString: "backtick string",
	Semicolon:      "semicolon",
	Comma:          "comma",
	Colon:          "colon",
	LeftBrace:      "left brace",
	RightBrace:     "right brace",
	LeftParen:      "left parenthesis",
	RightParen:     "right parenthesis",
	LeftBracket:    "left bracket",
	RightBracket:   "right bracket",
}

// String returns a stable token kind name.
func (k Kind) String() string {
	if int(k) >= 0 && int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return fmt.Sprintf("kind(%d)", k)
}

// Token is one lexical unit. Lexeme is the exact source spelling. Text is the
// useful interior value: it excludes sigils and quote delimiters, but is not
// otherwise decoded or unescaped.
type Token struct {
	Kind Kind

	Lexeme string
	Text   string
	Span   Span

	// TextSpan is the source range occupied by Text when it differs from the
	// complete token spelling. Quoted strings use it to retain the position of
	// their body after the quote delimiters have been removed. Other token
	// kinds leave it empty.
	TextSpan Span

	// LeadingWhitespace and TrailingWhitespace record literal whitespace next
	// to the token. Sleep's grammar uses this information to reject operators
	// whose terms are not separated by whitespace.
	LeadingWhitespace  bool
	TrailingWhitespace bool
}

// String returns a compact debugging representation of a token.
func (t Token) String() string {
	return fmt.Sprintf("%s(%q)", t.Kind, t.Lexeme)
}

// Adjacent reports whether this token and next touch in the source.
func (t Token) Adjacent(next Token) bool {
	return t.Span.Source == next.Span.Source && t.Span.End.Offset == next.Span.Start.Offset
}

// Is reports whether this token has kind and exact source spelling.
func (t Token) Is(kind Kind, lexeme string) bool {
	return t.Kind == kind && t.Lexeme == lexeme
}
