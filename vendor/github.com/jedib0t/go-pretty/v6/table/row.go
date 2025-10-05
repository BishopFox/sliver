package table

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/text"
)

// Row defines a single row in the Table.
type Row []interface{}

func (r Row) findColumnNumber(colName string) int {
	for colIdx, col := range r {
		if fmt.Sprint(col) == colName {
			return colIdx + 1
		}
	}
	return 0
}

// RowAttributes contains properties about the Row during the render.
type RowAttributes struct {
	Number       int // Row Number (1-indexed) as appended
	NumberSorted int // Row number (1-indexed) after sorting
}

// RowPainter is a custom function that takes a Row as input and returns the
// text.Colors{} to use on the entire row
type RowPainter func(row Row) text.Colors

// RowPainterWithAttributes is the same as RowPainter but passes in additional
// attributes from render time
type RowPainterWithAttributes func(row Row, attr RowAttributes) text.Colors

// rowStr defines a single row in the Table comprised of just string objects.
type rowStr []string

// areEqual returns true if the contents of the 2 given columns are the same
func (row rowStr) areEqual(colIdx1 int, colIdx2 int) bool {
	return colIdx1 >= 0 && colIdx1 < len(row) &&
		colIdx2 >= 0 && colIdx2 < len(row) &&
		row[colIdx1] == row[colIdx2]
}
