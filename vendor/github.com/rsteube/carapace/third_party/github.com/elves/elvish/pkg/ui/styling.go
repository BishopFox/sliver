package ui

import (
	"strings"
)

// Styling specifies how to change a Style. It can also be applied to a Segment
// or Text.
type Styling interface{ transform(*Style) }

// ApplyStyling returns a new Style with the given Styling's applied.
func ApplyStyling(s Style, ts ...Styling) Style {
	for _, t := range ts {
		if t != nil {
			t.transform(&s)
		}
	}
	return s
}

// Stylings joins several transformers into one.
func Stylings(ts ...Styling) Styling { return jointStyling(ts) }

// Common stylings.
var (
	Reset Styling = reset{}

	FgDefault Styling = setForeground{nil}

	FgBlack   Styling = setForeground{Black}
	FgRed     Styling = setForeground{Red}
	FgGreen   Styling = setForeground{Green}
	FgYellow  Styling = setForeground{Yellow}
	FgBlue    Styling = setForeground{Blue}
	FgMagenta Styling = setForeground{Magenta}
	FgCyan    Styling = setForeground{Cyan}
	FgWhite   Styling = setForeground{White}

	FgBrightBlack   Styling = setForeground{BrightBlack}
	FgBrightRed     Styling = setForeground{BrightRed}
	FgBrightGreen   Styling = setForeground{BrightGreen}
	FgBrightYellow  Styling = setForeground{BrightYellow}
	FgBrightBlue    Styling = setForeground{BrightBlue}
	FgBrightMagenta Styling = setForeground{BrightMagenta}
	FgBrightCyan    Styling = setForeground{BrightCyan}
	FgBrightWhite   Styling = setForeground{BrightWhite}

	BgDefault Styling = setBackground{nil}

	BgBlack   Styling = setBackground{Black}
	BgRed     Styling = setBackground{Red}
	BgGreen   Styling = setBackground{Green}
	BgYellow  Styling = setBackground{Yellow}
	BgBlue    Styling = setBackground{Blue}
	BgMagenta Styling = setBackground{Magenta}
	BgCyan    Styling = setBackground{Cyan}
	BgWhite   Styling = setBackground{White}

	BgBrightBlack   Styling = setBackground{BrightBlack}
	BgBrightRed     Styling = setBackground{BrightRed}
	BgBrightGreen   Styling = setBackground{BrightGreen}
	BgBrightYellow  Styling = setBackground{BrightYellow}
	BgBrightBlue    Styling = setBackground{BrightBlue}
	BgBrightMagenta Styling = setBackground{BrightMagenta}
	BgBrightCyan    Styling = setBackground{BrightCyan}
	BgBrightWhite   Styling = setBackground{BrightWhite}

	Bold       Styling = boolOn{boldField{}}
	Dim        Styling = boolOn{dimField{}}
	Italic     Styling = boolOn{italicField{}}
	Underlined Styling = boolOn{underlinedField{}}
	Blink      Styling = boolOn{blinkField{}}
	Inverse    Styling = boolOn{inverseField{}}

	NoBold       Styling = boolOff{boldField{}}
	NoDim        Styling = boolOff{dimField{}}
	NoItalic     Styling = boolOff{italicField{}}
	NoUnderlined Styling = boolOff{underlinedField{}}
	NoBlink      Styling = boolOff{blinkField{}}
	NoInverse    Styling = boolOff{inverseField{}}

	ToggleBold       Styling = boolToggle{boldField{}}
	ToggleDim        Styling = boolToggle{dimField{}}
	ToggleItalic     Styling = boolToggle{italicField{}}
	ToggleUnderlined Styling = boolToggle{underlinedField{}}
	ToggleBlink      Styling = boolToggle{blinkField{}}
	ToggleInverse    Styling = boolToggle{inverseField{}}
)

// Fg returns a Styling that sets the foreground color.
func Fg(c Color) Styling { return setForeground{c} }

// Bg returns a Styling that sets the background color.
func Bg(c Color) Styling { return setBackground{c} }

type reset struct{}
type setForeground struct{ c Color }
type setBackground struct{ c Color }
type boolOn struct{ f boolField }
type boolOff struct{ f boolField }
type boolToggle struct{ f boolField }

func (reset) transform(s *Style)           { *s = Style{} }
func (t setForeground) transform(s *Style) { s.Foreground = t.c }
func (t setBackground) transform(s *Style) { s.Background = t.c }
func (t boolOn) transform(s *Style)        { *t.f.get(s) = true }
func (t boolOff) transform(s *Style)       { *t.f.get(s) = false }
func (t boolToggle) transform(s *Style)    { p := t.f.get(s); *p = !*p }

type boolField interface{ get(*Style) *bool }

type boldField struct{}
type dimField struct{}
type italicField struct{}
type underlinedField struct{}
type blinkField struct{}
type inverseField struct{}

func (boldField) get(s *Style) *bool       { return &s.Bold }
func (dimField) get(s *Style) *bool        { return &s.Dim }
func (italicField) get(s *Style) *bool     { return &s.Italic }
func (underlinedField) get(s *Style) *bool { return &s.Underlined }
func (blinkField) get(s *Style) *bool      { return &s.Blink }
func (inverseField) get(s *Style) *bool    { return &s.Inverse }

type jointStyling []Styling

func (t jointStyling) transform(s *Style) {
	for _, t := range t {
		t.transform(s)
	}
}

// ParseStyling parses a text representation of Styling, which are kebab
// case counterparts to the names of the builtin Styling's. For example,
// ToggleInverse is expressed as "toggle-inverse".
//
// Multiple stylings can be joined by spaces, which is equivalent to calling
// Stylings.
//
// If the given string is invalid, ParseStyling returns nil.
func ParseStyling(s string) Styling {
	if !strings.ContainsRune(s, ' ') {
		return parseOneStyling(s)
	}
	var joint jointStyling
	for _, subs := range strings.Split(s, " ") {
		parsed := parseOneStyling(subs)
		if parsed == nil {
			return nil
		}
		joint = append(joint, parseOneStyling(subs))
	}
	return joint
}

var boolFields = map[string]boolField{
	"bold":       boldField{},
	"dim":        dimField{},
	"italic":     italicField{},
	"underlined": underlinedField{},
	"blink":      blinkField{},
	"inverse":    inverseField{},
}

func parseOneStyling(name string) Styling {
	switch {
	case name == "default" || name == "fg-default":
		return FgDefault
	case strings.HasPrefix(name, "fg-"):
		if color := parseColor(name[len("fg-"):]); color != nil {
			return setForeground{color}
		}
	case name == "bg-default":
		return BgDefault
	case strings.HasPrefix(name, "bg-"):
		if color := parseColor(name[len("bg-"):]); color != nil {
			return setBackground{color}
		}
	case strings.HasPrefix(name, "no-"):
		if f, ok := boolFields[name[len("no-"):]]; ok {
			return boolOff{f}
		}
	case strings.HasPrefix(name, "toggle-"):
		if f, ok := boolFields[name[len("toggle-"):]]; ok {
			return boolToggle{f}
		}
	default:
		if f, ok := boolFields[name]; ok {
			return boolOn{f}
		}
		if color := parseColor(name); color != nil {
			return setForeground{color}
		}
	}
	return nil
}
