package opfor

import (
	"path/filepath"

	"github.com/sliverarmory/opfor/source"
)

// Source is a named unit of Sleep or Aggressor Script source code.
//
// Data is interpreted as UTF-8 by the lexer. Positions use byte offsets and
// one-based line and column numbers.
type Source = source.Source

// Position identifies a point in a Source. Offset is a zero-based byte offset;
// Line and Column are one-based.
type Position = source.Position

// Span identifies an end-exclusive range in a Source.
type Span = source.Span

// Severity is the severity of a source diagnostic.
type Severity = source.Severity

const (
	// SeverityError identifies a fatal source diagnostic.
	SeverityError = source.SeverityError
	// SeverityWarning identifies a non-fatal source diagnostic.
	SeverityWarning = source.SeverityWarning
	// SeverityInfo identifies an informational source diagnostic.
	SeverityInfo = source.SeverityInfo
)

// Diagnostic describes a problem associated with a source span.
type Diagnostic = source.Diagnostic

// NewSource constructs a named source unit without copying data.
func NewSource(name string, data []byte) Source {
	return source.NewSource(name, data)
}

// sleepDisplayLine converts OPFOR's one-based source positions to the line
// numbering Sleep exposes in user-facing diagnostics. Dynamic &eval/&expr
// source is the one exception in the reference runtime: its displayed lines
// are zero-based even though ordinary files and OPFOR's public Span API remain
// one-based.
func sleepDisplayLine(span Span) int {
	return sleepDisplayLineNumber(span.Source, span.Start.Line)
}

func sleepDisplayLineNumber(source string, line int) int {
	if source == "eval" {
		line--
		if line < 0 {
			return 0
		}
	}
	return line
}

// sleepSourceDisplayName matches ScriptWarning.getNameShort and
// Block.getSourceLocation: execution retains the logical source identity, but
// user-facing warnings, traces, and closure descriptions render its basename.
func sleepSourceDisplayName(source string) string {
	if source == "" {
		return ""
	}
	name := filepath.Base(filepath.FromSlash(source))
	if name == "." || name == string(filepath.Separator) {
		return source
	}
	return name
}
