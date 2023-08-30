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
	p := &Parser{
		line: 1,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// Parse parses inputrc data from the reader, passing sets and binding keys to
// h based on the configured options.
func (p *Parser) Parse(r io.Reader, h Handler) error {
	var err error
	// reset parser state
	p.keymap, p.line, p.conds, p.errs = "emacs", 1, append(p.conds[:0], true), p.errs[:0]
	// scan file by lines
	var line []rune
	var i, end int
	s := bufio.NewScanner(r)
	for ; s.Scan(); p.line++ {
		line = []rune(s.Text())
		end = len(line)
		if i = findNonSpace(line, 0, end); i == end {
			continue
		}
		// skip blank/comment
		switch line[i] {
		case 0, '\r', '\n', '#':
			continue
		}
		// next
		if err = p.next(h, line, i, end); err != nil {
			p.errs = append(p.errs, err)
			if p.haltOnErr {
				return err
			}
		}
	}
	if err = s.Err(); err != nil {
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
func (p *Parser) next(h Handler, r []rune, i, end int) error {
	a, b, tok, err := p.readNext(r, i, end)
	if err != nil {
		return err
	}
	switch tok {
	case tokenBind, tokenBindMacro:
		return p.doBind(h, a, b, tok == tokenBindMacro)
	case tokenSet:
		return p.doSet(h, a, b)
	case tokenConstruct:
		return p.do(h, a, b)
	}
	return nil
}

// readNext reads the next statement.
func (p *Parser) readNext(r []rune, i, end int) (string, string, token, error) {
	i = findNonSpace(r, i, end)
	switch {
	case r[i] == 's' && grab(r, i+1, end) == 'e' && grab(r, i+2, end) == 't' && unicode.IsSpace(grab(r, i+3, end)):
		// read set
		return p.readSymbols(r, i+4, end, tokenSet)
	case r[i] == '$':
		// read construct
		return p.readSymbols(r, i, end, tokenConstruct)
	}
	// read key seq
	var seq string
	if r[i] == '"' || r[i] == '\'' {
		start, ok := i, false
		if i, ok = findStringEnd(r, i, end); !ok {
			return "", "", tokenNone, &ParseError{
				Name: p.name,
				Line: p.line,
				Text: string(r[start:]),
				Err:  ErrBindMissingClosingQuote,
			}
		}
		seq = unescapeRunes(r, start+1, i-1)
	} else {
		var err error
		if seq, i, err = decodeKey(r, i, end); err != nil {
			return "", "", tokenNone, &ParseError{
				Name: p.name,
				Line: p.line,
				Text: string(r),
				Err:  err,
			}
		}
	}
	// NOTE: this is technically different than the actual readline
	// implementation, as it doesn't allow whitespace, but silently fails (ie
	// does not bind a key) if a space follows the key declaration. made a
	// decision to instead return an error if the : is missing in all cases.
	// seek :
	for ; i < end && r[i] != ':'; i++ {
	}
	if i == end || r[i] != ':' {
		return "", "", tokenNone, &ParseError{
			Name: p.name,
			Line: p.line,
			Text: string(r),
			Err:  ErrMissingColon,
		}
	}
	// seek non space
	if i = findNonSpace(r, i+1, end); i == end || r[i] == '#' {
		return seq, "", tokenNone, nil
	}
	// seek
	if r[i] == '"' || r[i] == '\'' {
		start, ok := i, false
		if i, ok = findStringEnd(r, i, end); !ok {
			return "", "", tokenNone, &ParseError{
				Name: p.name,
				Line: p.line,
				Text: string(r[start:]),
				Err:  ErrMacroMissingClosingQuote,
			}
		}
		return seq, unescapeRunes(r, start+1, i-1), tokenBindMacro, nil
	}
	return seq, string(r[i:findEnd(r, i, end)]), tokenBind, nil
}

// readSet reads the next two symbols.
func (p *Parser) readSymbols(r []rune, i, end int, tok token) (string, string, token, error) {
	start := findNonSpace(r, i, end)
	i = findEnd(r, start, end)
	a := string(r[start:i])
	start = findNonSpace(r, i, end)
	i = findEnd(r, start, end)
	return a, string(r[start:i]), tok, nil
}

// doBind handles a bind.
func (p *Parser) doBind(h Handler, sequence, action string, macro bool) error {
	if !p.conds[len(p.conds)-1] {
		return nil
	}
	return h.Bind(p.keymap, sequence, action, macro)
}

// doSet handles a set.
func (p *Parser) doSet(h Handler, name, value string) error {
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
		return h.Set(name, value)
	}
	if v := h.Get(name); v != nil {
		// defined in vars, so pass to set only as that type
		var z interface{}
		switch v.(type) {
		case bool:
			z = strings.ToLower(value) == "on" || value == "1"
		case string:
			z = value
		case int:
			i, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			z = i
		default:
			panic(fmt.Sprintf("unsupported type %T", v))
		}
		return h.Set(name, z)
	}
	// not set, so try to convert to usable value
	if i, err := strconv.Atoi(value); err == nil {
		return h.Set(name, i)
	}
	switch strings.ToLower(value) {
	case "off":
		return h.Set(name, false)
	case "on":
		return h.Set(name, true)
	}
	return h.Set(name, value)
}

// do handles a construct.
func (p *Parser) do(h Handler, a, b string) error {
	switch a {
	case "$if":
		var eval bool
		switch {
		case strings.HasPrefix(b, "mode="):
			eval = strings.TrimPrefix(b, "mode=") == p.mode
		case strings.HasPrefix(b, "term="):
			eval = strings.TrimPrefix(b, "term=") == p.term
		default:
			eval = strings.ToLower(b) == p.app
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
		path := expandIncludePath(b)
		buf, err := h.ReadFile(path)
		switch {
		case err != nil && errors.Is(err, os.ErrNotExist):
			return nil
		case err != nil:
			return err
		}
		return Parse(bytes.NewReader(buf), h, WithName(b), WithApp(p.app), WithTerm(p.term), WithMode(p.mode))
	}
	if !p.conds[len(p.conds)-1] {
		return nil
	}
	// delegate unknown construct
	if err := h.Do(a, b); err != nil {
		return &ParseError{
			Name: p.name,
			Line: p.line,
			Text: a + " " + b,
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
func findStringEnd(r []rune, i, end int) (int, bool) {
	quote, c := r[i], rune(0)
	for i++; i < end; i++ {
		switch c = r[i]; {
		case c == '\\':
			i++
			continue
		case c == quote:
			return i + 1, true
		}
	}
	return i, false
}

// grab returns r[i] when i < end, 0 otherwise.
func grab(r []rune, i, end int) rune {
	if i < end {
		return r[i]
	}
	return 0
}

// decodeKey decodes named key sequence.
func decodeKey(r []rune, i, end int) (string, int, error) {
	// seek end of sequence
	start := i
	for c := grab(r, i+1, end); i < end && c != ':' && c != '#' && !unicode.IsSpace(c) && !unicode.IsControl(c); i++ {
		c = grab(r, i+1, end)
	}
	s := strings.ToLower(string(r[start:i]))
	meta, control := false, false
	for i := strings.Index(s, "-"); i != -1; i = strings.Index(s, "-") {
		switch s[:i] {
		case "control", "ctrl", "c":
			control = true
		case "meta", "m":
			meta = true
		default:
			return "", i, ErrUnknownModifier
		}
		s = s[i+1:]
	}
	var c rune
	switch s {
	case "":
		return "", i, nil

	case "delete", "del", "rubout":
		c = Delete
	case "escape", "esc":
		c = Esc
	case "newline", "linefeed", "lfd":
		c = Newline
	case "return", "ret":
		c = Return
	case "tab":
		c = Tab
	case "space", "spc":
		c = Space
	case "formfeed", "ffd":
		c = Formfeed
	case "vertical", "vrt":
		c = Vertical
	default:
		c, _ = utf8.DecodeRuneInString(s)
	}
	switch {
	case control && meta:
		return string([]rune{Esc, Encontrol(c)}), i, nil
	case control:
		c = Encontrol(c)
	case meta:
		c = Enmeta(c)
	}
	return string(c), i, nil
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
	var s []rune
	var c0, c1, c2, c3, c4, c5 rune
	for ; i < end; i++ {
		if c0 = r[i]; c0 == '\\' {
			c1, c2, c3, c4, c5 = grab(r, i+1, end), grab(r, i+2, end), grab(r, i+3, end), grab(r, i+4, end), grab(r, i+5, end)
			switch {
			case c1 == 'a': // \a alert (bell)
				s = append(s, Alert)
				i++
			case c1 == 'b': // \b backspace
				s = append(s, Backspace)
				i++
			case c1 == 'd': // \d delete
				s = append(s, Delete)
				i++
			case c1 == 'e': // \e escape
				s = append(s, Esc)
				i++
			case c1 == 'f': // \f form feed
				s = append(s, Formfeed)
				i++
			case c1 == 'n': // \n new line
				s = append(s, Newline)
				i++
			case c1 == 'r': // \r carriage return
				s = append(s, Return)
				i++
			case c1 == 't': // \t tab
				s = append(s, Tab)
				i++
			case c1 == 'v': // \v vertical
				s = append(s, Vertical)
				i++
			case c1 == '\\', c1 == '"', c1 == '\'': // \\ \" \' literal
				s = append(s, c1)
				i++
			case c1 == 'x' && hexDigit(c2) && hexDigit(c3): // \xHH hex
				s = append(s, hexVal(c2)<<4|hexVal(c3))
				i += 2
			case c1 == 'x' && hexDigit(c2): // \xH hex
				s = append(s, hexVal(c2))
				i++
			case octDigit(c1) && octDigit(c2) && octDigit(c3): // \nnn octal
				s = append(s, (c1-'0')<<6|(c2-'0')<<3|(c3-'0'))
				i += 3
			case octDigit(c1) && octDigit(c2): // \nn octal
				s = append(s, (c1-'0')<<3|(c2-'0'))
				i += 2
			case octDigit(c1): // \n octal
				s = append(s, c1-'0')
				i++
			case ((c1 == 'C' && c4 == 'M') || (c1 == 'M' && c4 == 'C')) && c2 == '-' && c3 == '\\' && c5 == '-':
				// \C-\M- or \M-\C- control meta prefix
				if c6 := grab(r, i+6, end); c6 != 0 {
					s = append(s, Esc, Encontrol(c6))
				}
				i += 6
			case c1 == 'C' && c2 == '-': // \C- control prefix
				if c3 == '?' {
					s = append(s, Delete)
				} else {
					s = append(s, Encontrol(c3))
				}
				i += 3
			case c1 == 'M' && c2 == '-': // \M- meta prefix
				if c3 == 0 {
					s = append(s, Esc)
					i += 2
				} else {
					s = append(s, Enmeta(c3))
					i += 3
				}
			default:
				s = append(s, c1)
				i++
			}
			continue
		}
		s = append(s, c0)
	}
	return string(s)
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
func hexVal(c rune) rune {
	switch {
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return c - '0'
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
