package table

import (
	"sort"
	"strconv"
	"strings"
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

	// IgnoreCase makes sorting case-insensitive
	IgnoreCase bool

	// CustomLess is a function that can be used to sort the column in a custom
	// manner. Note that:
	// * This overrides and ignores the Mode and IgnoreCase settings
	// * This is called after the column contents are converted to string form
	// * This function is expected to return:
	//   * -1 => when iStr comes before jStr
	//   *  0 => when iStr and jStr are considered equal
	//   *  1 => when iStr comes after jStr
	//
	// Use this when the default sorting logic is not sufficient.
	CustomLess func(iStr string, jStr string) int
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

// getSortedRowIndices sorts and returns the row indices in Sorted order as
// directed by Table.sortBy which can be set using Table.SortBy(...)
func (t *Table) getSortedRowIndices() []int {
	sortedIndices := make([]int, len(t.rows))
	for idx := range t.rows {
		sortedIndices[idx] = idx
	}

	if len(t.sortBy) > 0 {
		parsedSortBy := t.parseSortBy(t.sortBy)
		sort.Slice(sortedIndices, func(i, j int) bool {
			isEqual, isLess := false, false
			realI, realJ := sortedIndices[i], sortedIndices[j]
			for _, sortBy := range parsedSortBy {
				// extract the values/cells from the rows for comparison
				rowI, rowJ, colIdx := t.rows[realI], t.rows[realJ], sortBy.Number-1
				iVal, jVal := "", ""
				if colIdx < len(rowI) {
					iVal = rowI[colIdx]
				}
				if colIdx < len(rowJ) {
					jVal = rowJ[colIdx]
				}

				// compare and choose whether to continue
				isEqual, isLess = less(iVal, jVal, sortBy)
				// if the values are not equal, return the result immediately
				if !isEqual {
					return isLess
				}
				// if the values are equal, continue to the next column
			}
			return isLess
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
				Name:       col.Name,
				Number:     colNum,
				Mode:       col.Mode,
				IgnoreCase: col.IgnoreCase,
				CustomLess: col.CustomLess,
			})
		}
	}
	return resSortBy
}

func less(iVal string, jVal string, sb SortBy) (bool, bool) {
	if sb.CustomLess != nil {
		// use the custom less function to compare the values
		rc := sb.CustomLess(iVal, jVal)
		if rc < 0 {
			return false, true
		} else if rc > 0 {
			return false, false
		} else { // rc == 0
			return true, false
		}
	}

	// if the values are equal, return fast to continue to next column
	if iVal == jVal {
		return true, false
	}

	// otherwise, use the default sorting logic defined by Mode and IgnoreCase
	switch sb.Mode {
	case Asc, Dsc:
		return lessAlphabetic(iVal, jVal, sb)
	case AscNumeric, DscNumeric:
		return lessNumeric(iVal, jVal, sb)
	default: // AscAlphaNumeric, AscNumericAlpha, DscAlphaNumeric, DscNumericAlpha
		return lessMixedMode(iVal, jVal, sb)
	}
}

func lessAlphabetic(iVal string, jVal string, sb SortBy) (bool, bool) {
	if sb.IgnoreCase {
		iLow := strings.ToLower(iVal)
		jLow := strings.ToLower(jVal)
		// when two strings are case-insensitive identical, compare them casesensitive.
		// That makes sure to get a consistent sorting
		identical := iLow == jLow
		switch sb.Mode {
		case Asc, AscAlphaNumeric, AscNumericAlpha:
			return identical, (identical && iVal < jVal) || iLow < jLow
		default: // Dsc, DscAlphaNumeric, DscNumericAlpha
			return identical, (identical && iVal > jVal) || iLow > jLow
		}
	}
	switch sb.Mode {
	case Asc, AscAlphaNumeric, AscNumericAlpha:
		return false, iVal < jVal
	default: // Dsc, DscAlphaNumeric, DscNumericAlpha
		return false, iVal > jVal
	}
}

func lessMixedMode(iVal string, jVal string, sb SortBy) (bool, bool) {
	iNumVal, iErr := strconv.ParseFloat(iVal, 64)
	jNumVal, jErr := strconv.ParseFloat(jVal, 64)
	if iErr != nil && jErr != nil { // both are alphanumeric
		return lessAlphabetic(iVal, jVal, sb)
	}
	if iErr != nil { // iVal == "abc"; jVal == 5
		switch sb.Mode {
		case AscAlphaNumeric, DscAlphaNumeric:
			return false, true
		default: // AscNumericAlpha, DscNumericAlpha
			return false, false
		}
	}
	if jErr != nil { // iVal == 5; jVal	== "abc"
		switch sb.Mode {
		case AscAlphaNumeric, DscAlphaNumeric:
			return false, false
		default: // AscNumericAlpha, DscNumericAlpha:
			return false, true
		}
	}
	// both values numeric
	return lessNumericVal(iNumVal, jNumVal, sb)
}

func lessNumeric(iVal string, jVal string, sb SortBy) (bool, bool) {
	iNumVal, iErr := strconv.ParseFloat(iVal, 64)
	jNumVal, jErr := strconv.ParseFloat(jVal, 64)
	if iErr != nil || jErr != nil {
		return false, false
	}

	return lessNumericVal(iNumVal, jNumVal, sb)
}

func lessNumericVal(iVal float64, jVal float64, sb SortBy) (bool, bool) {
	if iVal == jVal {
		return true, false
	}

	switch sb.Mode {
	case AscNumeric, AscAlphaNumeric, AscNumericAlpha:
		return false, iVal < jVal
	default: // DscNumeric, DscAlphaNumeric, DscNumericAlpha
		return false, iVal > jVal
	}
}
