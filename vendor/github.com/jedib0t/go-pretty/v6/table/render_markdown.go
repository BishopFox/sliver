package table

import (
	"fmt"
	"strings"
)

// RenderMarkdown renders the Table in Markdown format. Example:
//
//	| # | First Name | Last Name | Salary |  |
//	| ---:| --- | --- | ---:| --- |
//	| 1 | Arya | Stark | 3000 |  |
//	| 20 | Jon | Snow | 2000 | You know nothing, Jon Snow! |
//	| 300 | Tyrion | Lannister | 5000 |  |
//	|  |  | Total | 10000 |  |
func (t *Table) RenderMarkdown() string {
	t.initForRender(renderModeMarkdown)

	var out strings.Builder
	if t.numColumns > 0 {
		out.Grow(t.estimatedRenderLength())
		t.markdownRenderTitle(&out)
		t.markdownRenderRowsHeader(&out)
		t.markdownRenderRows(&out, t.rows, renderHint{})
		t.markdownRenderRowsFooter(&out)
		t.markdownRenderCaption(&out)
	}
	return t.render(&out)
}

func (t *Table) markdownRenderCaption(out *strings.Builder) {
	if t.caption != "" {
		out.WriteRune('\n')
		out.WriteRune('_')
		out.WriteString(t.caption)
		out.WriteRune('_')
	}
}

func (t *Table) markdownRenderRow(out *strings.Builder, row rowStr, hint renderHint) {
	// when working on line number 2 or more, insert a newline first
	if out.Len() > 0 {
		out.WriteRune('\n')
	}

	// render each column up to the max. columns seen in all the rows
	out.WriteRune('|')
	for colIdx := 0; colIdx < t.numColumns; colIdx++ {
		t.markdownRenderRowAutoIndex(out, colIdx, hint)

		var colStr string
		if colIdx < len(row) {
			colStr = row[colIdx]
		}
		colStr = strings.ReplaceAll(colStr, "|", "\\|")
		colStr = strings.ReplaceAll(colStr, "\n", "<br/>")
		if t.style.Markdown.PadContent {
			out.WriteRune(' ')
			align := t.getAlign(colIdx, hint)
			out.WriteString(align.Apply(colStr, t.maxColumnLengths[colIdx]))
			out.WriteRune(' ')
		} else {
			out.WriteRune(' ')
			out.WriteString(colStr)
			out.WriteRune(' ')
		}
		out.WriteRune('|')
	}
}

func (t *Table) markdownRenderRowAutoIndex(out *strings.Builder, colIdx int, hint renderHint) {
	if colIdx == 0 && t.autoIndex {
		if hint.isSeparatorRow {
			if t.style.Markdown.PadContent {
				out.WriteString(" " + strings.Repeat("-", t.autoIndexVIndexMaxLength) + ":")
			} else {
				out.WriteRune(' ')
				out.WriteString("---:")
			}
		} else if hint.isRegularRow() {
			if t.style.Markdown.PadContent {
				rowNumStr := fmt.Sprint(hint.rowNumber)
				out.WriteRune(' ')
				fmt.Fprintf(out, "%*s", t.autoIndexVIndexMaxLength, rowNumStr)
				out.WriteRune(' ')
			} else {
				out.WriteRune(' ')
				fmt.Fprintf(out, "%d ", hint.rowNumber)
			}
		} else {
			if t.style.Markdown.PadContent {
				out.WriteRune(' ')
				out.WriteString(strings.Repeat(" ", t.autoIndexVIndexMaxLength))
				out.WriteRune(' ')
			} else {
				out.WriteRune(' ')
			}
		}
		out.WriteRune('|')
	}
}

func (t *Table) markdownRenderRows(out *strings.Builder, rows []rowStr, hint renderHint) {
	if len(rows) > 0 {
		for idx, row := range rows {
			hint.rowNumber = idx + 1
			t.markdownRenderRow(out, row, hint)

			if idx == len(rows)-1 && hint.isHeaderRow {
				t.markdownRenderSeparator(out, renderHint{isSeparatorRow: true})
			}
		}
	}
}

func (t *Table) markdownRenderRowsFooter(out *strings.Builder) {
	t.markdownRenderRows(out, t.rowsFooter, renderHint{isFooterRow: true})
}

func (t *Table) markdownRenderRowsHeader(out *strings.Builder) {
	if len(t.rowsHeader) > 0 {
		t.markdownRenderRows(out, t.rowsHeader, renderHint{isHeaderRow: true})
	} else if t.autoIndex {
		t.markdownRenderRows(out, []rowStr{t.getAutoIndexColumnIDs()}, renderHint{isAutoIndexRow: true, isHeaderRow: true})
	}
}

func (t *Table) markdownRenderSeparator(out *strings.Builder, hint renderHint) {
	// when working on line number 2 or more, insert a newline first
	if out.Len() > 0 {
		out.WriteRune('\n')
	}

	out.WriteRune('|')
	for colIdx := 0; colIdx < t.numColumns; colIdx++ {
		t.markdownRenderRowAutoIndex(out, colIdx, hint)

		align := t.getAlign(colIdx, hint)
		if t.style.Markdown.PadContent {
			out.WriteString(align.MarkdownProperty(t.maxColumnLengths[colIdx]))
		} else {
			out.WriteString(align.MarkdownProperty())
		}
		out.WriteRune('|')
	}
}

func (t *Table) markdownRenderTitle(out *strings.Builder) {
	if t.title != "" {
		out.WriteString("# ")
		out.WriteString(t.title)
	}
}
