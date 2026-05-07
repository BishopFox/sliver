package table

// HTMLOptions defines the global options to control HTML rendering.
type HTMLOptions struct {
	ConvertColorsToSpans bool   // convert ANSI escape sequences to HTML <span> tags with CSS classes? EscapeText will be true if this is true.
	CSSClass             string // CSS class to set on the overall <table> tag
	EmptyColumn          string // string to replace "" columns with (entire content being "")
	EscapeText           bool   // escape text into HTML-safe content?
	Newline              string // string to replace "\n" characters with
}

var (
	// DefaultHTMLOptions defines sensible HTML rendering defaults.
	DefaultHTMLOptions = HTMLOptions{
		ConvertColorsToSpans: true,
		CSSClass:             DefaultHTMLCSSClass,
		EmptyColumn:          "&nbsp;",
		EscapeText:           true,
		Newline:              "<br/>",
	}
)
