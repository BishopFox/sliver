// Copyright 2022 The Gc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gc // import "modernc.org/gc/v3"

import (
	"bytes"
	"fmt"
	"go/token"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"modernc.org/mathutil"
	mtoken "modernc.org/token"
)

var (
	_ Node = (*Token)(nil)
	_ Node = (*nonode)(nil)

	keywords = map[string]token.Token{
		"break":       BREAK,
		"case":        CASE,
		"chan":        CHAN,
		"const":       CONST,
		"continue":    CONTINUE,
		"default":     DEFAULT,
		"defer":       DEFER,
		"else":        ELSE,
		"fallthrough": FALLTHROUGH,
		"for":         FOR,
		"func":        FUNC,
		"go":          GO,
		"goto":        GOTO,
		"if":          IF,
		"import":      IMPORT,
		"interface":   INTERFACE,
		"map":         MAP,
		"package":     PACKAGE,
		"range":       RANGE,
		"return":      RETURN,
		"select":      SELECT,
		"struct":      STRUCT,
		"switch":      SWITCH,
		"type":        TYPE,
		"var":         VAR,
	}

	lineCommentTag = []byte("line ")
	znode          = &nonode{}
)

type nonode struct{}

func (*nonode) Position() (r token.Position) { return r }
func (*nonode) Source(full bool) string      { return "" }

// Token represents a lexeme, its position and its semantic value.
type Token struct { // 16 bytes on 64 bit arch
	source *source

	ch    int32
	index int32
}

// Ch returns which token t represents
func (t Token) Ch() token.Token { return token.Token(t.ch) }

// Source implements Node.
func (t Token) Source(full bool) string {
	// trc("%10s %v: #%v sep %v, src %v, buf %v", tokSource(t.Ch()), t.Position(), t.index, t.source.toks[t.index].sep, t.source.toks[t.index].src, len(t.source.buf))
	sep := t.Sep()
	if !full && sep != "" {
		sep = " "
	}
	src := t.Src()
	if !full && strings.ContainsRune(src, '\n') {
		src = " "
	}
	// trc("%q %q -> %q %q", t.Sep(), t.Src(), sep, src)
	return sep + src
}

// Positions implements Node.
func (t Token) Position() (r token.Position) {
	if t.source == nil {
		return r
	}

	s := t.source
	off := mathutil.MinInt32(int32(len(s.buf)), s.toks[t.index].src)
	return token.Position(s.file.PositionFor(mtoken.Pos(s.base+off), true))
}

// Prev returns the token preceding t or a zero value if no such token exists.
func (t Token) Prev() (r Token) {
	if index := t.index - 1; index >= 0 {
		s := t.source
		return Token{source: s, ch: s.toks[index].ch, index: index}
	}

	return r
}

// Next returns the token following t or a zero value if no such token exists.
func (t Token) Next() (r Token) {
	if index := t.index + 1; index < int32(len(t.source.toks)) {
		s := t.source
		return Token{source: s, ch: s.toks[index].ch, index: index}
	}

	return r
}

// Sep returns any separators, combined, preceding t.
func (t Token) Sep() string {
	s := t.source
	if p, ok := s.sepPatches[t.index]; ok {
		return p
	}

	return string(s.buf[s.toks[t.index].sep:s.toks[t.index].src])
}

// SetSep sets t's separator.
func (t Token) SetSep(s string) {
	src := t.source
	if src.sepPatches == nil {
		src.sepPatches = map[int32]string{}
	}
	src.sepPatches[t.index] = s
}

// Src returns t's source form.
func (t Token) Src() string {
	s := t.source
	if p, ok := s.srcPatches[t.index]; ok {
		return p
	}

	if t.ch != int32(EOF) {
		next := t.source.off
		if t.index < int32(len(s.toks))-1 {
			next = s.toks[t.index+1].sep
		}
		return string(s.buf[s.toks[t.index].src:next])
	}

	return ""
}

// SetSrc sets t's source form.
func (t Token) SetSrc(s string) {
	src := t.source
	if src.srcPatches == nil {
		src.srcPatches = map[int32]string{}
	}
	src.srcPatches[t.index] = s
}

// IsValid reports t is a valid token. Zero value reports false.
func (t Token) IsValid() bool { return t.source != nil }

type tok struct { // 12 bytes
	ch  int32
	sep int32
	src int32
}

func (t *tok) token() token.Token { return token.Token(t.ch) }

func (t *tok) position(s *source) (r token.Position) {
	off := mathutil.MinInt32(int32(len(s.buf)), t.src)
	return token.Position(s.file.PositionFor(mtoken.Pos(s.base+off), true))
}

// source represents a single Go source file, editor text buffer etc.
type source struct {
	buf        []byte
	file       *mtoken.File
	name       string
	sepPatches map[int32]string
	srcPatches map[int32]string
	toks       []tok

	base int32
	off  int32
}

// 'buf' becomes owned by the result and must not be modified afterwards.
func newSource(name string, buf []byte) *source {
	file := mtoken.NewFile(name, len(buf))
	return &source{
		buf:  buf,
		file: file,
		name: name,
		base: int32(file.Base()),
	}
}

type ErrWithPosition struct {
	pos token.Position
	err error
}

func (e ErrWithPosition) String() string {
	switch {
	case e.pos.IsValid():
		return fmt.Sprintf("%v: %v", e.pos, e.err)
	default:
		return fmt.Sprintf("%v", e.err)
	}
}

type errList []ErrWithPosition

func (e errList) Err() (r error) {
	if len(e) == 0 {
		return nil
	}

	return e
}

func (e errList) Error() string {
	w := 0
	prev := ErrWithPosition{pos: token.Position{Offset: -1}}
	for _, v := range e {
		if v.pos.Line == 0 || v.pos.Offset != prev.pos.Offset || v.err.Error() != prev.err.Error() {
			e[w] = v
			w++
			prev = v
		}
	}

	var a []string
	for _, v := range e {
		a = append(a, fmt.Sprint(v))
	}
	return strings.Join(a, "\n")
}

func (e *errList) err(pos token.Position, msg string, args ...interface{}) {
	if trcErrors {
		trc("FAIL "+msg, args...)
	}
	switch {
	case len(args) == 0:
		*e = append(*e, ErrWithPosition{pos, fmt.Errorf("%s", msg)})
	default:
		*e = append(*e, ErrWithPosition{pos, fmt.Errorf(msg, args...)})
	}
}

type scanner struct {
	*source
	dir  string
	errs errList
	tok  tok

	last int32

	errBudget int

	c byte // Lookahead byte.

	eof      bool
	isClosed bool
}

func newScanner(name string, buf []byte) *scanner {
	dir, _ := filepath.Split(name)
	r := &scanner{source: newSource(name, buf), errBudget: 10, dir: dir}
	switch {
	case len(buf) == 0:
		r.eof = true
	default:
		r.c = buf[0]
		if r.c == '\n' {
			r.file.AddLine(int(r.base + r.off))
		}
	}
	return r
}

func isDigit(c byte) bool      { return c >= '0' && c <= '9' }
func isHexDigit(c byte) bool   { return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' }
func isIDNext(c byte) bool     { return isIDFirst(c) || isDigit(c) }
func isOctalDigit(c byte) bool { return c >= '0' && c <= '7' }

func isIDFirst(c byte) bool {
	return c >= 'a' && c <= 'z' ||
		c >= 'A' && c <= 'Z' ||
		c == '_'
}

func (s *scanner) position() token.Position {
	return token.Position(s.source.file.PositionFor(mtoken.Pos(s.base+s.off), true))
}

func (s *scanner) pos(off int32) token.Position {
	return token.Position(s.file.PositionFor(mtoken.Pos(s.base+off), true))
}

func (s *scanner) token() Token {
	return Token{source: s.source, ch: s.tok.ch, index: int32(len(s.toks) - 1)}
}

func (s *scanner) err(off int32, msg string, args ...interface{}) {
	if s.errBudget <= 0 {
		s.close()
		return
	}

	s.errBudget--
	if n := int32(len(s.buf)); off >= n {
		off = n
	}
	s.errs.err(s.pos(off), msg, args...)
}

func (s *scanner) close() {
	if s.isClosed {
		return
	}

	s.tok.ch = int32(ILLEGAL)
	s.eof = true
	s.isClosed = true
}

func (s *scanner) next() {
	if s.eof {
		return
	}

	s.off++
	if int(s.off) == len(s.buf) {
		s.c = 0
		s.eof = true
		return
	}

	s.c = s.buf[s.off]
	if s.c == '\n' {
		s.file.AddLine(int(s.base + s.off))
	}
}

func (s *scanner) nextN(n int) {
	if int(s.off) == len(s.buf)-n {
		s.c = 0
		s.eof = true
		return
	}

	s.off += int32(n)
	s.c = s.buf[s.off]
	if s.c == '\n' {
		s.file.AddLine(int(s.base + s.off))
	}
}

func (s *scanner) scan() (r bool) {
	if s.isClosed {
		return false
	}

	s.last = s.tok.ch
	s.tok.sep = s.off
	s.tok.ch = -1
	for {
		if r = s.scan0(); !r || s.tok.ch >= 0 {
			s.toks = append(s.toks, s.tok)
			// trc("", dump(s.token()))
			return r
		}
	}
}

func (s *scanner) scan0() (r bool) {
	s.tok.src = mathutil.MinInt32(s.off, int32(len(s.buf)))
	switch s.c {
	case ' ', '\t', '\r', '\n':
		// White space, formed from spaces (U+0020), horizontal tabs (U+0009), carriage
		// returns (U+000D), and newlines (U+000A), is ignored except as it separates
		// tokens that would otherwise combine into a single token.
		if s.c == '\n' && s.injectSemi() {
			return true
		}

		s.next()
		return true
	case '/':
		off := s.off
		s.next()
		switch s.c {
		case '=':
			s.next()
			s.tok.ch = int32(QUO_ASSIGN)
		case '/':
			// Line comments start with the character sequence // and stop at the end of
			// the line.
			s.next()
			s.lineComment(off)
			return true
		case '*':
			// General comments start with the character sequence /* and stop with the
			// first subsequent character sequence */.
			s.next()
			s.generalComment(off)
			return true
		default:
			s.tok.ch = int32(QUO)
		}
	case '(':
		s.tok.ch = int32(LPAREN)
		s.next()
	case ')':
		s.tok.ch = int32(RPAREN)
		s.next()
	case '[':
		s.tok.ch = int32(LBRACK)
		s.next()
	case ']':
		s.tok.ch = int32(RBRACK)
		s.next()
	case '{':
		s.tok.ch = int32(LBRACE)
		s.next()
	case '}':
		s.tok.ch = int32(RBRACE)
		s.next()
	case ',':
		s.tok.ch = int32(COMMA)
		s.next()
	case ';':
		s.tok.ch = int32(SEMICOLON)
		s.next()
	case '~':
		s.tok.ch = int32(TILDE)
		s.next()
	case '"':
		off := s.off
		s.next()
		s.stringLiteral(off)
	case '\'':
		off := s.off
		s.next()
		s.runeLiteral(off)
	case '`':
		s.next()
		for {
			switch {
			case s.c == '`':
				s.next()
				s.tok.ch = int32(STRING)
				return true
			case s.eof:
				s.err(s.off, "raw string literal not terminated")
				s.tok.ch = int32(STRING)
				return true
			case s.c == 0:
				panic(todo("%v: %#U", s.position(), s.c))
			default:
				s.next()
			}
		}
	case '.':
		s.next()
		off := s.off
		if isDigit(s.c) {
			s.dot(false, true)
			return true
		}

		if s.c != '.' {
			s.tok.ch = int32(PERIOD)
			return true
		}

		s.next()
		if s.c != '.' {
			s.off = off
			s.c = '.'
			s.tok.ch = int32(PERIOD)
			return true
		}

		s.next()
		s.tok.ch = int32(ELLIPSIS)
		return true
	case '%':
		s.next()
		switch s.c {
		case '=':
			s.next()
			s.tok.ch = int32(REM_ASSIGN)
		default:
			s.tok.ch = int32(REM)
		}
	case '*':
		s.next()
		switch s.c {
		case '=':
			s.next()
			s.tok.ch = int32(MUL_ASSIGN)
		default:
			s.tok.ch = int32(MUL)
		}
	case '^':
		s.next()
		switch s.c {
		case '=':
			s.next()
			s.tok.ch = int32(XOR_ASSIGN)
		default:
			s.tok.ch = int32(XOR)
		}
	case '+':
		s.next()
		switch s.c {
		case '+':
			s.next()
			s.tok.ch = int32(INC)
		case '=':
			s.next()
			s.tok.ch = int32(ADD_ASSIGN)
		default:
			s.tok.ch = int32(ADD)
		}
	case '-':
		s.next()
		switch s.c {
		case '-':
			s.next()
			s.tok.ch = int32(DEC)
		case '=':
			s.next()
			s.tok.ch = int32(SUB_ASSIGN)
		default:
			s.tok.ch = int32(SUB)
		}
	case ':':
		s.next()
		switch {
		case s.c == '=':
			s.next()
			s.tok.ch = int32(DEFINE)
		default:
			s.tok.ch = int32(COLON)
		}
	case '=':
		s.next()
		switch {
		case s.c == '=':
			s.next()
			s.tok.ch = int32(EQL)
		default:
			s.tok.ch = int32(ASSIGN)
		}
	case '!':
		s.next()
		switch {
		case s.c == '=':
			s.next()
			s.tok.ch = int32(NEQ)
		default:
			s.tok.ch = int32(NOT)
		}
	case '>':
		s.next()
		switch s.c {
		case '=':
			s.next()
			s.tok.ch = int32(GEQ)
		case '>':
			s.next()
			switch s.c {
			case '=':
				s.next()
				s.tok.ch = int32(SHR_ASSIGN)
			default:
				s.tok.ch = int32(SHR)
			}
		default:
			s.tok.ch = int32(GTR)
		}
	case '<':
		s.next()
		switch s.c {
		case '=':
			s.next()
			s.tok.ch = int32(LEQ)
		case '<':
			s.next()
			switch s.c {
			case '=':
				s.next()
				s.tok.ch = int32(SHL_ASSIGN)
			default:
				s.tok.ch = int32(SHL)
			}
		case '-':
			s.next()
			s.tok.ch = int32(ARROW)
		default:
			s.tok.ch = int32(LSS)
		}
	case '|':
		s.next()
		switch s.c {
		case '|':
			s.next()
			s.tok.ch = int32(LOR)
		case '=':
			s.next()
			s.tok.ch = int32(OR_ASSIGN)
		default:
			s.tok.ch = int32(OR)
		}
	case '&':
		s.next()
		switch s.c {
		case '&':
			s.next()
			s.tok.ch = int32(LAND)
		case '^':
			s.next()
			switch s.c {
			case '=':
				s.next()
				s.tok.ch = int32(AND_NOT_ASSIGN)
			default:
				s.tok.ch = int32(AND_NOT)
			}
		case '=':
			s.next()
			s.tok.ch = int32(AND_ASSIGN)
		default:
			s.tok.ch = int32(AND)
		}
	default:
		switch {
		case isIDFirst(s.c):
			s.next()
			s.identifierOrKeyword()
		case isDigit(s.c):
			s.numericLiteral()
		case s.c >= 0x80:
			off := s.off
			switch r := s.rune(); {
			case unicode.IsLetter(r):
				s.identifierOrKeyword()
			case r == 0xfeff:
				if off == 0 { // Ignore BOM, but only at buffer start.
					return true
				}

				s.err(off, "illegal byte order mark")
				s.tok.ch = int32(ILLEGAL)
			default:
				s.err(s.off, "illegal character %#U", r)
				s.tok.ch = int32(ILLEGAL)
			}
		case s.eof:
			if s.injectSemi() {
				return true
			}

			s.close()
			s.tok.ch = int32(EOF)
			s.tok.sep = mathutil.MinInt32(s.tok.sep, s.tok.src)
			return false
		// case s.c == 0:
		// 	panic(todo("%v: %#U", s.position(), s.c))
		default:
			s.err(s.off, "illegal character %#U", s.c)
			s.next()
			s.tok.ch = int32(ILLEGAL)
		}
	}
	return true
}

func (s *scanner) runeLiteral(off int32) {
	// Leading ' consumed.
	ok := 0
	s.tok.ch = int32(CHAR)
	expOff := int32(-1)
	if s.eof {
		s.err(off, "rune literal not terminated")
		return
	}

	for {
		switch s.c {
		case '\\':
			ok++
			s.next()
			switch s.c {
			case '\'', '\\', 'a', 'b', 'f', 'n', 'r', 't', 'v':
				s.next()
			case 'x', 'X':
				s.next()
				for i := 0; i < 2; i++ {
					if s.c == '\'' {
						if i != 2 {
							s.err(s.off, "illegal character %#U in escape sequence", s.c)
						}
						s.next()
						return
					}

					if !isHexDigit(s.c) {
						s.err(s.off, "illegal character %#U in escape sequence", s.c)
						break
					}
					s.next()
				}
			case 'u':
				s.u(4)
			case 'U':
				s.u(8)
			default:
				switch {
				case s.eof:
					s.err(s.base+s.off, "escape sequence not terminated")
					return
				case isOctalDigit(s.c):
					for i := 0; i < 3; i++ {
						s.next()
						if s.c == '\'' {
							if i != 2 {
								s.err(s.off, "illegal character %#U in escape sequence", s.c)
							}
							s.next()
							return
						}

						if !isOctalDigit(s.c) {
							s.err(s.off, "illegal character %#U in escape sequence", s.c)
							break
						}
					}
				default:
					s.err(s.off, "unknown escape sequence")
				}
			}
		case '\'':
			s.next()
			if ok != 1 {
				s.err(off, "illegal rune literal")
			}
			return
		case '\t':
			s.next()
			ok++
		default:
			switch {
			case s.eof:
				switch {
				case ok != 0:
					s.err(expOff, "rune literal not terminated")
				default:
					s.err(s.base+s.off, "rune literal not terminated")
				}
				return
			case s.c == 0:
				panic(todo("%v: %#U", s.position(), s.c))
			case s.c < ' ':
				ok++
				s.err(s.off, "non-printable character: %#U", s.c)
				s.next()
			case s.c >= 0x80:
				ok++
				off := s.off
				if c := s.rune(); c == 0xfeff {
					s.err(off, "illegal byte order mark")
				}
			default:
				ok++
				s.next()
			}
		}
		if ok != 0 && expOff < 0 {
			expOff = s.off
			if s.eof {
				expOff++
			}
		}
	}
}

func (s *scanner) stringLiteral(off int32) {
	// Leadind " consumed.
	s.tok.ch = int32(STRING)
	for {
		switch {
		case s.c == '"':
			s.next()
			return
		case s.c == '\\':
			s.next()
			switch s.c {
			case '"', '\\', 'a', 'b', 'f', 'n', 'r', 't', 'v':
				s.next()
				continue
			case 'x', 'X':
				s.next()
				if !isHexDigit(s.c) {
					panic(todo("%v: %#U", s.position(), s.c))
				}

				s.next()
				if !isHexDigit(s.c) {
					panic(todo("%v: %#U", s.position(), s.c))
				}

				s.next()
				continue
			case 'u':
				s.u(4)
				continue
			case 'U':
				s.u(8)
				continue
			default:
				switch {
				case isOctalDigit(s.c):
					s.next()
					if isOctalDigit(s.c) {
						s.next()
					}
					if isOctalDigit(s.c) {
						s.next()
					}
					continue
				default:
					s.err(off-1, "unknown escape sequence")
				}
			}
		case s.c == '\n':
			fallthrough
		case s.eof:
			s.err(off, "string literal not terminated")
			return
		case s.c == 0:
			s.err(s.off, "illegal character NUL")
		}

		switch {
		case s.c >= 0x80:
			off := s.off
			if s.rune() == 0xfeff {
				s.err(off, "illegal byte order mark")
			}
			continue
		}

		s.next()
	}
}

func (s *scanner) u(n int) (r rune) {
	// Leading u/U not consumed.
	s.next()
	off := s.off
	for i := 0; i < n; i++ {
		switch {
		case isHexDigit(s.c):
			var n rune
			switch {
			case s.c >= '0' && s.c <= '9':
				n = rune(s.c) - '0'
			case s.c >= 'a' && s.c <= 'f':
				n = rune(s.c) - 'a' + 10
			case s.c >= 'A' && s.c <= 'F':
				n = rune(s.c) - 'A' + 10
			}
			r = 16*r + n
		default:
			switch {
			case s.eof:
				s.err(s.base+s.off, "escape sequence not terminated")
			default:
				s.err(s.off, "illegal character %#U in escape sequence", s.c)
			}
			return r
		}

		s.next()
	}
	if r < 0 || r > unicode.MaxRune || r >= 0xd800 && r <= 0xdfff {
		s.err(off-1, "escape sequence is invalid Unicode code point")
	}
	return r
}

func (s *scanner) identifierOrKeyword() {
out:
	for {
		switch {
		case isIDNext(s.c):
			s.next()
		case s.c >= 0x80:
			off := s.off
			c := s.c
			switch r := s.rune(); {
			case unicode.IsLetter(r) || unicode.IsDigit(r):
				// already consumed
			default:
				s.off = off
				s.c = c
				break out
			}
		case s.eof:
			break out
		case s.c == 0:
			s.err(s.off, "illegal character NUL")
			break out
		default:
			break out
		}
	}
	if s.tok.ch = int32(keywords[string(s.buf[s.tok.src:s.off])]); s.tok.ch == 0 {
		s.tok.ch = int32(IDENT)
	}
}

func (s *scanner) numericLiteral() {
	// Leading decimal digit not consumed.
	var hasHexMantissa, needFrac bool
more:
	switch s.c {
	case '0':
		s.next()
		switch s.c {
		case '.':
			// nop
		case 'b', 'B':
			s.next()
			s.binaryLiteral()
			return
		case 'e', 'E':
			s.exponent()
			s.tok.ch = int32(FLOAT)
			return
		case 'p', 'P':
			s.err(s.off, "'%c' exponent requires hexadecimal mantissa", s.c)
			s.exponent()
			s.tok.ch = int32(FLOAT)
			return
		case 'o', 'O':
			s.next()
			s.octalLiteral()
			return
		case 'x', 'X':
			hasHexMantissa = true
			needFrac = true
			s.tok.ch = int32(INT)
			s.next()
			if s.c == '.' {
				s.next()
				s.dot(hasHexMantissa, needFrac)
				return
			}

			if s.hexadecimals() == 0 {
				s.err(s.base+s.off, "hexadecimal literal has no digits")
				return
			}

			needFrac = false
		case 'i':
			s.next()
			s.tok.ch = int32(IMAG)
			return
		default:
			invalidOff := int32(-1)
			var invalidDigit byte
			for {
				if s.c == '_' {
					for n := 0; s.c == '_'; n++ {
						if n == 1 {
							s.err(s.off, "'_' must separate successive digits")
						}
						s.next()
					}
					if !isDigit(s.c) {
						s.err(s.off-1, "'_' must separate successive digits")
					}
				}
				if isOctalDigit(s.c) {
					s.next()
					continue
				}

				if isDigit(s.c) {
					if invalidOff < 0 {
						invalidOff = s.off
						invalidDigit = s.c
					}
					s.next()
					continue
				}

				break
			}
			switch s.c {
			case '.', 'e', 'E', 'i':
				break more
			}
			if isDigit(s.c) {
				break more
			}
			if invalidOff > 0 {
				s.err(invalidOff, "invalid digit '%c' in octal literal", invalidDigit)
			}
			s.tok.ch = int32(INT)
			return
		}
	default:
		s.decimals()
	}
	switch s.c {
	case '.':
		s.next()
		s.dot(hasHexMantissa, needFrac)
	case 'p', 'P':
		if !hasHexMantissa {
			s.err(s.off, "'%c' exponent requires hexadecimal mantissa", s.c)
		}
		fallthrough
	case 'e', 'E':
		s.exponent()
		if s.c == 'i' {
			s.next()
			s.tok.ch = int32(IMAG)
			return
		}

		s.tok.ch = int32(FLOAT)
	case 'i':
		s.next()
		s.tok.ch = int32(IMAG)
	default:
		s.tok.ch = int32(INT)
	}
}

func (s *scanner) octalLiteral() {
	// Leading 0o consumed.
	ok := false
	invalidOff := int32(-1)
	var invalidDigit byte
	s.tok.ch = int32(INT)
	for {
		for n := 0; s.c == '_'; n++ {
			if n == 1 {
				s.err(s.off, "'_' must separate successive digits")
			}
			s.next()
		}
		switch s.c {
		case '0', '1', '2', '3', '4', '5', '6', '7':
			s.next()
			ok = true
		case '8', '9':
			if invalidOff < 0 {
				invalidOff = s.off
				invalidDigit = s.c
			}
			s.next()
		case '.':
			s.tok.ch = int32(FLOAT)
			s.err(s.off, "invalid radix point in octal literal")
			s.next()
		case 'e', 'E':
			s.tok.ch = int32(FLOAT)
			s.err(s.off, "'%c' exponent requires decimal mantissa", s.c)
			s.exponent()
		case 'p', 'P':
			s.tok.ch = int32(FLOAT)
			s.err(s.off, "'%c' exponent requires hexadecimal mantissa", s.c)
			s.exponent()
		default:
			switch {
			case !ok:
				s.err(s.base+s.off, "octal literal has no digits")
			case invalidOff > 0:
				s.err(invalidOff, "invalid digit '%c' in octal literal", invalidDigit)
			}
			if s.c == 'i' {
				s.next()
				s.tok.ch = int32(IMAG)
			}
			return
		}
	}
}

func (s *scanner) binaryLiteral() {
	// Leading 0b consumed.
	ok := false
	invalidOff := int32(-1)
	var invalidDigit byte
	s.tok.ch = int32(INT)
	for {
		for n := 0; s.c == '_'; n++ {
			if n == 1 {
				s.err(s.off, "'_' must separate successive digits")
			}
			s.next()
		}
		switch s.c {
		case '0', '1':
			s.next()
			ok = true
		case '.':
			s.tok.ch = int32(FLOAT)
			s.err(s.off, "invalid radix point in binary literal")
			s.next()
		case 'e', 'E':
			s.tok.ch = int32(FLOAT)
			s.err(s.off, "'%c' exponent requires decimal mantissa", s.c)
			s.exponent()
		case 'p', 'P':
			s.tok.ch = int32(FLOAT)
			s.err(s.off, "'%c' exponent requires hexadecimal mantissa", s.c)
			s.exponent()
		default:
			if isDigit(s.c) {
				if invalidOff < 0 {
					invalidOff = s.off
					invalidDigit = s.c
				}
				s.next()
				continue
			}

			switch {
			case !ok:
				s.err(s.base+s.off, "binary literal has no digits")
			case invalidOff > 0:
				s.err(invalidOff, "invalid digit '%c' in binary literal", invalidDigit)
			}
			if s.c == 'i' {
				s.next()
				s.tok.ch = int32(IMAG)
			}
			return
		}
	}
}

func (s *scanner) generalComment(off int32) (injectSemi bool) {
	// Leading /* consumed
	off0 := s.off - 2
	var nl bool
	for {
		switch {
		case s.c == '*':
			s.next()
			switch s.c {
			case '/':
				s.lineInfo(off0, s.off+1)
				s.next()
				if nl {
					return s.injectSemi()
				}

				return false
			}
		case s.c == '\n':
			nl = true
			s.next()
		case s.eof:
			s.tok.ch = 0
			s.err(off, "comment not terminated")
			return true
		case s.c == 0:
			panic(todo("%v: %#U", s.position(), s.c))
		default:
			s.next()
		}
	}
}

func (s *scanner) lineComment(off int32) (injectSemi bool) {
	// Leading // consumed
	off0 := s.off - 2
	for {
		switch {
		case s.c == '\n':
			s.lineInfo(off0, s.off+1)
			if s.injectSemi() {
				return true
			}

			s.next()
			return false
		case s.c >= 0x80:
			if c := s.rune(); c == 0xfeff {
				s.err(off+2, "illegal byte order mark")
			}
		case s.eof:
			s.off++
			if s.injectSemi() {
				return true
			}

			return false
		case s.c == 0:
			return false
		default:
			s.next()
		}
	}
}

func (s *scanner) lineInfo(off, next int32) {
	if off != 0 && s.buf[off+1] != '*' && s.buf[off-1] != '\n' && s.buf[off-1] != '\r' {
		return
	}

	str := s.buf[off:next]
	if !bytes.HasPrefix(str[len("//"):], lineCommentTag) {
		return
	}

	switch {
	case str[1] == '*':
		str = str[:len(str)-len("*/")]
	default:
		str = str[:len(str)-len("\n")]
	}
	str = str[len("//"):]

	str, ln, ok := s.lineInfoNum(str[len("line "):])
	col := 0
	if ok == liBadNum || ok == liNoNum {
		return
	}

	hasCol := false
	var n int
	if str, n, ok = s.lineInfoNum(str); ok == liBadNum {
		return
	}

	if ok != liNoNum {
		col = ln
		ln = n
		hasCol = true
	}

	fn := strings.TrimSpace(string(str))
	switch {
	case fn == "" && hasCol:
		fn = s.pos(off).Filename
	case fn != "":
		fn = filepath.Clean(fn)
		if !filepath.IsAbs(fn) {
			fn = filepath.Join(s.dir, fn)
		}
	}
	// trc("set %v %q %v %v", next, fn, ln, col)
	s.file.AddLineColumnInfo(int(next), fn, ln, col)
}

const (
	liNoNum = iota
	liBadNum
	liOK
)

func (s *scanner) lineInfoNum(str []byte) (_ []byte, n, r int) {
	// trc("==== %q", str)
	x := len(str) - 1
	if x < 0 || !isDigit(str[x]) {
		return str, 0, liNoNum
	}

	mul := 1
	for x > 0 && isDigit(str[x]) {
		n += mul * (int(str[x]) - '0')
		mul *= 10
		x--
		if n < 0 {
			return str, 0, liBadNum
		}
	}
	if x < 0 || str[x] != ':' {
		return str, 0, liBadNum
	}

	// trc("---- %q %v %v", str[:x], n, liOK)
	return str[:x], n, liOK
}

func (s *scanner) rune() rune {
	switch r, sz := utf8.DecodeRune(s.buf[s.off:]); {
	case r == utf8.RuneError && sz == 0:
		panic(todo("%v: %#U", s.position(), s.c))
	case r == utf8.RuneError && sz == 1:
		s.err(s.off, "illegal UTF-8 encoding")
		s.next()
		return r
	default:
		s.nextN(sz)
		return r
	}
}

func (s *scanner) dot(hasHexMantissa, needFrac bool) {
	// '.' already consumed
	switch {
	case hasHexMantissa:
		if s.hexadecimals() == 0 && needFrac {
			s.err(s.off, "hexadecimal literal has no digits")
		}
		switch s.c {
		case 'p', 'P':
			// ok
		default:
			s.err(s.off, "hexadecimal mantissa requires a 'p' exponent")
		}
	default:
		if s.decimals() == 0 && needFrac {
			panic(todo("%v: %#U", s.position(), s.c))
		}
	}
	switch s.c {
	case 'p', 'P':
		if !hasHexMantissa {
			s.err(s.off, "'%c' exponent requires hexadecimal mantissa", s.c)
		}
		fallthrough
	case 'e', 'E':
		s.exponent()
		if s.c == 'i' {
			s.next()
			s.tok.ch = int32(IMAG)
			return
		}

		s.tok.ch = int32(FLOAT)
	case 'i':
		s.next()
		s.tok.ch = int32(IMAG)
	default:
		s.tok.ch = int32(FLOAT)
	}
}

func (s *scanner) exponent() {
	// Leanding e or E not consumed.
	s.next()
	switch s.c {
	case '+', '-':
		s.next()
	}
	if !isDigit(s.c) {
		s.err(s.base+s.off, "exponent has no digits")
		return
	}

	s.decimals()
}

func (s *scanner) decimals() (r int) {
	first := true
	for {
		switch {
		case isDigit(s.c):
			first = false
			s.next()
			r++
		case s.c == '_':
			for n := 0; s.c == '_'; n++ {
				if first || n == 1 {
					s.err(s.off, "'_' must separate successive digits")
				}
				s.next()
			}
			if !isDigit(s.c) {
				s.err(s.off-1, "'_' must separate successive digits")
			}
		default:
			return r
		}
	}
}

func (s *scanner) hexadecimals() (r int) {
	for {
		switch {
		case isHexDigit(s.c):
			s.next()
			r++
		case s.c == '_':
			for n := 0; s.c == '_'; n++ {
				if n == 1 {
					s.err(s.off, "'_' must separate successive digits")
				}
				s.next()
			}
			if !isHexDigit(s.c) {
				s.err(s.off-1, "'_' must separate successive digits")
			}
		default:
			return r
		}
	}
}

// When the input is broken into tokens, a semicolon is automatically inserted
// into the token stream immediately after a line's final token if that token
// is
//
//   - an identifier
//   - an integer, floating-point, imaginary, rune, or string literal
//   - one of the keywords break, continue, fallthrough, or return
//   - one of the operators and punctuation ++, --, ), ], or }
func (s *scanner) injectSemi() bool {
	switch token.Token(s.last) {
	case
		IDENT, INT, FLOAT, IMAG, CHAR, STRING,
		BREAK, CONTINUE, FALLTHROUGH, RETURN,
		INC, DEC, RPAREN, RBRACK, RBRACE:

		s.tok.ch = int32(SEMICOLON)
		s.last = 0
		if s.c == '\n' {
			s.next()
		}
		return true
	}

	s.last = 0
	return false
}
