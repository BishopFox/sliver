package table

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FilterBy defines what to filter (Column Name or Number), how to filter (Operator),
// and the value to compare against.
type FilterBy struct {
	// Name is the name of the Column as it appears in the first Header row.
	// If a Header is not provided, or the name is not found in the header, this
	// will not work.
	Name string
	// Number is the Column # from left. When specified, it overrides the Name
	// property. If you know the exact Column number, use this instead of Name.
	Number int

	// Operator defines how to compare the column value against the Value.
	Operator FilterOperator

	// Value is the value to compare against. The type should match the expected
	// comparison type (string for string operations, numeric for numeric operations).
	// For Contains, StartsWith, EndsWith, and RegexMatch, Value should be a string.
	// For numeric comparisons (Equal, NotEqual, GreaterThan, etc.), Value can be
	// a number (int, float64) or a string representation of a number.
	Value interface{}

	// IgnoreCase makes string comparisons case-insensitive (only applies to
	// string-based operators).
	IgnoreCase bool

	// CustomFilter is a function that can be used to filter rows in a custom
	// manner. Note that:
	// * This overrides and ignores the Operator, Value, and IgnoreCase settings
	// * This is called after the column contents are converted to string form
	// * This function is expected to return:
	//   * true => include the row
	//   * false => exclude the row
	//
	// Use this when the default filtering logic is not sufficient.
	CustomFilter func(cellValue string) bool
}

// FilterOperator defines how to filter.
type FilterOperator int

const (
	// Equal filters rows where the column value equals the Value.
	Equal FilterOperator = iota
	// NotEqual filters rows where the column value does not equal the Value.
	NotEqual
	// GreaterThan filters rows where the column value is greater than the Value.
	GreaterThan
	// GreaterThanOrEqual filters rows where the column value is greater than or equal to the Value.
	GreaterThanOrEqual
	// LessThan filters rows where the column value is less than the Value.
	LessThan
	// LessThanOrEqual filters rows where the column value is less than or equal to the Value.
	LessThanOrEqual
	// Contains filters rows where the column value contains the Value (string search).
	Contains
	// NotContains filters rows where the column value does not contain the Value (string search).
	NotContains
	// StartsWith filters rows where the column value starts with the Value.
	StartsWith
	// EndsWith filters rows where the column value ends with the Value.
	EndsWith
	// RegexMatch filters rows where the column value matches the Value as a regular expression.
	RegexMatch
	// RegexNotMatch filters rows where the column value does not match the Value as a regular expression.
	RegexNotMatch
)

func (t *Table) parseFilterBy(filterBy []FilterBy) []FilterBy {
	var resFilterBy []FilterBy
	for _, filter := range filterBy {
		colNum := 0
		if filter.Number > 0 && filter.Number <= t.numColumns {
			colNum = filter.Number
		} else if filter.Name != "" && len(t.rowsHeaderRaw) > 0 {
			// Parse from raw header rows
			for idx, colName := range t.rowsHeaderRaw[0] {
				if fmt.Sprint(colName) == filter.Name {
					colNum = idx + 1
					break
				}
			}
		}
		if colNum > 0 {
			resFilterBy = append(resFilterBy, FilterBy{
				Name:         filter.Name,
				Number:       colNum,
				Operator:     filter.Operator,
				Value:        filter.Value,
				IgnoreCase:   filter.IgnoreCase,
				CustomFilter: filter.CustomFilter,
			})
		}
	}
	return resFilterBy
}

func (t *Table) matchesFiltersRaw(row Row, filters []FilterBy) bool {
	// All filters must match (AND logic)
	for _, filter := range filters {
		if !t.matchesFilterRaw(row, filter) {
			return false
		}
	}
	return true
}

func (t *Table) matchesFilterRaw(row Row, filter FilterBy) bool {
	colIdx := filter.Number - 1
	if colIdx < 0 || colIdx >= len(row) {
		return false
	}

	cellValue := row[colIdx]
	cellValueStr := fmt.Sprint(cellValue)

	// Use custom filter if provided
	if filter.CustomFilter != nil {
		return filter.CustomFilter(cellValueStr)
	}

	// Use operator-based filtering
	return t.matchesOperator(cellValueStr, filter)
}

func (t *Table) matchesOperator(cellValue string, filter FilterBy) bool {
	switch filter.Operator {
	case Equal:
		return t.compareEqual(cellValue, filter.Value, filter.IgnoreCase)
	case NotEqual:
		return !t.compareEqual(cellValue, filter.Value, filter.IgnoreCase)
	case GreaterThan:
		return t.compareNumeric(cellValue, filter.Value, func(a, b float64) bool { return a > b })
	case GreaterThanOrEqual:
		return t.compareNumeric(cellValue, filter.Value, func(a, b float64) bool { return a >= b })
	case LessThan:
		return t.compareNumeric(cellValue, filter.Value, func(a, b float64) bool { return a < b })
	case LessThanOrEqual:
		return t.compareNumeric(cellValue, filter.Value, func(a, b float64) bool { return a <= b })
	case Contains:
		return t.compareContains(cellValue, filter.Value, filter.IgnoreCase)
	case NotContains:
		return !t.compareContains(cellValue, filter.Value, filter.IgnoreCase)
	case StartsWith:
		return t.compareStartsWith(cellValue, filter.Value, filter.IgnoreCase)
	case EndsWith:
		return t.compareEndsWith(cellValue, filter.Value, filter.IgnoreCase)
	case RegexMatch:
		return t.compareRegexMatch(cellValue, filter.Value, filter.IgnoreCase)
	case RegexNotMatch:
		return !t.compareRegexMatch(cellValue, filter.Value, filter.IgnoreCase)
	default:
		return false
	}
}

func (t *Table) compareEqual(cellValue string, filterValue interface{}, ignoreCase bool) bool {
	filterStr := fmt.Sprint(filterValue)
	if ignoreCase {
		return strings.EqualFold(cellValue, filterStr)
	}
	return cellValue == filterStr
}

func (t *Table) compareNumeric(cellValue string, filterValue interface{}, compareFunc func(float64, float64) bool) bool {
	cellNum, cellErr := strconv.ParseFloat(cellValue, 64)
	if cellErr != nil {
		return false
	}

	var filterNum float64
	switch v := filterValue.(type) {
	case int:
		filterNum = float64(v)
	case int64:
		filterNum = float64(v)
	case float64:
		filterNum = v
	case float32:
		filterNum = float64(v)
	case string:
		var err error
		filterNum, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return false
		}
	default:
		// Try to convert to string and parse
		filterStr := fmt.Sprint(filterValue)
		var err error
		filterNum, err = strconv.ParseFloat(filterStr, 64)
		if err != nil {
			return false
		}
	}

	return compareFunc(cellNum, filterNum)
}

func (t *Table) compareContains(cellValue string, filterValue interface{}, ignoreCase bool) bool {
	filterStr := fmt.Sprint(filterValue)
	if ignoreCase {
		return strings.Contains(strings.ToLower(cellValue), strings.ToLower(filterStr))
	}
	return strings.Contains(cellValue, filterStr)
}

func (t *Table) compareStartsWith(cellValue string, filterValue interface{}, ignoreCase bool) bool {
	filterStr := fmt.Sprint(filterValue)
	if ignoreCase {
		return strings.HasPrefix(strings.ToLower(cellValue), strings.ToLower(filterStr))
	}
	return strings.HasPrefix(cellValue, filterStr)
}

func (t *Table) compareEndsWith(cellValue string, filterValue interface{}, ignoreCase bool) bool {
	filterStr := fmt.Sprint(filterValue)
	if ignoreCase {
		return strings.HasSuffix(strings.ToLower(cellValue), strings.ToLower(filterStr))
	}
	return strings.HasSuffix(cellValue, filterStr)
}

func (t *Table) compareRegexMatch(cellValue string, filterValue interface{}, ignoreCase bool) bool {
	filterStr := fmt.Sprint(filterValue)

	// Compile the regex pattern
	var pattern *regexp.Regexp
	var err error
	if ignoreCase {
		pattern, err = regexp.Compile("(?i)" + filterStr)
	} else {
		pattern, err = regexp.Compile(filterStr)
	}

	if err != nil {
		// If regex compilation fails, fall back to simple string matching
		return t.compareEqual(cellValue, filterValue, ignoreCase)
	}

	return pattern.MatchString(cellValue)
}
