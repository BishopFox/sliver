// Package inputrc parses readline inputrc files.
package inputrc

//go:generate go run gen.go

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

// Parse parses inputrc data from r.
func Parse(r io.Reader, h Handler, opts ...Option) error {
	return New(opts...).Parse(r, h)
}

// ParseBytes parses inputrc data from buf.
func ParseBytes(buf []byte, h Handler, opts ...Option) error {
	return New(opts...).Parse(bytes.NewReader(buf), h)
}

// ParseFile parses inputrc data from a file name.
func ParseFile(name string, h Handler, opts ...Option) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}

	defer f.Close()

	return New(append(opts, WithName(name))...).Parse(f, h)
}

// UserDefault loads default inputrc settings for the user.
func UserDefault(u *user.User, cfg *Config, opts ...Option) error {
	// build possible file list
	var files []string
	if name := os.Getenv("INPUTRC"); name != "" {
		files = append(files, name)
	}

	if u != nil {
		name := ".inputrc"
		if runtime.GOOS == "windows" {
			name = "_inputrc"
		}

		files = append(files, filepath.Join(u.HomeDir, name))
	}

	if runtime.GOOS != "windows" {
		files = append(files, "/etc/inputrc")
	}
	// load first available file
	for _, name := range files {
		buf, err := cfg.ReadFile(name)

		switch {
		case err != nil && errors.Is(err, os.ErrNotExist):
			continue
		case err != nil:
			return err
		}

		return ParseBytes(buf, cfg, append(opts, WithName(name))...)
	}

	return nil
}

// Unescape unescapes a inputrc string.
func Unescape(s string) string {
	r := []rune(s)
	return unescapeRunes(r, 0, len(r))
}

// Escape escapes a inputrc string.
func Escape(s string) string {
	return escape(s, map[rune]string{
		Delete: `\C-?`,
		Return: `\C-M`,
	})
}

// EscapeMacro escapes a inputrc macro.
func EscapeMacro(s string) string {
	return escape(s, map[rune]string{
		Delete: `\d`,
		Return: `\r`,
	})
}

// escape escapes s using m.
func escape(s string, m map[rune]string) string {
	var v []string

	for _, c := range s {
		switch c {
		case Alert:
			v = append(v, `\a`)
		case Backspace:
			v = append(v, `\b`)
		case Delete:
			v = append(v, m[Delete]) // \C-? or \d
		case Esc:
			v = append(v, `\e`)
		case Formfeed:
			v = append(v, `\f`)
		case Newline:
			v = append(v, `\n`)
		case Return:
			v = append(v, m[Return]) // \C-M or \r
		case Tab:
			v = append(v, `\t`)
		case Vertical:
			v = append(v, `\v`)
		case '\\', '"', '\'':
			v = append(v, `\`+string(c))
		default:
			var s string
			if IsControl(c) {
				s += `\C-`
				c = Decontrol(c)
			}

			if IsMeta(c) {
				s += `\M-`
				c = Demeta(c)
			}

			if unicode.IsPrint(c) {
				s += string(c)
			} else {
				s += fmt.Sprintf(`\x%2x`, c)
			}

			v = append(v, s)
		}
	}

	return strings.Join(v, "")
}

// Encontrol encodes a Control-c code.
func Encontrol(c rune) rune {
	return unicode.ToUpper(c) & Control
}

// Decontrol decodes a Control-c code.
func Decontrol(c rune) rune {
	return unicode.ToUpper(c | 0x40)
}

// IsControl returns true when c is a Control-c code.
func IsControl(c rune) bool {
	return c < Space && c&Meta == 0
}

// Enmeta encodes a Meta-c code.
func Enmeta(c rune) rune {
	return c | Meta
}

// Demeta decodes a Meta-c code.
func Demeta(c rune) rune {
	return c & ^Meta
}

// IsMeta returns true when c is a Meta-c code.
func IsMeta(c rune) bool {
	return c > Delete && c <= 0xff
}

// Error is a error.
type Error string

// Errors.
const (
	// ErrBindMissingClosingQuote is the bind missing closing quote error.
	ErrBindMissingClosingQuote Error = `bind missing closing quote`
	// ErrMissingColon is the missing : error.
	ErrMissingColon Error = "missing :"
	// ErrMacroMissingClosingQuote is the macro missing closing quote error.
	ErrMacroMissingClosingQuote Error = `macro missing closing quote`
	// ErrInvalidKeymap is the invalid keymap error.
	ErrInvalidKeymap Error = "invalid keymap"
	// ErrInvalidEditingMode is the invalid editing mode error.
	ErrInvalidEditingMode Error = "invalid editing mode"
	// ErrElseWithoutMatchingIf is the $else without matching $if error.
	ErrElseWithoutMatchingIf Error = "$else without matching $if"
	// ErrEndifWithoutMatchingIf is the $endif without matching $if error.
	ErrEndifWithoutMatchingIf Error = "$endif without matching $if"
	// ErrUnknownModifier is the unknown modifier error.
	ErrUnknownModifier Error = "unknown modifier"
)

// Error satisfies the error interface.
func (err Error) Error() string {
	return string(err)
}

// Keys.
const (
	Control   rune = 0x1f
	Meta      rune = 0x80
	Esc       rune = 0x1b
	Delete    rune = 0x7f
	Alert     rune = '\a'
	Backspace rune = '\b'
	Formfeed  rune = '\f'
	Newline   rune = '\n'
	Return    rune = '\r'
	Tab       rune = '\t'
	Vertical  rune = '\v'
	Space     rune = ' '
	// Rubout = Delete.
)
