package ui

import (
	"fmt"
	"strings"
)

// Style specifies how something (mostly a string) shall be displayed.
type Style struct {
	Foreground Color
	Background Color
	Bold       bool
	Dim        bool
	Italic     bool
	Underlined bool
	Blink      bool
	Inverse    bool
}

// SGR returns SGR sequence for the style.
func (s Style) SGR() string {
	var sgr []string

	addIf := func(b bool, code string) {
		if b {
			sgr = append(sgr, code)
		}
	}
	addIf(s.Bold, "1")
	addIf(s.Dim, "2")
	addIf(s.Italic, "3")
	addIf(s.Underlined, "4")
	addIf(s.Blink, "5")
	addIf(s.Inverse, "7")
	if s.Foreground != nil {
		sgr = append(sgr, s.Foreground.fgSGR())
	}
	if s.Background != nil {
		sgr = append(sgr, s.Background.bgSGR())
	}

	return strings.Join(sgr, ";")
}

// MergeFromOptions merges all recognized values from a map to the current
// Style.
func (s *Style) MergeFromOptions(options map[string]interface{}) error {
	assignColor := func(val interface{}, colorField *Color) string {
		if val == "default" {
			*colorField = nil
			return ""
		} else if s, ok := val.(string); ok {
			color := parseColor(s)
			if color != nil {
				*colorField = color
				return ""
			}
		}
		return "valid color string"
	}
	assignBool := func(val interface{}, attrField *bool) string {
		if b, ok := val.(bool); ok {
			*attrField = b
		} else {
			return "bool value"
		}
		return ""
	}

	for k, v := range options {
		var need string

		switch k {
		case "fg-color":
			need = assignColor(v, &s.Foreground)
		case "bg-color":
			need = assignColor(v, &s.Background)
		case "bold":
			need = assignBool(v, &s.Bold)
		case "dim":
			need = assignBool(v, &s.Dim)
		case "italic":
			need = assignBool(v, &s.Italic)
		case "underlined":
			need = assignBool(v, &s.Underlined)
		case "blink":
			need = assignBool(v, &s.Blink)
		case "inverse":
			need = assignBool(v, &s.Inverse)

		default:
			return fmt.Errorf("unrecognized option '%s'", k)
		}

		if need != "" {
			return fmt.Errorf("value for option '%s' must be a %s", k, need)
		}
	}

	return nil
}
