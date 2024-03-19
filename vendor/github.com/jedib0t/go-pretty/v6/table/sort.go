package table

import (
	"sort"
	"strconv"
)

// SortBy defines What to sort (Column Name or Number), and How to sort (Mode).
type SortBy struct {
	// Name is the name of the Column as it appears in the first Header row.
	// If a Header is not provided, or the name is not found in the header, this
	// will not work.
	Name string
	// Number is the Column # from left. When specified, it overrides the Name
	// property. If you know the exact Column number, use this instead of Name.
	Number int

	// Mode tells the Writer how to Sort. Asc/Dsc/etc.
	Mode SortMode
}

// SortMode defines How to sort.
type SortMode int

const (
	// Asc sorts the column in Ascending order alphabetically.
	Asc SortMode = iota
	// AscAlphaNumeric sorts the column in Ascending order alphabetically and
	// then numerically.
	AscAlphaNumeric
	// AscNumeric sorts the column in Ascending order numerically.
	AscNumeric
	// AscNumericAlpha sorts the column in Ascending order numerically and
	// then alphabetically.
	AscNumericAlpha
	// Dsc sorts the column in Descending order alphabetically.
	Dsc
	// DscAlphaNumeric sorts the column in Descending order alphabetically and
	// then numerically.
	DscAlphaNumeric
	// DscNumeric sorts the column in Descending order numerically.
	DscNumeric
	// DscNumericAlpha sorts the column in Descending order numerically and
	// then alphabetically.
	DscNumericAlpha
)

type rowsSorter struct {
	rows          []rowStr
	sortBy        []SortBy
	sortedIndices []int
}

// getSortedRowIndices sorts and returns the row indices in Sorted order as
// directed by Table.sortBy which can be set using Table.SortBy(...)
func (t *Table) getSortedRowIndices() []int {
	sortedIndices := make([]int, len(t.rows))
	for idx := range t.rows {
		sortedIndices[idx] = idx
	}

	if t.sortBy != nil && len(t.sortBy) > 0 {
		sort.Sort(rowsSorter{
			rows:          t.rows,
			sortBy:        t.parseSortBy(t.sortBy),
			sortedIndices: sortedIndices,
		})
	}

	return sortedIndices
}

func (t *Table) parseSortBy(sortBy []SortBy) []SortBy {
	var resSortBy []SortBy
	for _, col := range sortBy {
		colNum := 0
		if col.Number > 0 && col.Number <= t.numColumns {
			colNum = col.Number
		} else if col.Name != "" && len(t.rowsHeader) > 0 {
			for idx, colName := range t.rowsHeader[0] {
				if col.Name == colName {
					colNum = idx + 1
					break
				}
			}
		}
		if colNum > 0 {
			resSortBy = append(resSortBy, SortBy{
				Name:   col.Name,
				Number: colNum,
				Mode:   col.Mode,
			})
		}
	}
	return resSortBy
}

func (rs rowsSorter) Len() int {
	return len(rs.rows)
}

func (rs rowsSorter) Swap(i, j int) {
	rs.sortedIndices[i], rs.sortedIndices[j] = rs.sortedIndices[j], rs.sortedIndices[i]
}

func (rs rowsSorter) Less(i, j int) bool {
	realI, realJ := rs.sortedIndices[i], rs.sortedIndices[j]
	for _, sortBy := range rs.sortBy {
		// extract the values/cells from the rows for comparison
		rowI, rowJ, colIdx := rs.rows[realI], rs.rows[realJ], sortBy.Number-1
		iVal, jVal := "", ""
		if colIdx < len(rowI) {
			iVal = rowI[colIdx]
		}
		if colIdx < len(rowJ) {
			jVal = rowJ[colIdx]
		}

		// compare and choose whether to continue
		shouldContinue, returnValue := less(iVal, jVal, sortBy.Mode)
		if !shouldContinue {
			return returnValue
		}
	}
	return false
}

func less(iVal string, jVal string, mode SortMode) (bool, bool) {
	if iVal == jVal {
		return true, false
	}

	switch mode {
	case Asc, Dsc:
		return lessAlphabetic(iVal, jVal, mode)
	case AscNumeric, DscNumeric:
		return lessNumeric(iVal, jVal, mode)
	default: // AscAlphaNumeric, AscNumericAlpha, DscAlphaNumeric, DscNumericAlpha
		return lessMixedMode(iVal, jVal, mode)
	}
}

func lessAlphabetic(iVal string, jVal string, mode SortMode) (bool, bool) {
	switch mode {
	case Asc, AscAlphaNumeric, AscNumericAlpha:
		return false, iVal < jVal
	default: // Dsc, DscAlphaNumeric, DscNumericAlpha
		return false, iVal > jVal
	}
}

func lessAlphaNumericI(mode SortMode) (bool, bool) {
	// i == "abc"; j == 5
	switch mode {
	case AscAlphaNumeric, DscAlphaNumeric:
		return false, true
	default: // AscNumericAlpha, DscNumericAlpha
		return false, false
	}
}

func lessAlphaNumericJ(mode SortMode) (bool, bool) {
	// i == 5; j == "abc"
	switch mode {
	case AscAlphaNumeric, DscAlphaNumeric:
		return false, false
	default: // AscNumericAlpha, DscNumericAlpha:
		return false, true
	}
}

func lessMixedMode(iVal string, jVal string, mode SortMode) (bool, bool) {
	iNumVal, iErr := strconv.ParseFloat(iVal, 64)
	jNumVal, jErr := strconv.ParseFloat(jVal, 64)
	if iErr != nil && jErr != nil { // both are alphanumeric
		return lessAlphabetic(iVal, jVal, mode)
	}
	if iErr != nil { // iVal is alphabetic, jVal is numeric
		return lessAlphaNumericI(mode)
	}
	if jErr != nil { // iVal is numeric, jVal is alphabetic
		return lessAlphaNumericJ(mode)
	}
	// both values numeric
	return lessNumericVal(iNumVal, jNumVal, mode)
}

func lessNumeric(iVal string, jVal string, mode SortMode) (bool, bool) {
	iNumVal, iErr := strconv.ParseFloat(iVal, 64)
	jNumVal, jErr := strconv.ParseFloat(jVal, 64)
	if iErr != nil || jErr != nil {
		return false, false
	}

	return lessNumericVal(iNumVal, jNumVal, mode)
}

func lessNumericVal(iVal float64, jVal float64, mode SortMode) (bool, bool) {
	if iVal == jVal {
		return true, false
	}

	switch mode {
	case AscNumeric, AscAlphaNumeric, AscNumericAlpha:
		return false, iVal < jVal
	default: // DscNumeric, DscAlphaNumeric, DscNumericAlpha
		return false, iVal > jVal
	}
}
