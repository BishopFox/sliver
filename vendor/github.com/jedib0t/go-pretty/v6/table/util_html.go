package table

import (
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
)

// convertEscSequencesToSpans converts ANSI escape sequences to HTML <span> tags with CSS classes.
func convertEscSequencesToSpans(str string) string {
	converter := newEscSeqToSpanConverter()
	return converter.Convert(str)
}

// escSeqToSpanConverter converts ANSI escape sequences to HTML <span> tags with CSS classes.
type escSeqToSpanConverter struct {
	result        strings.Builder
	esp           text.EscSeqParser
	currentColors map[int]bool
}

// newEscSeqToSpanConverter creates a new escape sequence to span converter.
func newEscSeqToSpanConverter() *escSeqToSpanConverter {
	return &escSeqToSpanConverter{
		currentColors: make(map[int]bool),
	}
}

// Convert converts ANSI escape sequences in the string to HTML <span> tags with CSS classes.
func (c *escSeqToSpanConverter) Convert(str string) string {
	c.reset()

	// Process the string character by character
	for _, char := range str {
		wasInSequence := c.esp.InSequence()
		c.esp.Consume(char)

		if c.esp.InSequence() {
			// We're inside an escape sequence, skip it (don't write to result)
			continue
		}

		if wasInSequence {
			// We just finished an escape sequence, update colors
			newColors := make(map[int]bool)
			for _, code := range c.esp.Codes() {
				newColors[code] = true
			}
			c.updateSpan(newColors)
		} else {
			// Regular character, escape it for HTML safety and write it
			// (will be inside current span if colors are active)
			c.writeEscapedRune(char)
		}
	}

	// Close any open span
	if len(c.currentColors) > 0 {
		c.result.WriteString("</span>")
	}

	return c.result.String()
}

// clearColors clears the current color tracking.
func (c *escSeqToSpanConverter) clearColors() {
	c.currentColors = make(map[int]bool)
}

// closeSpan closes the current span if one is open.
func (c *escSeqToSpanConverter) closeSpan() {
	if len(c.currentColors) > 0 {
		c.result.WriteString("</span>")
	}
}

// colorsChanged checks if the color set has changed.
func (c *escSeqToSpanConverter) colorsChanged(newColors map[int]bool) bool {
	// we never set the map values to false, so a simple size compare is enough
	return len(c.currentColors) != len(newColors)
}

// cssClasses converts color codes to CSS class names.
func (c *escSeqToSpanConverter) cssClasses(codes map[int]bool) string {
	var colors text.Colors
	for code := range codes {
		colors = append(colors, text.Color(code))
	}
	return colors.CSSClasses()
}

// openSpan opens a new span with the given CSS class and tracks the colors.
func (c *escSeqToSpanConverter) openSpan(class string, newColors map[int]bool) {
	c.result.WriteString("<span class=\"")
	c.result.WriteString(class)
	c.result.WriteString("\">")
	// Track colors since we opened a span
	c.currentColors = make(map[int]bool)
	for code := range newColors {
		c.currentColors[code] = true
	}
}

// reset initializes the converter state for a new conversion.
func (c *escSeqToSpanConverter) reset() {
	c.result.Reset()
	c.esp = text.EscSeqParser{}
	c.currentColors = make(map[int]bool)
}

// updateSpan updates span tags when colors change.
func (c *escSeqToSpanConverter) updateSpan(newColors map[int]bool) {
	if !c.colorsChanged(newColors) {
		return
	}

	c.closeSpan()

	// Open new span if there are colors with valid CSS classes
	if len(newColors) > 0 {
		class := c.cssClasses(newColors)
		if class != "" {
			c.openSpan(class, newColors)
		} else {
			// No CSS classes, so don't track these colors
			c.clearColors()
		}
	} else {
		// No colors, clear tracking
		c.clearColors()
	}
}

// writeEscapedRune writes a rune to the result, escaping it if necessary for HTML safety.
func (c *escSeqToSpanConverter) writeEscapedRune(char rune) {
	switch char {
	case '<':
		c.result.WriteString("&lt;")
	case '>':
		c.result.WriteString("&gt;")
	case '&':
		c.result.WriteString("&amp;")
	case '"':
		c.result.WriteString("&#34;")
	case '\'':
		c.result.WriteString("&#39;")
	default:
		// Most characters don't need escaping, write directly
		c.result.WriteRune(char)
	}
}
