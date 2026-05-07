// parser.go - ICU MessageFormat parser implementation
// TypeScript original code:
// /**
//  * An AST parser for ICU MessageFormat strings
//  */
// import { lexer } from './lexer.js';
// import { Lexer, Token as LexerToken } from 'moo';
//
// export type Token = Content | PlainArg | FunctionArg | Select | Octothorpe;
//
// export interface Content {
//   type: 'content';
//   value: string;
//   ctx: Context;
// }
//
// export interface PlainArg {
//   type: 'argument';
//   arg: string;
//   ctx: Context;
// }
//
// export interface FunctionArg {
//   type: 'function';
//   arg: string;
//   key: string;
//   param?: Array<Content | PlainArg | FunctionArg | Select | Octothorpe>;
//   ctx: Context;
// }
//
// export interface Select {
//   type: 'plural' | 'select' | 'selectordinal';
//   arg: string;
//   cases: SelectCase[];
//   pluralOffset?: number;
//   ctx: Context;
// }
//
// export interface SelectCase {
//   key: string;
//   tokens: Array<Content | PlainArg | FunctionArg | Select | Octothorpe>;
//   ctx: Context;
// }
//
// export interface Octothorpe {
//   type: 'octothorpe';
//   ctx: Context;
// }
//
// export interface Context {
//   offset: number;
//   line: number;
//   col: number;
//   text: string;
//   lineBreaks: number;
// }

package v1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Token interface for all AST node types
type Token interface {
	GetType() string
	GetContext() Context
}

// Context represents parsing context with position information
type Context struct {
	Offset     int    // Token start index from the beginning of the input string
	Line       int    // Token start line number, starting from 1
	Col        int    // Token start column, starting from 1
	Text       string // The raw input source for the token
	LineBreaks int    // The number of line breaks consumed while parsing the token
}

// Content represents text content of the message
type Content struct {
	Type  string  `json:"type"`
	Value string  `json:"value"`
	Ctx   Context `json:"ctx"`
}

func (c *Content) GetType() string     { return c.Type }
func (c *Content) GetContext() Context { return c.Ctx }

// PlainArg represents a simple placeholder
type PlainArg struct {
	Type string  `json:"type"`
	Arg  string  `json:"arg"`
	Ctx  Context `json:"ctx"`
}

func (p *PlainArg) GetType() string     { return p.Type }
func (p *PlainArg) GetContext() Context { return p.Ctx }

// FunctionArg represents a placeholder for a mapped argument
type FunctionArg struct {
	Type  string  `json:"type"`
	Arg   string  `json:"arg"`
	Key   string  `json:"key"`
	Param []Token `json:"param,omitempty"`
	Ctx   Context `json:"ctx"`
}

func (f *FunctionArg) GetType() string     { return f.Type }
func (f *FunctionArg) GetContext() Context { return f.Ctx }

// Select represents a selector between multiple variants
type Select struct {
	Type         string       `json:"type"`
	Arg          string       `json:"arg"`
	Cases        []SelectCase `json:"cases"`
	PluralOffset *int         `json:"pluralOffset,omitempty"`
	Ctx          Context      `json:"ctx"`
}

func (s *Select) GetType() string     { return s.Type }
func (s *Select) GetContext() Context { return s.Ctx }

// SelectCase represents a case within a Select
type SelectCase struct {
	Key    string  `json:"key"`
	Tokens []Token `json:"tokens"`
	Ctx    Context `json:"ctx"`
}

// Octothorpe represents the # character
type Octothorpe struct {
	Type string  `json:"type"`
	Ctx  Context `json:"ctx"`
}

func (o *Octothorpe) GetType() string     { return o.Type }
func (o *Octothorpe) GetContext() Context { return o.Ctx }

// Note: PluralCategory is defined in plurals.go

// ParseOptions represents options for the parser
type ParseOptions struct {
	// Array of valid plural categories for the current locale, used to validate `plural` keys.
	// If nil, the full set of valid PluralCategory keys is used.
	// To disable this check, pass in an empty array.
	Cardinal []PluralCategory `json:"cardinal,omitempty"`

	// Array of valid plural categories for the current locale, used to validate `selectordinal` keys.
	// If nil, the full set of valid PluralCategory keys is used.
	// To disable this check, pass in an empty array.
	Ordinal []PluralCategory `json:"ordinal,omitempty"`

	// By default, the parsing applies a few relaxations to the ICU MessageFormat spec.
	// Setting Strict to true will disable these relaxations.
	Strict bool `json:"strict,omitempty"`

	// By default, the parser will reject any plural keys that are not valid
	// Unicode CLDR plural category keys.
	// Setting StrictPluralKeys to false will disable this check.
	StrictPluralKeys *bool `json:"strictPluralKeys,omitempty"`
}

// ParseError represents a parsing error
type ParseError struct {
	Message string
	Token   *LexerToken
}

func (pe *ParseError) Error() string {
	if pe.Token == nil {
		return fmt.Sprintf("ParseError: %s", pe.Message)
	}
	return globalLexer.FormatError(pe.Token, pe.Message)
}

// NewParseError creates a new ParseError
func NewParseError(token *LexerToken, message string) *ParseError {
	return &ParseError{
		Message: message,
		Token:   token,
	}
}

// Parser implements the MessageFormat parser following TypeScript's structure
type Parser struct {
	lexer            *Lexer
	tokens           []LexerToken
	tokenIndex       int
	strict           bool
	cardinalKeys     []PluralCategory
	ordinalKeys      []PluralCategory
	strictPluralKeys bool
}

// NewParser creates a new parser instance
func NewParser(src string, options *ParseOptions) *Parser {
	lexer := NewLexer(src)
	tokens, _ := lexer.Tokenize()

	// Set default options
	cardinalKeys := []PluralCategory{PluralZero, PluralOne, PluralTwo, PluralFew, PluralMany, PluralOther}
	ordinalKeys := []PluralCategory{PluralZero, PluralOne, PluralTwo, PluralFew, PluralMany, PluralOther}
	strict := false
	strictPluralKeys := true

	if options != nil {
		if options.Cardinal != nil {
			cardinalKeys = options.Cardinal
		}
		if options.Ordinal != nil {
			ordinalKeys = options.Ordinal
		}
		strict = options.Strict
		if options.StrictPluralKeys != nil {
			strictPluralKeys = *options.StrictPluralKeys
		}
	}

	return &Parser{
		lexer:            lexer,
		tokens:           tokens,
		tokenIndex:       0,
		strict:           strict,
		cardinalKeys:     cardinalKeys,
		ordinalKeys:      ordinalKeys,
		strictPluralKeys: strictPluralKeys,
	}
}

// getContext converts LexerToken to Context
func getContext(lt *LexerToken) Context {
	return Context{
		Offset:     lt.Offset,
		Line:       lt.Line,
		Col:        lt.Col,
		Text:       lt.Text,
		LineBreaks: lt.LineBreaks,
	}
}

// nextToken returns the next token and advances the index
func (p *Parser) nextToken() *LexerToken {
	if p.tokenIndex >= len(p.tokens) {
		return nil
	}
	token := &p.tokens[p.tokenIndex]
	p.tokenIndex++
	return token
}

// peekToken returns the next token without advancing
func (p *Parser) peekToken() *LexerToken {
	if p.tokenIndex >= len(p.tokens) {
		return nil
	}
	return &p.tokens[p.tokenIndex]
}

// isSelectType checks if a type string is a valid select type
func isSelectType(typeStr string) bool {
	return typeStr == "plural" || typeStr == "select" || typeStr == "selectordinal"
}

// strictArgStyleParam processes parameters in strict mode
func strictArgStyleParam(lt *LexerToken, param []Token) ([]Token, error) {
	value := ""
	text := ""

	for _, p := range param {
		pText := p.GetContext().Text
		text += pText

		switch token := p.(type) {
		case *Content:
			value += token.Value
		case *PlainArg:
			value += pText
		case *FunctionArg:
			value += pText
		case *Octothorpe:
			value += pText
		default:
			return nil, NewParseError(lt, fmt.Sprintf("Unsupported part in strict mode function arg style: %s", pText))
		}
	}

	// Create combined context
	ctx := param[0].GetContext()
	ctx.Text = text

	content := &Content{
		Type:  "content",
		Value: strings.TrimSpace(value),
		Ctx:   ctx,
	}

	return []Token{content}, nil
}

// strictArgTypes defines valid argument types in strict mode
var strictArgTypes = []string{
	"number", "date", "time", "spellout", "ordinal", "duration",
}

// isStrictArgType checks if an argument type is valid in strict mode
func isStrictArgType(argType string) bool {
	for _, valid := range strictArgTypes {
		if argType == valid {
			return true
		}
	}
	return false
}

// checkSelectKey validates select case keys
func (p *Parser) checkSelectKey(lt *LexerToken, selectType string, key string) error {
	if key != "" && key[0] == '=' {
		if selectType == "select" {
			return NewParseError(lt, fmt.Sprintf("The case %s is not valid with select", key))
		}
		// For plural and selectordinal, = prefix must be followed by a number
		if len(key) == 1 {
			return NewParseError(lt, fmt.Sprintf("Invalid exact match key: %s", key))
		}
		// Check if the part after '=' is a valid number
		numPart := key[1:]
		if _, err := strconv.Atoi(numPart); err != nil {
			return NewParseError(lt, fmt.Sprintf("Invalid exact match key: %s (must be a number after =)", key))
		}
	} else if selectType != "select" {
		var keys []PluralCategory
		if selectType == "plural" {
			keys = p.cardinalKeys
		} else {
			keys = p.ordinalKeys
		}

		if p.strictPluralKeys && len(keys) > 0 {
			found := false
			for _, validKey := range keys {
				if string(validKey) == key {
					found = true
					break
				}
			}
			if !found {
				return NewParseError(lt, fmt.Sprintf("The %s case %s is not valid in this locale", selectType, key))
			}
		}
	}
	return nil
}

// parseSelect parses a select/plural/selectordinal statement
func (p *Parser) parseSelect(argToken *LexerToken, inPlural bool, ctx Context, selectType string) (*Select, error) {
	sel := &Select{
		Type:  selectType,
		Arg:   argToken.Value,
		Cases: []SelectCase{},
		Ctx:   ctx,
	}

	if selectType == "plural" || selectType == "selectordinal" {
		inPlural = true
	} else if p.strict {
		inPlural = false
	}

	for {
		lt := p.nextToken()
		if lt == nil {
			return nil, NewParseError(nil, "Unexpected message end")
		}

		switch lt.Type {
		case TokenOffset:
			if selectType == "select" {
				return nil, NewParseError(lt, "Unexpected plural offset for select")
			}
			if len(sel.Cases) > 0 {
				return nil, NewParseError(lt, "Plural offset must be set before cases")
			}
			offset, err := strconv.Atoi(lt.Value)
			if err != nil {
				return nil, NewParseError(lt, fmt.Sprintf("Invalid offset value: %s", lt.Value))
			}
			if offset < 0 {
				return nil, NewParseError(lt, "Plural offset must be non-negative")
			}
			sel.PluralOffset = &offset
			ctx.Text += lt.Text
			ctx.LineBreaks += lt.LineBreaks

		case TokenCase:
			err := p.checkSelectKey(lt, selectType, lt.Value)
			if err != nil {
				var parseErr *ParseError
				if errors.As(err, &parseErr) {
					parseErr.Token = lt
				}
				return nil, err
			}

			caseTokens, err := p.parseBody(inPlural, false)
			if err != nil {
				return nil, err
			}

			sel.Cases = append(sel.Cases, SelectCase{
				Key:    lt.Value,
				Tokens: caseTokens,
				Ctx:    getContext(lt),
			})

		case TokenEnd:
			return sel, nil

		default:
			return nil, NewParseError(lt, fmt.Sprintf("Unexpected lexer token: %s", lt.Type))
		}
	}
}

// parseArgToken parses an argument token
func (p *Parser) parseArgToken(lt *LexerToken, inPlural bool) (Token, error) {
	ctx := getContext(lt)

	argType := p.nextToken()
	if argType == nil {
		return nil, NewParseError(nil, "Unexpected message end")
	}

	ctx.Text += argType.Text
	ctx.LineBreaks += argType.LineBreaks

	// Check strict mode restrictions
	if p.strict && (argType.Type == TokenFuncSimple || argType.Type == TokenFuncArgs) {
		if !isStrictArgType(argType.Value) {
			return nil, NewParseError(lt, fmt.Sprintf("Invalid strict mode function arg type: %s", argType.Value))
		}
	}

	switch argType.Type {
	case TokenEnd:
		return &PlainArg{
			Type: "argument",
			Arg:  lt.Value,
			Ctx:  ctx,
		}, nil

	case TokenFuncSimple:
		end := p.nextToken()
		if end == nil {
			return nil, NewParseError(nil, "Unexpected message end")
		}
		if end.Type != TokenEnd {
			return nil, NewParseError(end, fmt.Sprintf("Unexpected lexer token: %s", end.Type))
		}
		ctx.Text += end.Text

		if isSelectType(strings.ToLower(argType.Value)) {
			return nil, NewParseError(argType, fmt.Sprintf("Invalid type identifier: %s", argType.Value))
		}

		return &FunctionArg{
			Type: "function",
			Arg:  lt.Value,
			Key:  argType.Value,
			Ctx:  ctx,
		}, nil

	case TokenFuncArgs:
		if isSelectType(strings.ToLower(argType.Value)) {
			return nil, NewParseError(argType, fmt.Sprintf("Invalid type identifier: %s", argType.Value))
		}

		param, err := p.parseBody(p.strict && !inPlural || !p.strict && inPlural, false)
		if err != nil {
			return nil, err
		}

		if p.strict && len(param) > 0 {
			param, err = strictArgStyleParam(lt, param)
			if err != nil {
				return nil, err
			}
		}

		return &FunctionArg{
			Type:  "function",
			Arg:   lt.Value,
			Key:   argType.Value,
			Param: param,
			Ctx:   ctx,
		}, nil

	case TokenSelect:
		if isSelectType(argType.Value) {
			return p.parseSelect(lt, inPlural, ctx, argType.Value)
		} else {
			return nil, NewParseError(argType, fmt.Sprintf("Unexpected select type %s", argType.Value))
		}

	default:
		return nil, NewParseError(argType, fmt.Sprintf("Unexpected lexer token: %s", argType.Type))
	}
}

// parseBody parses the body of a message or case
func (p *Parser) parseBody(inPlural bool, atRoot bool) ([]Token, error) {
	tokens := []Token{}
	var content *Content = nil

	for {
		lt := p.peekToken()
		if lt == nil {
			if atRoot {
				return tokens, nil
			}
			return nil, NewParseError(nil, "Unexpected message end")
		}

		// Advance token index since we're consuming this token
		p.nextToken()

		switch {
		case lt.Type == TokenArgument:
			if content != nil {
				content = nil
			}
			argToken, err := p.parseArgToken(lt, inPlural)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, argToken)
		case lt.Type == TokenOctothorpe && inPlural:
			if content != nil {
				content = nil
			}
			tokens = append(tokens, &Octothorpe{
				Type: "octothorpe",
				Ctx:  getContext(lt),
			})
		case lt.Type == TokenEnd && !atRoot:
			return tokens, nil
		default:
			value := lt.Value

			// Handle quoted patterns in non-plural contexts
			if !inPlural && lt.Type == TokenQuoted && len(value) > 0 && value[0] == '#' {
				if strings.Contains(value, "{") {
					return nil, NewParseError(lt, fmt.Sprintf("Unsupported escape pattern: %s", value))
				}
				value = lt.Text // Use original text with quotes
			}

			if content != nil {
				// Append to existing content token
				content.Value += value
				content.Ctx.Text += lt.Text
				content.Ctx.LineBreaks += lt.LineBreaks
			} else {
				// Create new content token
				content = &Content{
					Type:  "content",
					Value: value,
					Ctx:   getContext(lt),
				}
				tokens = append(tokens, content)
			}
		}
	}
}

// Parse parses the message and returns the AST tokens
func (p *Parser) Parse() ([]Token, error) {
	return p.parseBody(false, true)
}

// Parse function that matches TypeScript's parse signature
func Parse(src string, options *ParseOptions) ([]Token, error) {
	parser := NewParser(src, options)
	return parser.Parse()
}
