// Package targettable contains shared layout helpers for the sessions and
// beacons tables.
package targettable

import (
	"path"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Layout describes one rendering candidate for a target table.
type Layout struct {
	ProcessBasename  bool
	IncludeIntegrity bool
	IncludeLocale    bool
	Compact          bool
}

// Fit returns the first table that fits within width. Layouts are tried from
// most to least detailed: full process path, process basename, no Integrity,
// no Locale, and finally the legacy compact layout bounded to width. When no
// Integrity column exists, that omission step is skipped. A non-positive width
// is treated as unknown and keeps the full layout.
func Fit(width int, hasIntegrity bool, build func(Layout) table.Writer) table.Writer {
	layout := Layout{
		IncludeIntegrity: hasIntegrity,
		IncludeLocale:    true,
	}
	if candidate := build(layout); width <= 0 || text.LongestLineLen(candidate.Render()) <= width {
		return candidate
	}

	layout.ProcessBasename = true
	if candidate := build(layout); text.LongestLineLen(candidate.Render()) <= width {
		return candidate
	}

	if hasIntegrity {
		layout.IncludeIntegrity = false
		if candidate := build(layout); text.LongestLineLen(candidate.Render()) <= width {
			return candidate
		}
	}

	layout.IncludeLocale = false
	if candidate := build(layout); text.LongestLineLen(candidate.Render()) <= width {
		return candidate
	}

	compact := build(Layout{Compact: true})
	compact.Style().Size.WidthMax = width
	return compact
}

// ProcessName returns the final path component of a target process path.
// Target paths can use a different separator than the client host, so Windows
// separators are normalized before applying path.Base.
func ProcessName(processPath string) string {
	if processPath == "" {
		return ""
	}
	return path.Base(strings.ReplaceAll(processPath, `\`, "/"))
}
