// Package lexer tokenizes Sleep and Aggressor Script source code.
package lexer

import "github.com/sliverarmory/opfor/source"

const anonymousSourceName = "<input>"

// Source is a named unit of source code.
type Source = source.Source

// NewSource constructs a named source unit without copying data.
func NewSource(name string, data []byte) Source {
	return source.NewSource(name, data)
}

func sourceName(s Source) string {
	if s.Name == "" {
		return anonymousSourceName
	}
	return s.Name
}

// Position identifies a point in a Source. Offset is a zero-based byte offset;
// Line and Column are one-based. Columns count Unicode code points, not bytes.
type Position = source.Position

// Span identifies an end-exclusive range in a Source.
type Span = source.Span

// Severity is the severity of a source diagnostic.
type Severity = source.Severity

const (
	SeverityError   = source.SeverityError
	SeverityWarning = source.SeverityWarning
	SeverityInfo    = source.SeverityInfo
)

// Diagnostic describes a problem associated with a source span.
type Diagnostic = source.Diagnostic
