package table

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// resize resizes the table to fit the specified width.
//
// Given a user defined table width, we must ensure the table is exactly that
// width. This must account for all borders, column, separators, and column
// data.
//
// In the case where the table is narrower than the specified table width,
// we simply expand the columns evenly to fit the width.
// For example, a table with 3 columns takes up 50 characters total, and the
// width specified is 80, we expand each column by 10 characters, adding 30
// to the total width.
//
// In the case where the table is wider than the specified table width, we
// _could_ simply shrink the columns evenly but this would result in data
// being truncated (perhaps unnecessarily). The naive approach could result
// in very poor cropping of the table data. So, instead of shrinking columns
// evenly, we calculate the median non-whitespace length of each column, and
// shrink the columns based on the largest median.
//
// For example,
//
//	┌──────┬───────────────┬──────────┐
//	│ Name │ Age of Person │ Location │
//	├──────┼───────────────┼──────────┤
//	│ Kini │ 40            │ New York │
//	│ Eli  │ 30            │ London   │
//	│ Iris │ 20            │ Paris    │
//	└──────┴───────────────┴──────────┘
//
// Median non-whitespace length  vs column width of each column:
//
// Name: 4 / 5
// Age of Person: 2 / 15
// Location: 6 / 10
//
// The biggest difference is 15 - 2, so we can shrink the 2nd column by 13.
func (t *Table) resize() {
	hasHeaders := len(t.headers) > 0
	rows := dataToMatrix(t.data)
	r := newResizer(t.width, t.height, t.headers, rows)
	r.wrap = t.wrap
	r.borderColumn = t.borderColumn
	r.yPaddings = make([][]int, len(r.allRows))

	var allRows [][]string
	if hasHeaders {
		allRows = append([][]string{t.headers}, rows...)
	} else {
		allRows = rows
	}

	styleFunc := t.styleFunc
	if t.styleFunc == nil {
		styleFunc = DefaultStyles
	}

	r.rowHeights = r.defaultRowHeights()

	for i, row := range allRows {
		r.yPaddings[i] = make([]int, len(row))

		for j := range row {
			column := &r.columns[j]

			// Making sure we're passing the right index to `styleFunc`. The header row should be `-1` and
			// the others should start from `0`.
			rowIndex := i
			if hasHeaders {
				rowIndex--
			}
			style := styleFunc(rowIndex, j)

			topMargin, rightMargin, bottomMargin, leftMargin := style.GetMargin()
			topPadding, rightPadding, bottomPadding, leftPadding := style.GetPadding()

			totalHorizontalPadding := leftMargin + rightMargin + leftPadding + rightPadding
			column.xPadding = max(column.xPadding, totalHorizontalPadding)
			column.fixedWidth = max(column.fixedWidth, style.GetWidth())

			r.rowHeights[i] = max(r.rowHeights[i], style.GetHeight())

			totalVerticalPadding := topMargin + bottomMargin + topPadding + bottomPadding
			r.yPaddings[i][j] = totalVerticalPadding
		}
	}

	// A table width wasn't specified. In this case, detect according to
	// content width.
	if r.tableWidth <= 0 {
		r.tableWidth = r.detectTableWidth()
	}

	t.widths, t.heights = r.optimizedWidths()
}

// resizerColumn is a column in the resizer.
type resizerColumn struct {
	index      int
	min        int
	max        int
	median     int
	rows       [][]string
	xPadding   int // horizontal padding
	fixedWidth int
}

// resizer is a table resizer.
type resizer struct {
	tableWidth  int
	tableHeight int
	headers     []string
	allRows     [][]string
	rowHeights  []int
	columns     []resizerColumn

	wrap         bool
	borderColumn bool
	yPaddings    [][]int // vertical paddings
}

// newResizer creates a new resizer.
func newResizer(tableWidth, tableHeight int, headers []string, rows [][]string) *resizer {
	r := &resizer{
		tableWidth:  tableWidth,
		tableHeight: tableHeight,
		headers:     headers,
	}

	if len(headers) > 0 {
		r.allRows = append([][]string{headers}, rows...)
	} else {
		r.allRows = rows
	}

	for _, row := range r.allRows {
		for i, cell := range row {
			cellLen := lipgloss.Width(cell)

			// Header or first row. Just add as is.
			if len(r.columns) <= i {
				r.columns = append(r.columns, resizerColumn{
					index:  i,
					min:    cellLen,
					max:    cellLen,
					median: cellLen,
				})
				continue
			}

			r.columns[i].rows = append(r.columns[i].rows, row)
			r.columns[i].min = min(r.columns[i].min, cellLen)
			r.columns[i].max = max(r.columns[i].max, cellLen)
		}
	}
	for j := range r.columns {
		widths := make([]int, len(r.columns[j].rows))
		for i, row := range r.columns[j].rows {
			widths[i] = lipgloss.Width(row[j])
		}
		r.columns[j].median = median(widths)
	}

	return r
}

// optimizedWidths returns the optimized column widths and row heights.
func (r *resizer) optimizedWidths() (colWidths, rowHeights []int) {
	if r.maxTotal() <= r.tableWidth {
		return r.expandTableWidth()
	}
	return r.shrinkTableWidth()
}

// detectTableWidth detects the table width.
func (r *resizer) detectTableWidth() int {
	return r.maxCharCount() + r.totalHorizontalPadding() + r.totalHorizontalBorder()
}

// expandTableWidth expands the table width.
func (r *resizer) expandTableWidth() (colWidths, rowHeights []int) {
	colWidths = r.maxColumnWidths()

	for {
		totalWidth := sum(colWidths) + r.totalHorizontalBorder()
		if totalWidth >= r.tableWidth {
			break
		}

		shorterColumnIndex := 0
		shorterColumnWidth := math.MaxInt32

		for j, width := range colWidths {
			if width == r.columns[j].fixedWidth {
				continue
			}
			if width < shorterColumnWidth {
				shorterColumnWidth = width
				shorterColumnIndex = j
			}
		}

		colWidths[shorterColumnIndex]++
	}

	rowHeights = r.expandRowHeights(colWidths)
	return
}

// shrinkTableWidth shrinks the table width.
func (r *resizer) shrinkTableWidth() (colWidths, rowHeights []int) {
	colWidths = r.maxColumnWidths()

	// Cut width of columns that are way too big.
	shrinkBiggestColumns := func(veryBigOnly bool) {
		for {
			totalWidth := sum(colWidths) + r.totalHorizontalBorder()
			if totalWidth <= r.tableWidth {
				break
			}

			bigColumnIndex := -math.MaxInt32
			bigColumnWidth := -math.MaxInt32

			for j, width := range colWidths {
				if width == r.columns[j].fixedWidth {
					continue
				}
				if veryBigOnly {
					if width >= (r.tableWidth/2) && width > bigColumnWidth { //nolint:mnd
						bigColumnWidth = width
						bigColumnIndex = j
					}
				} else {
					if width > bigColumnWidth {
						bigColumnWidth = width
						bigColumnIndex = j
					}
				}
			}

			if bigColumnIndex < 0 || colWidths[bigColumnIndex] == 0 {
				break
			}
			colWidths[bigColumnIndex]--
		}
	}

	// Cut width of columns that differ the most from the median.
	shrinkToMedian := func() {
		for {
			totalWidth := sum(colWidths) + r.totalHorizontalBorder()
			if totalWidth <= r.tableWidth {
				break
			}

			biggestDiffToMedian := -math.MaxInt32
			biggestDiffToMedianIndex := -math.MaxInt32

			for j, width := range colWidths {
				if width == r.columns[j].fixedWidth {
					continue
				}
				diffToMedian := width - r.columns[j].median
				if diffToMedian > 0 && diffToMedian > biggestDiffToMedian {
					biggestDiffToMedian = diffToMedian
					biggestDiffToMedianIndex = j
				}
			}

			if biggestDiffToMedianIndex <= 0 || colWidths[biggestDiffToMedianIndex] == 0 {
				break
			}
			colWidths[biggestDiffToMedianIndex]--
		}
	}

	shrinkBiggestColumns(true)
	shrinkToMedian()
	shrinkBiggestColumns(false)

	return colWidths, r.expandRowHeights(colWidths)
}

// expandRowHeights expands the row heights.
func (r *resizer) expandRowHeights(colWidths []int) (rowHeights []int) {
	rowHeights = r.defaultRowHeights()
	if !r.wrap {
		return rowHeights
	}
	hasHeaders := len(r.headers) > 0

	for i, row := range r.allRows {
		for j, cell := range row {
			// NOTE(@andreynering): Headers always have a height of 1, even when wrap is enabled.
			if hasHeaders && i == 0 {
				continue
			}
			height := r.detectContentHeight(cell, colWidths[j]-r.xPaddingForCol(j)) + r.xPaddingForCell(i, j)
			if height > rowHeights[i] {
				rowHeights[i] = height
			}
		}
	}
	return
}

// defaultRowHeights returns the default row heights.
func (r *resizer) defaultRowHeights() (rowHeights []int) {
	rowHeights = make([]int, len(r.allRows))
	for i := range rowHeights {
		if i < len(r.rowHeights) {
			rowHeights[i] = r.rowHeights[i]
		}
		if rowHeights[i] < 1 {
			rowHeights[i] = 1
		}
	}
	return
}

// maxColumnWidths returns the maximum column widths.
func (r *resizer) maxColumnWidths() []int {
	maxColumnWidths := make([]int, len(r.columns))
	for i, col := range r.columns {
		if col.fixedWidth > 0 {
			maxColumnWidths[i] = col.fixedWidth
		} else {
			maxColumnWidths[i] = col.max + r.xPaddingForCol(col.index)
		}
	}
	return maxColumnWidths
}

// columnCount returns the column count.
func (r *resizer) columnCount() int {
	return len(r.columns)
}

// maxCharCount returns the maximum character count.
func (r *resizer) maxCharCount() int {
	var count int
	for _, col := range r.columns {
		if col.fixedWidth > 0 {
			count += col.fixedWidth - r.xPaddingForCol(col.index)
		} else {
			count += col.max
		}
	}
	return count
}

// maxTotal returns the maximum total width.
func (r *resizer) maxTotal() (maxTotal int) {
	for j, column := range r.columns {
		if column.fixedWidth > 0 {
			maxTotal += column.fixedWidth
		} else {
			maxTotal += column.max + r.xPaddingForCol(j)
		}
	}
	return
}

// totalHorizontalPadding returns the total padding.
func (r *resizer) totalHorizontalPadding() (totalHorizontalPadding int) {
	for _, col := range r.columns {
		totalHorizontalPadding += col.xPadding
	}
	return
}

// xPaddingForCol returns the horizontal padding for a column.
func (r *resizer) xPaddingForCol(j int) int {
	if j >= len(r.columns) {
		return 0
	}
	return r.columns[j].xPadding
}

// xPaddingForCell returns the horizontal padding for a cell.
func (r *resizer) xPaddingForCell(i, j int) int {
	if i >= len(r.yPaddings) || j >= len(r.yPaddings[i]) {
		return 0
	}
	return r.yPaddings[i][j]
}

// totalHorizontalBorder returns the total border.
func (r *resizer) totalHorizontalBorder() int {
	return (r.columnCount() * r.borderPerCell()) + r.extraBorder()
}

// borderPerCell returns number of border chars per cell.
func (r *resizer) borderPerCell() int {
	if r.borderColumn {
		return 1
	}
	return 0
}

// extraBorder returns the number of the extra border char at the end of the table.
func (r *resizer) extraBorder() int {
	if r.borderColumn {
		return 1
	}
	return 0
}

// detectContentHeight detects the content height.
func (r *resizer) detectContentHeight(content string, width int) (height int) {
	if width == 0 {
		return 1
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	for _, line := range strings.Split(content, "\n") {
		height += strings.Count(ansi.Wrap(line, width, ""), "\n") + 1
	}
	return
}
