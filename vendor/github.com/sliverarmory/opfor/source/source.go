// Package source defines OPFOR's public source-location and diagnostic types.
//
// The root opfor package aliases these types for convenience. Keeping their
// identity in a public package lets lexer, parser, compiler, and embedding
// applications exchange diagnostics without making an internal implementation
// package part of the public API contract.
package source

import "fmt"

// Source is a named unit of Sleep or Aggressor Script source code.
//
// Data is interpreted as UTF-8 by OPFOR's lexer. Callers must not mutate Data
// while a compile operation is using it; OPFOR copies the bytes before
// retaining a compiled Program.
type Source struct {
	Name string
	Data []byte
}

// NewSource constructs a named source unit without copying data.
func NewSource(name string, data []byte) Source {
	return Source{Name: name, Data: data}
}

// Position identifies a point in a Source. Offset is a zero-based byte offset;
// Line and Column are one-based. Columns count Unicode code points, not bytes.
type Position struct {
	Offset int
	Line   int
	Column int
}

// String formats a position as line:column.
func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Span identifies an end-exclusive range in a Source.
type Span struct {
	Source string
	Start  Position
	End    Position
}

// String formats a source span for diagnostics.
func (s Span) String() string {
	if s.Start.Line == s.End.Line {
		return fmt.Sprintf("%s:%d:%d-%d", s.Source, s.Start.Line, s.Start.Column, s.End.Column)
	}
	return fmt.Sprintf("%s:%d:%d-%d:%d", s.Source, s.Start.Line, s.Start.Column, s.End.Line, s.End.Column)
}

// Severity is the severity of a source diagnostic.
type Severity uint8

const (
	// SeverityError identifies a fatal source diagnostic.
	SeverityError Severity = iota
	// SeverityWarning identifies a non-fatal source diagnostic.
	SeverityWarning
	// SeverityInfo identifies an informational source diagnostic.
	SeverityInfo
)

// String returns a stable human-readable severity name.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "unknown"
	}
}

// Diagnostic describes a problem associated with a source span.
type Diagnostic struct {
	Severity Severity
	Code     string
	Message  string
	Span     Span
}

// Error formats a diagnostic as a conventional source error.
func (d Diagnostic) Error() string {
	if d.Code == "" {
		return fmt.Sprintf("%s: %s: %s", d.Span, d.Severity, d.Message)
	}
	return fmt.Sprintf("%s: %s %s: %s", d.Span, d.Severity, d.Code, d.Message)
}
