// Package v1 provides MessageFormat v1 runtime functions
// TypeScript original code: /packages/runtime/src/runtime.ts
package v1

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Safe integer range constants (JavaScript Number.MAX_SAFE_INTEGER/MIN_SAFE_INTEGER)
// Integers outside this range may lose precision in JavaScript environments
const (
	maxSafeInteger = 1e15
	minSafeInteger = -1e15
)

// Number formatters cache with sync.Map for better performance
// TypeScript original code:
// const _nf: Record<string, Intl.NumberFormat> = {};
//
//	function _nf(lc: string): Intl.NumberFormat {
//	  return _nf[lc] || (_nf[lc] = new Intl.NumberFormat(lc));
//	}

// Number formats a number with locale-specific formatting and offset using golang.org/x/text/number
// TypeScript original code:
//
//	export function number(lc: string, value: number, offset: number) {
//	  return _nf(lc).format(value - offset);
//	}
func Number(lc string, value float64, offset float64) string {
	result := value - offset

	tag, err := language.Parse(lc)
	if err != nil {
		tag = language.English
	}

	// Use locale-specific formatting
	printer := message.NewPrinter(tag)
	if result == math.Trunc(result) && result >= minSafeInteger && result <= maxSafeInteger {
		return printer.Sprintf("%.0f", result)
	}
	return printer.Sprintf("%.10g", result)
}

// StrictNumber provides strict number formatting with error checking
// TypeScript original code:
// export function strictNumber(
//
//	lc: string,
//	value: number,
//	offset: number,
//	name: string
//
//	) {
//	  const n = value - offset;
//	  if (isNaN(n)) throw new Error('`' + name + '` or its offset is not a number');
//	  return _nf(lc).format(n);
//	}
func StrictNumber(lc string, value any, offset float64, name string) (string, error) {
	numValue, err := toFloat64(value)
	if err != nil {
		return "", WrapMissingParameter("`" + name + "` or its offset is not a number")
	}

	result := numValue - offset
	if math.IsNaN(result) {
		return "", WrapMissingParameter("`" + name + "` or its offset is not a number")
	}

	return Number(lc, numValue, offset), nil
}

// Plural handles plural form selection based on value and plural rules
// TypeScript original code:
// export function plural(
//
//	value: number,
//	offset: number,
//	lcfunc: (value: number, isOrdinal?: boolean) => string,
//	data: { [key: string]: unknown },
//	isOrdinal?: boolean
//
//	) {
//	  if ({}.hasOwnProperty.call(data, value)) return data[value];
//	  if (offset) value -= offset;
//	  const key = lcfunc(value, isOrdinal);
//	  return key in data ? data[key] : data.other;
//	}
func Plural(value any, offset float64, lcfunc PluralFunction, data map[string]any, isOrdinal ...bool) any {
	// Convert value to number
	numValue, err := toFloat64(value)
	if err != nil {
		if other, exists := data["other"]; exists {
			return other
		}
		return ""
	}

	// Check exact match first (like =0, =1, =2, etc.) - TypeScript behavior
	exactKey := formatExactKey(numValue)
	if exactValue, exists := data[exactKey]; exists {
		return exactValue
	}

	// Apply offset if provided
	adjustedValue := numValue
	if offset != 0 {
		adjustedValue = numValue - offset
	}

	// TypeScript behavior: non-integer values (with decimal part) should use "other"
	// This handles cases like "1.0" which should be treated as "other", not "one"
	if adjustedValue != math.Trunc(adjustedValue) {
		if other, exists := data["other"]; exists {
			return other
		}
		return ""
	}

	// Also check if the original value was a string representation of a decimal
	// e.g., "1.0" should be treated as decimal even if it converts to 1.0
	if valueStr, ok := value.(string); ok {
		if strings.Contains(valueStr, ".") && valueStr != strings.TrimRight(strings.TrimRight(valueStr, "0"), ".") {
			if other, exists := data["other"]; exists {
				return other
			}
			return ""
		}
	}

	// Determine if this is ordinal plural
	ordinal := len(isOrdinal) > 0 && isOrdinal[0]

	// Get plural category using the locale function
	category, err := lcfunc(adjustedValue, ordinal)
	if err != nil {
		if other, exists := data["other"]; exists {
			return other
		}
		return ""
	}

	// Return the value for the category, or "other" as fallback
	if categoryValue, exists := data[string(category)]; exists {
		return categoryValue
	}

	if other, exists := data["other"]; exists {
		return other
	}

	return ""
}

// SelectValue handles select statement processing
// TypeScript original code:
//
//	export function select(value: string, data: { [key: string]: unknown }) {
//	  return {}.hasOwnProperty.call(data, value) ? data[value] : data.other;
//	}
func SelectValue(value string, data map[string]any) any {
	if selectedValue, exists := data[value]; exists {
		return selectedValue
	}

	if other, exists := data["other"]; exists {
		return other
	}

	return ""
}

// ReqArgs validates that all required arguments are present
// TypeScript original code:
//
//	export function reqArgs(keys: string[], data: { [key: string]: unknown }) {
//	  for (let i = 0; i < keys.length; ++i) {
//	    if (!data || data[keys[i]] === undefined) {
//	      throw new Error(`Message requires argument '${keys[i]}'`);
//	    }
//	  }
//	}
func ReqArgs(keys []string, data map[string]any) error {
	for _, key := range keys {
		if data == nil {
			return WrapMissingArgument(key)
		}
		if _, exists := data[key]; !exists {
			return WrapMissingArgument(key)
		}
	}
	return nil
}

// Helper functions

// toFloat64 converts various numeric types to float64
func toFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, WrapInvalidType(fmt.Sprintf("cannot convert %T to float64", value))
	}
}

// formatExactKey formats a number as an exact key (e.g., "=1", "=0")
func formatExactKey(value float64) string {
	if value == math.Trunc(value) && value >= minSafeInteger && value <= maxSafeInteger {
		return fmt.Sprintf("=%.0f", value)
	}
	return fmt.Sprintf("=%g", value)
}

// ReplaceOctothorpe replaces # symbols with formatted numbers in plural contexts
// TypeScript original code (from compiler.ts):
// const rep = token.type === 'plural' ? token : pluralToken;
// if (rep) res = res.replace(/(^|[^\\])#/g, `$1${this.numbr(rep.arg)}`);
func ReplaceOctothorpe(content string, argValue any, locale string, offset float64) string {
	if content == "" {
		return content
	}

	numValue, err := toFloat64(argValue)
	if err != nil {
		return content
	}

	formattedNumber := Number(locale, numValue, offset)

	// Replace only the special __OCTOTHORPE__ placeholders, not literal # characters
	// This ensures that quoted '#' from ContentTokens remain as literal #
	return strings.ReplaceAll(content, "__OCTOTHORPE__", formattedNumber)
}

// ProcessPluralContent processes plural case content with octothorpe replacement
func ProcessPluralContent(content any, argValue any, locale string, offset float64) string {
	contentStr := fmt.Sprintf("%v", content)
	return ReplaceOctothorpe(contentStr, argValue, locale, offset)
}
