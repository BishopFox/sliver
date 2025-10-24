package table

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/jedib0t/go-pretty/v6/text"
)

func (t *Table) analyzeAndStringify(row Row, hint renderHint) rowStr {
	// update t.numColumns if this row is the longest seen till now
	if len(row) > t.numColumns {
		// init the slice for the first time; and pad it the rest of the time
		if t.numColumns == 0 {
			t.columnIsNonNumeric = make([]bool, len(row))
		} else {
			t.columnIsNonNumeric = append(t.columnIsNonNumeric, make([]bool, len(row)-t.numColumns)...)
		}
		// update t.numColumns
		t.numColumns = len(row)
	}

	// convert each column to string and figure out if it has non-numeric data
	rowOut := make(rowStr, len(row))
	for colIdx, col := range row {
		// if the column is not a number, keep track of it
		if !hint.isHeaderRow && !hint.isFooterRow && !t.columnIsNonNumeric[colIdx] && !isNumber(col) {
			t.columnIsNonNumeric[colIdx] = true
		}

		rowOut[colIdx] = t.analyzeAndStringifyColumn(colIdx, col, hint)
	}
	return rowOut
}

func (t *Table) analyzeAndStringifyColumn(colIdx int, col interface{}, hint renderHint) string {
	// convert to a string and store it in the row
	var colStr string
	if transformer := t.getColumnTransformer(colIdx, hint); transformer != nil {
		colStr = transformer(col)
	} else if colStrVal, ok := col.(string); ok {
		colStr = colStrVal
	} else {
		colStr = fmt.Sprint(col)
	}
	colStr = strings.ReplaceAll(colStr, "\t", "    ")
	colStr = text.ProcessCRLF(colStr)
	return fmt.Sprintf("%s%s", t.style.Format.Direction.Modifier(), colStr)
}

func (t *Table) extractMaxColumnLengths(rows []rowStr, hint renderHint) {
	for rowIdx, row := range rows {
		hint.rowNumber = rowIdx + 1
		t.extractMaxColumnLengthsFromRow(row, t.getMergedColumnIndices(row, hint))
	}
}

func (t *Table) extractMaxColumnLengthsFromRow(row rowStr, mci mergedColumnIndices) {
	for colIdx := 0; colIdx < len(row); colIdx++ {
		colStr := row[colIdx]
		longestLineLen := text.LongestLineLen(colStr)
		maxColWidth := t.getColumnWidthMax(colIdx)
		if maxColWidth > 0 && maxColWidth < longestLineLen {
			longestLineLen = maxColWidth
		}

		if mergeEndIndex, ok := mci[colIdx]; ok {
			startIndexMap := t.maxMergedColumnLengths[mergeEndIndex]
			if startIndexMap == nil {
				startIndexMap = make(map[int]int)
				t.maxMergedColumnLengths[mergeEndIndex] = startIndexMap
			}
			if longestLineLen > startIndexMap[colIdx] {
				startIndexMap[colIdx] = longestLineLen
			}
			colIdx = mergeEndIndex
		} else if longestLineLen > t.maxColumnLengths[colIdx] {
			t.maxColumnLengths[colIdx] = longestLineLen
		}
	}
}

// reBalanceMaxMergedColumnLengths tries to re-balance the merged column lengths
// across all columns. It does this from the lowest end index to the highest,
// and within that set from the highest start index to the lowest. It
// distributes the length across the columns not already exceeding the average.
func (t *Table) reBalanceMaxMergedColumnLengths() {
	endIndexKeys, startIndexKeysMap := getSortedKeys(t.maxMergedColumnLengths)
	middleSepLen := text.StringWidthWithoutEscSequences(t.style.Box.MiddleSeparator)
	for _, endIndexKey := range endIndexKeys {
		startIndexKeys := startIndexKeysMap[endIndexKey]
		for idx := len(startIndexKeys) - 1; idx >= 0; idx-- {
			startIndexKey := startIndexKeys[idx]
			columnBalanceMap := map[int]struct{}{}
			for index := startIndexKey; index <= endIndexKey; index++ {
				columnBalanceMap[index] = struct{}{}
			}
			mergedColumnLength := t.maxMergedColumnLengths[endIndexKey][startIndexKey] -
				((len(columnBalanceMap) - 1) * middleSepLen)

			// keep reducing the set of columns until the remainder are the ones less than
			// the average of the remaining length (total merged length - all lengths > average)
			for {
				if mergedColumnLength <= 0 { // already exceeded the merged length
					columnBalanceMap = map[int]struct{}{}
					break
				}
				numMergedColumns := len(columnBalanceMap)
				maxLengthSplitAcrossColumns := mergedColumnLength / numMergedColumns
				mapReduced := false
				for mergedColumn := range columnBalanceMap {
					maxColumnLength := t.maxColumnLengths[mergedColumn]
					if maxColumnLength >= maxLengthSplitAcrossColumns {
						mapReduced = true
						mergedColumnLength -= maxColumnLength
						delete(columnBalanceMap, mergedColumn)
					}
				}
				if !mapReduced {
					break
				}
			}

			// act on any remaining columns that need balancing
			if len(columnBalanceMap) > 0 {
				// remove the max column sizes from the remaining amount to balance, then
				// share out the remainder amongst the columns.
				numRebalancedColumns := len(columnBalanceMap)
				balanceColumns := make([]int, 0, numRebalancedColumns)
				for balanceColumn := range columnBalanceMap {
					mergedColumnLength -= t.maxColumnLengths[balanceColumn]
					balanceColumns = append(balanceColumns, balanceColumn)
				}
				// pad out the columns one by one
				sort.Ints(balanceColumns)
				columnLengthRemaining := mergedColumnLength
				columnsRemaining := numRebalancedColumns
				for index := 0; index < numRebalancedColumns; index++ {
					balancedSpace := columnLengthRemaining / columnsRemaining
					balanceColumn := balanceColumns[index]
					t.maxColumnLengths[balanceColumn] += balancedSpace
					columnLengthRemaining -= balancedSpace
					columnsRemaining--
				}
			}
		}
	}
}

func (t *Table) initForRender() {
	// pick a default style if none was set until now
	t.Style()

	// reset rendering state
	t.reset()

	// initialize the column configs and normalize them
	t.initForRenderColumnConfigs()

	// initialize and stringify all the raw rows
	t.initForRenderRows()

	// find the longest continuous line in each column
	t.initForRenderColumnLengths()
	t.initForRenderMaxRowLength()
	t.initForRenderPaddedColumns()

	// generate a separator row and calculate maximum row length
	t.initForRenderRowSeparator()

	// reset the counter for the number of lines rendered
	t.numLinesRendered = 0
}

func (t *Table) initForRenderColumnConfigs() {
	t.columnConfigMap = map[int]ColumnConfig{}
	for _, colCfg := range t.columnConfigs {
		// find the column number if none provided; this logic can work only if
		// a header row is present and has a column with the given name
		if colCfg.Number == 0 {
			for _, row := range t.rowsHeaderRaw {
				colCfg.Number = row.findColumnNumber(colCfg.Name)
				if colCfg.Number > 0 {
					break
				}
			}
		}
		if colCfg.Number > 0 {
			t.columnConfigMap[colCfg.Number-1] = colCfg
		}
	}
}

func (t *Table) initForRenderColumnLengths() {
	t.maxColumnLengths = make([]int, t.numColumns)
	t.maxMergedColumnLengths = make(map[int]map[int]int)
	t.extractMaxColumnLengths(t.rowsHeader, renderHint{isHeaderRow: true})
	t.extractMaxColumnLengths(t.rows, renderHint{})
	t.extractMaxColumnLengths(t.rowsFooter, renderHint{isFooterRow: true})

	// increase the column lengths if any are under the limits
	for colIdx := range t.maxColumnLengths {
		minWidth := t.getColumnWidthMin(colIdx)
		if minWidth > 0 && t.maxColumnLengths[colIdx] < minWidth {
			t.maxColumnLengths[colIdx] = minWidth
		}
	}
	t.reBalanceMaxMergedColumnLengths()
}

func (t *Table) initForRenderHideColumns() {
	if !t.hasHiddenColumns() {
		return
	}
	colIdxMap := t.hideColumns()

	// re-create columnIsNonNumeric with new column indices
	columnIsNonNumeric := make([]bool, t.numColumns)
	for oldColIdx, nonNumeric := range t.columnIsNonNumeric {
		if newColIdx, ok := colIdxMap[oldColIdx]; ok {
			columnIsNonNumeric[newColIdx] = nonNumeric
		}
	}
	t.columnIsNonNumeric = columnIsNonNumeric

	// re-create columnConfigMap with new column indices
	columnConfigMap := make(map[int]ColumnConfig)
	for oldColIdx, cc := range t.columnConfigMap {
		if newColIdx, ok := colIdxMap[oldColIdx]; ok {
			columnConfigMap[newColIdx] = cc
		}
	}
	t.columnConfigMap = columnConfigMap
}

func (t *Table) initForRenderMaxRowLength() {
	t.maxRowLength = 0
	if t.autoIndex {
		t.maxRowLength += text.StringWidthWithoutEscSequences(t.style.Box.PaddingLeft)
		t.maxRowLength += len(fmt.Sprint(len(t.rows)))
		t.maxRowLength += text.StringWidthWithoutEscSequences(t.style.Box.PaddingRight)
		if t.style.Options.SeparateColumns {
			t.maxRowLength += text.StringWidthWithoutEscSequences(t.style.Box.MiddleSeparator)
		}
	}
	if t.style.Options.SeparateColumns {
		t.maxRowLength += text.StringWidthWithoutEscSequences(t.style.Box.MiddleSeparator) * (t.numColumns - 1)
	}
	for _, maxColumnLength := range t.maxColumnLengths {
		maxColumnLength += text.StringWidthWithoutEscSequences(t.style.Box.PaddingLeft + t.style.Box.PaddingRight)
		t.maxRowLength += maxColumnLength
	}
	if t.style.Options.DrawBorder {
		t.maxRowLength += text.StringWidthWithoutEscSequences(t.style.Box.Left + t.style.Box.Right)
	}
}

func (t *Table) initForRenderPaddedColumns() {
	paddingSize := t.style.Size.WidthMin - t.maxRowLength
	for paddingSize > 0 {
		// distribute padding equally among all columns
		numColumnsPadded := 0
		for colIdx := 0; paddingSize > 0 && colIdx < t.numColumns; colIdx++ {
			colWidthMax := t.getColumnWidthMax(colIdx)
			if colWidthMax == 0 || t.maxColumnLengths[colIdx] < colWidthMax {
				t.maxColumnLengths[colIdx]++
				numColumnsPadded++
				paddingSize--
			}
		}

		// avoid endless looping because all columns are at max size and cannot
		// be expanded any further
		if numColumnsPadded == 0 {
			break
		}
	}
}

func (t *Table) initForRenderRows() {
	// auto-index: calc the index column's max length
	t.autoIndexVIndexMaxLength = len(fmt.Sprint(len(t.rowsRaw)))

	// stringify all the rows to make it easy to render
	t.rows = t.initForRenderRowsStringify(t.rowsRaw, renderHint{})
	t.rowsFooter = t.initForRenderRowsStringify(t.rowsFooterRaw, renderHint{isFooterRow: true})
	t.rowsHeader = t.initForRenderRowsStringify(t.rowsHeaderRaw, renderHint{isHeaderRow: true})

	// sort the rows as requested
	t.initForRenderSortRows()

	// find the row colors (if any)
	t.initForRenderRowPainterColors()

	// suppress columns without any content
	t.initForRenderSuppressColumns()

	// strip out hidden columns
	t.initForRenderHideColumns()
}

func (t *Table) initForRenderRowsStringify(rows []Row, hint renderHint) []rowStr {
	rowsStr := make([]rowStr, len(rows))
	for idx, row := range rows {
		hint.rowNumber = idx + 1
		rowsStr[idx] = t.analyzeAndStringify(row, hint)
	}
	return rowsStr
}

func (t *Table) initForRenderRowPainterColors() {
	if !t.hasRowPainter() {
		return
	}

	// generate the colors
	t.rowsColors = make([]text.Colors, len(t.rowsRaw))
	for idx, row := range t.rowsRaw {
		idxColors := idx
		if len(t.sortedRowIndices) > 0 {
			// override with the sorted row index
			for j := 0; j < len(t.sortedRowIndices); j++ {
				if t.sortedRowIndices[j] == idx {
					idxColors = j
					break
				}
			}
		}

		if t.rowPainter != nil {
			t.rowsColors[idxColors] = t.rowPainter(row)
		} else if t.rowPainterWithAttributes != nil {
			t.rowsColors[idxColors] = t.rowPainterWithAttributes(row, RowAttributes{
				Number:       idx + 1,
				NumberSorted: idxColors + 1,
			})
		}
	}
}

func (t *Table) initForRenderRowSeparator() {
	t.rowSeparator = make(rowStr, t.numColumns)
	for colIdx, maxColumnLength := range t.maxColumnLengths {
		maxColumnLength += text.StringWidthWithoutEscSequences(t.style.Box.PaddingLeft + t.style.Box.PaddingRight)
		t.rowSeparator[colIdx] = text.RepeatAndTrim(t.style.Box.MiddleHorizontal, maxColumnLength)
	}
}

func (t *Table) initForRenderSortRows() {
	if len(t.sortBy) == 0 {
		return
	}

	// sort the rows
	t.sortedRowIndices = t.getSortedRowIndices()
	sortedRows := make([]rowStr, len(t.rows))
	for idx := range t.rows {
		sortedRows[idx] = t.rows[t.sortedRowIndices[idx]]
	}
	t.rows = sortedRows
}

func (t *Table) initForRenderSuppressColumns() {
	shouldSuppressColumn := func(colIdx int) bool {
		for _, row := range t.rows {
			if colIdx < len(row) && row[colIdx] != "" {
				// Columns may contain non-printable characters. For example
				// the text.Direction modifiers. These should not be considered
				// when deciding to suppress a column.
				for _, r := range row[colIdx] {
					if unicode.IsPrint(r) {
						return false
					}
				}
				return true
			}
		}
		return true
	}

	if t.suppressEmptyColumns {
		for colIdx := 0; colIdx < t.numColumns; colIdx++ {
			if shouldSuppressColumn(colIdx) {
				cc := t.columnConfigMap[colIdx]
				cc.Hidden = true
				t.columnConfigMap[colIdx] = cc
			}
		}
	}
}

// reset initializes all the variables used to maintain rendering information
// that are written to in this file
func (t *Table) reset() {
	t.autoIndexVIndexMaxLength = 0
	t.columnConfigMap = nil
	t.columnIsNonNumeric = nil
	t.firstRowOfPage = true
	t.maxColumnLengths = nil
	t.maxRowLength = 0
	t.numColumns = 0
	t.numLinesRendered = 0
	t.rowSeparator = nil
	t.rows = nil
	t.rowsColors = nil
	t.rowsFooter = nil
	t.rowsHeader = nil
}
