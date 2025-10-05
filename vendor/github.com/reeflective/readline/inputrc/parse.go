package inputrc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	emacs           = "emacs"
	hexValNum       = 10
	metaSeqLength   = 6
	setDirectiveLen = 4
)

// Parser is a inputrc parser.
type Parser struct {
	haltOnErr bool
	strict    bool
	name      string
	app       string
	term      string
	mode      string
	keymap    string
	line      int
	conds     []bool
	errs      []error
}

// New creates a new inputrc parser.
func New(opts ...Option) *Parser {
	// build parser state
	parser := &Parser{
		line: 1,
	}
	for _, o := range opts {
		o(parser)
	}

	return parser
}

// Parse parses inputrc data from the reader, passing sets and binding keys to
// h based on the configured options.
func (p *Parser) Parse(stream io.Reader, handler Handler) error {
	var err error
	// reset parser state
	p.keymap, p.line, p.conds, p.errs = emacs, 1, append(p.conds[:0], true), p.errs[:0]
	// scan file by lines
	var line []rune
	var pos, end int
	scanner := bufio.NewScanner(stream)

	for ; scanner.Scan(); p.line++ {
		line = []rune(scanner.Text())
		end = len(line)

		if pos = findNonSpace(line, 0, end); pos == end {
			continue
		}
		// skip blank/comment
		switch line[pos] {
		case 0, '\r', '\n', '#':
			continue
		}

		// next
		if err = p.next(handler, line, pos, end); err != nil {
			p.errs = append(p.errs, err)
			if p.haltOnErr {
				return err
			}
		}
	}

	if err = scanner.Err(); err != nil {
		p.errs = append(p.errs, err)
		return err
	}

	return nil
}

// Errs returns the parse errors encountered.
func (p *Parser) Errs() []error {
	return p.errs
}

// next handles the next statement.
func (p *Parser) next(handler Handler, seq []rune, pos, end int) error {
	directive, val, tok, err := p.readNext(seq, pos, end)
	if err != nil {
		return err
	}

	switch tok {
	case tokenBind, tokenBindMacro:
		return p.doBind(handler, directive, val, tok == tokenBindMacro)
	case tokenSet:
		return p.doSet(handler, directive, val)
	case tokenConstruct:
		return p.do(handler, directive, val)
	}

	return nil
}

// readNext reads the next statement.
func (p *Parser) readNext(seq []rune, pos, end int) (string, string, token, error) {
	pos = findNonSpace(seq, pos, end)

	switch {
	case seq[pos] == 's' && grab(seq, pos+1, end) == 'e' && grab(seq, pos+2, end) == 't' && unicode.IsSpace(grab(seq, pos+3, end)):
		// read set
		return p.readSymbols(seq, pos+setDirectiveLen, end, tokenSet, true)
	case seq[pos] == '$':
		// read construct
		return p.readSymbols(seq, pos, end, tokenConstruct, false)
	}
	// read key keySeq
	var keySeq string

	if seq[pos] == '"' || seq[pos] == '\'' {
		var ok bool
		start := pos

		if pos, ok = findStringEnd(seq, pos, end); !ok {
			return "", "", tokenNone, &ParseError{
				Name: p.name,
				Line: p.line,
				Text: string(seq[start:]),
				Err:  ErrBindMissingClosingQuote,
			}
		}

		keySeq = unescapeRunes(seq, start+1, pos-1)
	} else {
		var err error
		if keySeq, pos, err = decodeKey(seq, pos, end); err != nil {
			return "", "", tokenNone, &ParseError{
				Name: p.name,
				Line: p.line,
				Text: string(seq),
				Err:  err,
			}
		}
	}
	// NOTE: this is technically different than the actual readline
	// implementation, as it doesn't allow whitespace, but silently fails (ie
	// does not bind a key) if a space follows the key declaration. made a
	// decision to instead return an error if the : is missing in all cases.
	// seek :
	for ; pos < end && seq[pos] != ':'; pos++ {
	}

	if pos == end || seq[pos] != ':' {
		return "", "", tokenNone, &ParseError{
			Name: p.name,
			Line: p.line,
			Text: string(seq),
			Err:  ErrMissingColon,
		}
	}
	// seek non space
	if pos = findNonSpace(seq, pos+1, end); pos == end || seq[pos] == '#' {
		return keySeq, "", tokenNone, nil
	}
	// seek
	if seq[pos] == '"' || seq[pos] == '\'' {
		var ok bool
		start := pos

		if pos, ok = findStringEnd(seq, pos, end); !ok {
			return "", "", tokenNone, &ParseError{
				Name: p.name,
				Line: p.line,
				Text: string(seq[start:]),
				Err:  ErrMacroMissingClosingQuote,
			}
		}

		return keySeq, unescapeRunes(seq, start+1, pos-1), tokenBindMacro, nil
	}

	return keySeq, string(seq[pos:findEnd(seq, pos, end)]), tokenBind, nil
}

// readSet reads the next two symbols.
func (p *Parser) readSymbols(seq []rune, pos, end int, tok token, allowStrings bool) (string, string, token, error) {
	start := findNonSpace(seq, pos, end)
	pos = findEnd(seq, start, end)
	val := string(seq[start:pos])
	start = findNonSpace(seq, pos, end)
	var ok bool

	if c := grab(seq, start, end); allowStrings || c == '"' || c == '\'' {
		var epos int
		if epos, ok = findStringEnd(seq, start, end); ok {
			pos = epos
		}
	}

	if !allowStrings || !ok {
		pos = findEnd(seq, start, end)
	}

	return val, string(seq[start:pos]), tok, nil
}

// doBind handles a bind.
func (p *Parser) doBind(h Handler, sequence, action string, macro bool) error {
	if !p.conds[len(p.conds)-1] {
		return nil
	}

	return h.Bind(p.keymap, sequence, action, macro)
}

// doSet handles a set.
func (p *Parser) doSet(handler Handler, name, value string) error {
	if !p.conds[len(p.conds)-1] {
		return nil
	}

	switch name {
	case "keymap":
		if p.strict {
			switch value {
			// see: man readline
			// see: https://unix.stackexchange.com/questions/303479/what-are-readlines-modes-keymaps-and-their-default-bindings
			case "emacs", "emacs-standard", "emacs-meta", "emacs-ctlx",
				"vi", "vi-move", "vi-command", "vi-insert":
			default:
				return &ParseError{
					Name: p.name,
					Line: p.line,
					Text: value,
					Err:  ErrInvalidKeymap,
				}
			}
		}

		p.keymap = value

		return nil

	case "editing-mode":
		switch value {
		case "emacs", "vi":
		default:
			return &ParseError{
				Name: p.name,
				Line: p.line,
				Text: value,
				Err:  ErrInvalidEditingMode,
			}
		}

		return handler.Set(name, value)
	}

	if val := handler.Get(name); val != nil {
		// defined in vars, so pass to set only as that type
		var data interface{}
		switch val.(type) {
		case bool:
			data = strings.ToLower(value) == "on" || value == "1"
		case string:
			data = value
		case int:
			i, err := strconv.Atoi(value)
			if err != nil {
				return err
			}

			data = i

		default:
			panic(fmt.Sprintf("unsupported type %T", val))
		}

		return handler.Set(name, data)
	}
	// not set, so try to convert to usable value
	if i, err := strconv.Atoi(value); err == nil {
		return handler.Set(name, i)
	}

	switch strings.ToLower(value) {
	case "off":
		return handler.Set(name, false)
	case "on":
		return handler.Set(name, true)
	}

	return handler.Set(name, value)
}

// do handles a construct.
func (p *Parser) do(handler Handler, keyword, val string) error {
	switch keyword {
	case "$if":
		var eval bool

		switch {
		case strings.HasPrefix(val, "mode="):
			eval = strings.TrimPrefix(val, "mode=") == p.mode
		case strings.HasPrefix(val, "term="):
			eval = strings.TrimPrefix(val, "term=") == p.term
		default:
			eval = strings.ToLower(val) == p.app
		}

		p.conds = append(p.conds, eval)

		return nil

	case "$else":
		if len(p.conds) == 1 {
			return &ParseError{
				Name: p.name,
				Line: p.line,
				Text: "$else",
				Err:  ErrElseWithoutMatchingIf,
			}
		}

		p.conds[len(p.conds)-1] = !p.conds[len(p.conds)-1]

		return nil

	case "$endif":
		if len(p.conds) == 1 {
			return &ParseError{
				Name: p.name,
				Line: p.line,
				Text: "$endif",
				Err:  ErrEndifWithoutMatchingIf,
			}
		}

		p.conds = p.conds[:len(p.conds)-1]

		return nil

	case "$include":
		if !p.conds[len(p.conds)-1] {
			return nil
		}

		path := expandIncludePath(val)
		buf, err := handler.ReadFile(path)

		switch {
		case err != nil && errors.Is(err, os.ErrNotExist):
			return nil
		case err != nil:
			return err
		}

		return Parse(bytes.NewReader(buf), handler, WithName(val), WithApp(p.app), WithTerm(p.term), WithMode(p.mode))
	}

	if !p.conds[len(p.conds)-1] {
		return nil
	}
	// delegate unknown construct
	if err := handler.Do(keyword, val); err != nil {
		return &ParseError{
			Name: p.name,
			Line: p.line,
			Text: keyword + " " + val,
			Err:  err,
		}
	}

	return nil
}

// Option is a parser option.
type Option func(*Parser)

// WithHaltOnErr is a parser option to set halt on every encountered error.
func WithHaltOnErr(haltOnErr bool) Option {
	return func(p *Parser) {
		p.haltOnErr = haltOnErr
	}
}

// WithStrict is a parser option to set strict keymap parsing.
func WithStrict(strict bool) Option {
	return func(p *Parser) {
		p.strict = strict
	}
}

// WithName is a parser option to set the file name.
func WithName(name string) Option {
	return func(p *Parser) {
		p.name = name
	}
}

// WithApp is a parser option to set the app name.
func WithApp(app string) Option {
	return func(p *Parser) {
		p.app = app
	}
}

// WithTerm is a parser option to set the term name.
func WithTerm(term string) Option {
	return func(p *Parser) {
		p.term = term
	}
}

// WithMode is a parser option to set the mode name.
func WithMode(mode string) Option {
	return func(p *Parser) {
		p.mode = mode
	}
}

// ParseError is a parse error.
type ParseError struct {
	Name string
	Line int
	Text string
	Err  error
}

// Error satisfies the error interface.
func (err *ParseError) Error() string {
	var s string
	if err.Name != "" {
		s = " " + err.Name + ":"
	}

	return fmt.Sprintf("inputrc:%s line %d: %s: %v", s, err.Line, err.Text, err.Err)
}

// Unwrap satisfies the errors.Unwrap call.
func (err *ParseError) Unwrap() error {
	return err.Err
}

// token is a inputrc line token.
type token int

// inputrc line tokens.
const (
	tokenNone token = iota
	tokenBind
	tokenBindMacro
	tokenSet
	tokenConstruct
)

// String satisfies the fmt.Stringer interface.
func (tok token) String() string {
	switch tok {
	case tokenNone:
		return "none"
	case tokenBind:
		return "bind"
	case tokenBindMacro:
		return "bind-macro"
	case tokenSet:
		return "set"
	case tokenConstruct:
		return "construct"
	}

	return fmt.Sprintf("token(%d)", tok)
}

// findNonSpace finds first non space rune in r, returning end if not found.
func findNonSpace(r []rune, i, end int) int {
	for ; i < end && unicode.IsSpace(r[i]); i++ {
	}
	return i
}

// findEnd finds end of the current symbol (position of next #, space, or line
// end), returning end if not found.
func findEnd(r []rune, i, end int) int {
	for c := grab(r, i+1, end); i < end && c != '#' && !unicode.IsSpace(c) && !unicode.IsControl(c); i++ {
		c = grab(r, i+1, end)
	}

	return i
}

// findStringEnd finds end of the string, returning end if not found.
func findStringEnd(seq []rune, pos, end int) (int, bool) {
	var char rune
	quote := seq[pos]

	for pos++; pos < end; pos++ {
		switch char = seq[pos]; {
		case char == '\\':
			pos++
			continue
		case char == quote:
			return pos + 1, true
		}
	}

	return pos, false
}

// grab returns r[i] when i < end, 0 otherwise.
func grab(r []rune, i, end int) rune {
	if i < end {
		return r[i]
	}

	return 0
}

// decodeKey decodes named key sequence.
func decodeKey(seq []rune, pos, end int) (string, int, error) {
	// seek end of sequence
	start := pos

	for c := grab(seq, pos+1, end); pos < end && c != ':' && c != '#' && !unicode.IsSpace(c) && !unicode.IsControl(c); pos++ {
		c = grab(seq, pos+1, end)
	}

	val := strings.ToLower(string(seq[start:pos]))
	meta, control := false, false

	for idx := strings.Index(val, "-"); idx != -1; idx = strings.Index(val, "-") {
		switch val[:idx] {
		case "control", "ctrl", "c":
			control = true
		case "meta", "m":
			meta = true
		default:
			return "", idx, ErrUnknownModifier
		}

		val = val[idx+1:]
	}

	var char rune

	switch val {
	case "":
		return "", pos, nil

	case "delete", "del", "rubout":
		char = Delete
	case "escape", "esc":
		char = Esc
	case "newline", "linefeed", "lfd":
		char = Newline
	case "return", "ret":
		char = Return
	case "tab":
		char = Tab
	case "space", "spc":
		char = Space
	case "formfeed", "ffd":
		char = Formfeed
	case "vertical", "vrt":
		char = Vertical
	default:
		char, _ = utf8.DecodeRuneInString(val)
	}

	switch {
	case control && meta:
		return string([]rune{Esc, Encontrol(char)}), pos, nil
	case control:
		char = Encontrol(char)
	case meta:
		char = Enmeta(char)
	}

	return string(char), pos, nil
}

/*
// decodeRunes decodes runes.
func decodeRunes(r []rune, i, end int) string {
	r = []rune(unescapeRunes(r, i, end))
	var s []rune
	var c0, c1, c3 rune
	for i, end = 0, len(r); i < end; i++ {
		c0, c1, c3 = grab(r, i, end), grab(r, i+1, end), grab(r, i+2, end)
		switch {
		case c0 == Meta && c1 == Control, c0 == Control && c1 == Meta:
			s = append(s, Esc, Encontrol(c3))
			i += 2
		case c0 == Control:
			s = append(s, Encontrol(c1))
			i++
		case c0 == Meta:
			s = append(s, Enmeta(c1))
			i++
		default:
			s = append(s, c0)
		}
	}
	return string(s)
}
*/

// unescapeRunes decodes escaped string sequence.
func unescapeRunes(r []rune, i, end int) string {
	var seq []rune
	var char0, char1, char2, char3, char4, char5 rune

	if len(r) == 1 {
		return string(r)
	}

	for ; i < end; i++ {
		if char0 = r[i]; char0 == '\\' {
			char1, char2, char3, char4, char5 = grab(r, i+1, end), grab(r, i+2, end), grab(r, i+3, end), grab(r, i+4, end), grab(r, i+5, end)

			switch {
			case char1 == 'a': // \a alert (bell)
				seq = append(seq, Alert)
				i++
			case char1 == 'b': // \b backspace
				seq = append(seq, Backspace)
				i++
			case char1 == 'd': // \d delete
				seq = append(seq, Delete)
				i++
			case char1 == 'e': // \e escape
				seq = append(seq, Esc)
				i++
			case char1 == 'f': // \f form feed
				seq = append(seq, Formfeed)
				i++
			case char1 == 'n': // \n new line
				seq = append(seq, Newline)
				i++
			case char1 == 'r': // \r carriage return
				seq = append(seq, Return)
				i++
			case char1 == 't': // \t tab
				seq = append(seq, Tab)
				i++
			case char1 == 'v': // \v vertical
				seq = append(seq, Vertical)
				i++
			case char1 == '\\', char1 == '"', char1 == '\'': // \\ \" \' literal
				seq = append(seq, char1)
				i++
			case char1 == 'x' && hexDigit(char2) && hexDigit(char3): // \xHH hex
				seq = append(seq, hexVal(char2)<<4|hexVal(char3))
				i += 2
			case char1 == 'x' && hexDigit(char2): // \xH hex
				seq = append(seq, hexVal(char2))
				i++
			case octDigit(char1) && octDigit(char2) && octDigit(char3): // \nnn octal
				seq = append(seq, (char1-'0')<<6|(char2-'0')<<3|(char3-'0'))
				i += 3
			case octDigit(char1) && octDigit(char2): // \nn octal
				seq = append(seq, (char1-'0')<<3|(char2-'0'))
				i += 2
			case octDigit(char1): // \n octal
				seq = append(seq, char1-'0')
				i++
			case ((char1 == 'C' && char4 == 'M') || (char1 == 'M' && char4 == 'C')) && char2 == '-' && char3 == '\\' && char5 == '-':
				// \C-\M- or \M-\C- control meta prefix
				if c6 := grab(r, i+metaSeqLength, end); c6 != 0 {
					seq = append(seq, Esc, Encontrol(c6))
				}

				i += 6
			case char1 == 'C' && char2 == '-': // \C- control prefix
				if char3 == '?' {
					seq = append(seq, Delete)
				} else {
					seq = append(seq, Encontrol(char3))
				}

				i += 3
			case char1 == 'M' && char2 == '-': // \M- meta prefix
				if char3 == 0 {
					seq = append(seq, Esc)
					i += 2
				} else {
					seq = append(seq, Enmeta(char3))
					i += 3
				}
			default:
				seq = append(seq, char1)
				i++
			}

			continue
		}

		seq = append(seq, char0)
	}

	return string(seq)
}

// octDigit returns true when r is 0-7.
func octDigit(c rune) bool {
	return '0' <= c && c <= '7'
}

// hexDigit returns true when r is 0-9A-Fa-f.
func hexDigit(c rune) bool {
	return '0' <= c && c <= '9' || 'A' <= c && c <= 'F' || 'a' <= c && c <= 'f'
}

// hexVal converts a rune to its hex value.
func hexVal(char rune) rune {
	switch {
	case 'a' <= char && char <= 'f':
		return char - 'a' + hexValNum
	case 'A' <= char && char <= 'F':
		return char - 'A' + hexValNum
	}

	return char - '0'
}

// expandIncludePath handles tilde home directory expansion in $include path directives.
func expandIncludePath(file string) string {
	if !strings.HasPrefix(file, "~/") {
		return file
	}

	u, err := user.Current()
	if err != nil || u == nil || u.HomeDir == "" {
		return file
	}

	return filepath.Join(u.HomeDir, file[2:])
}
