// Package table provides a styled table renderer for terminals.
package table

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// HeaderRow denotes the header's row index used when rendering headers. Use
// this value when looking to customize header styles in StyleFunc.
const HeaderRow int = -1

// StyleFunc is the style function that determines the style of a Cell.
//
// It takes the row and column of the cell as an input and determines the
// lipgloss Style to use for that cell position.
//
// Example:
//
//	t := table.New().
//	    Headers("Name", "Age").
//	    Row("Kini", 4).
//	    Row("Eli", 1).
//	    Row("Iris", 102).
//	    StyleFunc(func(row, col int) lipgloss.Style {
//	        switch {
//	           case row == 0:
//	               return HeaderStyle
//	           case row%2 == 0:
//	               return EvenRowStyle
//	           default:
//	               return OddRowStyle
//	           }
//	    })
type StyleFunc func(row, col int) lipgloss.Style

// DefaultStyles is a TableStyleFunc that returns a new Style with no attributes.
func DefaultStyles(_, _ int) lipgloss.Style {
	return lipgloss.NewStyle()
}

// Table is a type for rendering tables.
type Table struct {
	styleFunc StyleFunc
	border    lipgloss.Border

	borderTop    bool
	borderBottom bool
	borderLeft   bool
	borderRight  bool
	borderHeader bool
	borderColumn bool
	borderRow    bool

	borderStyle lipgloss.Style
	headers     []string
	data        Data

	width           int
	height          int
	useManualHeight bool
	offset          int
	wrap            bool

	// widths tracks the width of each column.
	widths []int

	// heights tracks the height of each row.
	heights []int
}

// New returns a new Table that can be modified through different
// attributes.
//
// By default, a table has no border, no styling, and no rows.
func New() *Table {
	return &Table{
		styleFunc:    DefaultStyles,
		border:       lipgloss.RoundedBorder(),
		borderBottom: true,
		borderColumn: true,
		borderHeader: true,
		borderLeft:   true,
		borderRight:  true,
		borderTop:    true,
		wrap:         true,
		data:         NewStringData(),
	}
}

// ClearRows clears the table rows.
func (t *Table) ClearRows() *Table {
	t.data = NewStringData()
	return t
}

// StyleFunc sets the style for a cell based on it's position (row, column).
func (t *Table) StyleFunc(style StyleFunc) *Table {
	t.styleFunc = style
	return t
}

// style returns the style for a cell based on it's position (row, column).
func (t *Table) style(row, col int) lipgloss.Style {
	if t.styleFunc == nil {
		return lipgloss.NewStyle()
	}
	return t.styleFunc(row, col)
}

// Data sets the table data.
func (t *Table) Data(data Data) *Table {
	t.data = data
	return t
}

// Rows appends rows to the table data.
func (t *Table) Rows(rows ...[]string) *Table {
	for _, row := range rows {
		switch t.data.(type) {
		case *StringData:
			t.data.(*StringData).Append(row)
		}
	}
	return t
}

// Row appends a row to the table data.
func (t *Table) Row(row ...string) *Table {
	switch t.data.(type) {
	case *StringData:
		t.data.(*StringData).Append(row)
	}
	return t
}

// Headers sets the table headers.
func (t *Table) Headers(headers ...string) *Table {
	t.headers = headers
	return t
}

// Border sets the table border.
func (t *Table) Border(border lipgloss.Border) *Table {
	t.border = border
	return t
}

// BorderTop sets the top border.
func (t *Table) BorderTop(v bool) *Table {
	t.borderTop = v
	return t
}

// BorderBottom sets the bottom border.
func (t *Table) BorderBottom(v bool) *Table {
	t.borderBottom = v
	return t
}

// BorderLeft sets the left border.
func (t *Table) BorderLeft(v bool) *Table {
	t.borderLeft = v
	return t
}

// BorderRight sets the right border.
func (t *Table) BorderRight(v bool) *Table {
	t.borderRight = v
	return t
}

// BorderHeader sets the header separator border.
func (t *Table) BorderHeader(v bool) *Table {
	t.borderHeader = v
	return t
}

// BorderColumn sets the column border separator.
func (t *Table) BorderColumn(v bool) *Table {
	t.borderColumn = v
	return t
}

// BorderRow sets the row border separator.
func (t *Table) BorderRow(v bool) *Table {
	t.borderRow = v
	return t
}

// BorderStyle sets the style for the table border.
func (t *Table) BorderStyle(style lipgloss.Style) *Table {
	t.borderStyle = style
	return t
}

// Width sets the table width, this auto-sizes the columns to fit the width by
// either expanding or contracting the widths of each column as a best effort
// approach.
func (t *Table) Width(w int) *Table {
	t.width = w
	return t
}

// Height sets the table height.
func (t *Table) Height(h int) *Table {
	t.height = h
	t.useManualHeight = true
	return t
}

// Offset sets the table rendering offset.
//
// Warning: you may declare Offset only after setting Rows. Otherwise it will be
// ignored.
func (t *Table) Offset(o int) *Table {
	t.offset = o
	return t
}

// Wrap dictates whether or not the table content should wrap.
func (t *Table) Wrap(w bool) *Table {
	t.wrap = w
	return t
}

// String returns the table as a string.
func (t *Table) String() string {
	hasHeaders := len(t.headers) > 0
	hasRows := t.data != nil && t.data.Rows() > 0

	if !hasHeaders && !hasRows {
		return ""
	}

	// Add empty cells to the headers, until it's the same length as the longest
	// row (only if there are at headers in the first place).
	if hasHeaders {
		for i := len(t.headers); i < t.data.Columns(); i++ {
			t.headers = append(t.headers, "")
		}
	}

	// Do all the sizing calculations for width and height.
	t.resize()

	var sb strings.Builder

	if t.borderTop {
		sb.WriteString(t.constructTopBorder())
		sb.WriteString("\n")
	}

	if hasHeaders {
		sb.WriteString(t.constructHeaders())
		sb.WriteString("\n")
	}

	var bottom string
	if t.borderBottom {
		bottom = t.constructBottomBorder()
	}

	// If there are no data rows render nothing.
	if t.data.Rows() > 0 {
		switch {
		case t.useManualHeight:
			// The height of the top border. Subtract 1 for the newline.
			topHeight := lipgloss.Height(sb.String()) - 1
			availableLines := t.height - (topHeight + lipgloss.Height(bottom))

			// if the height is larger than the number of rows, use the number
			// of rows.
			if availableLines > t.data.Rows() {
				availableLines = t.data.Rows()
			}
			sb.WriteString(t.constructRows(availableLines))

		default:
			for r := t.offset; r < t.data.Rows(); r++ {
				sb.WriteString(t.constructRow(r, false))
			}
		}
	}

	sb.WriteString(bottom)

	return lipgloss.NewStyle().
		MaxHeight(t.computeHeight()).
		MaxWidth(t.width).
		Render(sb.String())
}

// computeHeight computes the height of the table in it's current configuration.
func (t *Table) computeHeight() int {
	hasHeaders := len(t.headers) > 0
	return sum(t.heights) - 1 + btoi(hasHeaders) +
		btoi(t.borderTop) + btoi(t.borderBottom) +
		btoi(t.borderHeader) + t.data.Rows()*btoi(t.borderRow)
}

// Render returns the table as a string.
func (t *Table) Render() string {
	return t.String()
}

// constructTopBorder constructs the top border for the table given it's current
// border configuration and data.
func (t *Table) constructTopBorder() string {
	var s strings.Builder
	if t.borderLeft {
		s.WriteString(t.borderStyle.Render(t.border.TopLeft))
	}
	for i := 0; i < len(t.widths); i++ {
		s.WriteString(t.borderStyle.Render(strings.Repeat(t.border.Top, t.widths[i])))
		if i < len(t.widths)-1 && t.borderColumn {
			s.WriteString(t.borderStyle.Render(t.border.MiddleTop))
		}
	}
	if t.borderRight {
		s.WriteString(t.borderStyle.Render(t.border.TopRight))
	}
	return s.String()
}

// constructBottomBorder constructs the bottom border for the table given it's current
// border configuration and data.
func (t *Table) constructBottomBorder() string {
	var s strings.Builder
	if t.borderLeft {
		s.WriteString(t.borderStyle.Render(t.border.BottomLeft))
	}
	for i := 0; i < len(t.widths); i++ {
		s.WriteString(t.borderStyle.Render(strings.Repeat(t.border.Bottom, t.widths[i])))
		if i < len(t.widths)-1 && t.borderColumn {
			s.WriteString(t.borderStyle.Render(t.border.MiddleBottom))
		}
	}
	if t.borderRight {
		s.WriteString(t.borderStyle.Render(t.border.BottomRight))
	}
	return s.String()
}

// constructHeaders constructs the headers for the table given it's current
// header configuration and data.
func (t *Table) constructHeaders() string {
	height := t.heights[HeaderRow+1]

	var s strings.Builder
	if t.borderLeft {
		s.WriteString(t.borderStyle.Render(t.border.Left))
	}
	for i, header := range t.headers {
		cellStyle := t.style(HeaderRow, i)

		if !t.wrap {
			header = t.truncateCell(header, HeaderRow, i)
		}

		s.WriteString(cellStyle.
			Height(height - cellStyle.GetVerticalMargins()).
			MaxHeight(height).
			Width(t.widths[i] - cellStyle.GetHorizontalMargins()).
			MaxWidth(t.widths[i]).
			Render(t.truncateCell(header, HeaderRow, i)))
		if i < len(t.headers)-1 && t.borderColumn {
			s.WriteString(t.borderStyle.Render(t.border.Left))
		}
	}
	if t.borderHeader {
		if t.borderRight {
			s.WriteString(t.borderStyle.Render(t.border.Right))
		}
		s.WriteString("\n")
		if t.borderLeft {
			s.WriteString(t.borderStyle.Render(t.border.MiddleLeft))
		}
		for i := 0; i < len(t.headers); i++ {
			s.WriteString(t.borderStyle.Render(strings.Repeat(t.border.Top, t.widths[i])))
			if i < len(t.headers)-1 && t.borderColumn {
				s.WriteString(t.borderStyle.Render(t.border.Middle))
			}
		}
		if t.borderRight {
			s.WriteString(t.borderStyle.Render(t.border.MiddleRight))
		}
	}
	if t.borderRight && !t.borderHeader {
		s.WriteString(t.borderStyle.Render(t.border.Right))
	}
	return s.String()
}

func (t *Table) constructRows(availableLines int) string {
	var sb strings.Builder

	// The number of rows to render after removing the offset.
	offsetRowCount := t.data.Rows() - t.offset

	// The number of rows to render. We always render at least one row.
	rowsToRender := availableLines
	rowsToRender = max(rowsToRender, 1)

	// Check if we need to render an overflow row.
	needsOverflow := rowsToRender < offsetRowCount

	// only use the offset as the starting value if there is overflow.
	rowIdx := t.offset
	if !needsOverflow {
		// if there is no overflow, just render to the height of the table
		// check there's enough content to fill the table
		rowIdx = t.data.Rows() - rowsToRender
	}
	for rowsToRender > 0 && rowIdx < t.data.Rows() {
		// Whenever the height is too small to render all rows, the bottom row will be an overflow row (ellipsis).
		isOverflow := needsOverflow && rowsToRender == 1

		sb.WriteString(t.constructRow(rowIdx, isOverflow))

		rowIdx++
		rowsToRender--
	}
	return sb.String()
}

// constructRow constructs the row for the table given an index and row data
// based on the current configuration. If isOverflow is true, the row is
// rendered as an overflow row (using ellipsis).
func (t *Table) constructRow(index int, isOverflow bool) string {
	var s strings.Builder

	hasHeaders := len(t.headers) > 0
	height := t.heights[index+btoi(hasHeaders)]
	if isOverflow {
		height = 1
	}

	var cells []string
	left := strings.Repeat(t.borderStyle.Render(t.border.Left)+"\n", height)
	if t.borderLeft {
		cells = append(cells, left)
	}

	for c := 0; c < t.data.Columns(); c++ {
		cell := "…"
		if !isOverflow {
			cell = t.data.At(index, c)
		}

		cellStyle := t.style(index, c)
		if !t.wrap {
			cell = t.truncateCell(cell, index, c)
		}
		cells = append(cells, cellStyle.
			// Account for the margins in the cell sizing.
			Height(height-cellStyle.GetVerticalMargins()).
			MaxHeight(height).
			Width(t.widths[c]-cellStyle.GetHorizontalMargins()).
			MaxWidth(t.widths[c]).
			Render(cell))

		if c < t.data.Columns()-1 && t.borderColumn {
			cells = append(cells, left)
		}
	}

	if t.borderRight {
		right := strings.Repeat(t.borderStyle.Render(t.border.Right)+"\n", height)
		cells = append(cells, right)
	}

	for i, cell := range cells {
		cells[i] = strings.TrimRight(cell, "\n")
	}

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...) + "\n")

	if t.borderRow && index < t.data.Rows()-1 && !isOverflow {
		s.WriteString(t.borderStyle.Render(t.border.MiddleLeft))
		for i := 0; i < len(t.widths); i++ {
			s.WriteString(t.borderStyle.Render(strings.Repeat(t.border.Bottom, t.widths[i])))
			if i < len(t.widths)-1 && t.borderColumn {
				s.WriteString(t.borderStyle.Render(t.border.Middle))
			}
		}
		s.WriteString(t.borderStyle.Render(t.border.MiddleRight) + "\n")
	}

	return s.String()
}

func (t *Table) truncateCell(cell string, rowIndex, colIndex int) string {
	hasHeaders := len(t.headers) > 0
	height := t.heights[rowIndex+btoi(hasHeaders)]
	cellWidth := t.widths[colIndex]
	cellStyle := t.style(rowIndex, colIndex)

	length := (cellWidth * height) - cellStyle.GetHorizontalPadding() - cellStyle.GetHorizontalMargins()
	return ansi.Truncate(cell, length, "…")
}
