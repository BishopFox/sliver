package jsonschema

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-json-experiment/json"
)

// evaluateUniqueItems checks if all elements in the array are unique when the "uniqueItems" property is set to true.
// According to the JSON Schema Draft 2020-12:
//   - If "uniqueItems" is false, the data always validates successfully.
//   - If "uniqueItems" is true, the data validates successfully only if all elements in the array are unique.
//
// This function only applies when the data is an array and "uniqueItems" is true.
//
// This method ensures that the array elements conform to the uniqueness constraints defined in the schema.
// If the uniqueness constraint is violated, it returns a EvaluationError detailing the issue.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-uniqueitems
func evaluateUniqueItems(schema *Schema, data []any) *EvaluationError {
	// If uniqueItems is false or not set, no validation is needed
	if schema.UniqueItems == nil || !*schema.UniqueItems {
		return nil
	}

	// Determine the array length to validate
	maxLength := len(data)

	// If items is false, only validate items defined by prefixItems
	if schema.Items != nil && schema.Items.Boolean != nil && !*schema.Items.Boolean {
		if schema.PrefixItems != nil {
			maxLength = len(schema.PrefixItems)
			if maxLength > len(data) {
				maxLength = len(data)
			}
		} else {
			maxLength = 0
		}
	}

	// If there are no items to validate, return immediately
	if maxLength == 0 {
		return nil
	}

	// Use a map to track the index of each item
	seen := make(map[string][]int)
	for index, item := range data[:maxLength] {
		itemKey, err := normalizeForComparison(item)
		if err != nil {
			return NewEvaluationError("uniqueItems", "item_normalization_error", "Error normalizing item at index {index}", map[string]any{
				"index": fmt.Sprint(index),
			})
		}
		seen[itemKey] = append(seen[itemKey], index)
	}

	// Prepare to report all duplicate item positions
	var duplicates []string
	for _, indices := range seen {
		if len(indices) > 1 {
			// Convert to 1-based indices for more user-friendly output
			for i := range indices {
				indices[i]++
			}
			duplicates = append(duplicates, fmt.Sprintf("(%s)", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(indices)), ", "), "[]")))
		}
	}

	if len(duplicates) > 0 {
		return NewEvaluationError("uniqueItems", "unique_items_mismatch", "Found duplicates at the following index groups: {duplicates}", map[string]any{
			"duplicates": strings.Join(duplicates, ", "),
		})
	}
	return nil
}

// normalizeForComparison creates a normalized string representation of any value
// for unique comparison, ensuring that objects with same key-value pairs but
// different property orders are considered equal.
func normalizeForComparison(value any) (string, error) {
	return normalizeValue(value)
}

// normalizeValue recursively normalizes a value for comparison.
// This optimized version uses type assertions for common JSON types to avoid
// reflection overhead, which provides 5-10x performance improvement for typical usage.
func normalizeValue(value any) (string, error) {
	// Fast path: Use type assertions for common JSON types
	switch v := value.(type) {
	case nil:
		return "null", nil

	case string:
		return fmt.Sprintf(`"%s"`, v), nil

	case bool:
		return fmt.Sprintf("%t", v), nil

	case float64:
		return fmt.Sprintf("%g", v), nil

	case int:
		return fmt.Sprintf("%d", v), nil

	case int64:
		return fmt.Sprintf("%d", v), nil

	case int32:
		return fmt.Sprintf("%d", v), nil

	case uint:
		return fmt.Sprintf("%d", v), nil

	case uint64:
		return fmt.Sprintf("%d", v), nil

	case uint32:
		return fmt.Sprintf("%d", v), nil

	case map[string]any:
		// For maps, sort keys to ensure consistent ordering
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		var sb strings.Builder
		sb.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(fmt.Sprintf(`"%s":`, k))
			normalized, err := normalizeValue(v[k])
			if err != nil {
				return "", err
			}
			sb.WriteString(normalized)
		}
		sb.WriteByte('}')
		return sb.String(), nil

	case []any:
		var sb strings.Builder
		sb.WriteByte('[')
		for i, elem := range v {
			if i > 0 {
				sb.WriteByte(',')
			}
			normalized, err := normalizeValue(elem)
			if err != nil {
				return "", err
			}
			sb.WriteString(normalized)
		}
		sb.WriteByte(']')
		return sb.String(), nil
	}

	// Slow path: Fall back to reflection for uncommon types
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		// For maps, sort keys to ensure consistent ordering
		keys := rv.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int {
			return cmp.Compare(
				fmt.Sprintf("%v", a.Interface()),
				fmt.Sprintf("%v", b.Interface()),
			)
		})

		var pairs []string
		for _, key := range keys {
			keyStr, err := normalizeValue(key.Interface())
			if err != nil {
				return "", err
			}
			valueStr, err := normalizeValue(rv.MapIndex(key).Interface())
			if err != nil {
				return "", err
			}
			pairs = append(pairs, fmt.Sprintf("%s:%s", keyStr, valueStr))
		}
		return fmt.Sprintf("{%s}", strings.Join(pairs, ",")), nil

	case reflect.Slice, reflect.Array:
		var elements []string
		for i := 0; i < rv.Len(); i++ {
			elemStr, err := normalizeValue(rv.Index(i).Interface())
			if err != nil {
				return "", err
			}
			elements = append(elements, elemStr)
		}
		return fmt.Sprintf("[%s]", strings.Join(elements, ",")), nil

	case reflect.String:
		return fmt.Sprintf(`"%s"`, rv.String()), nil

	case reflect.Bool:
		return fmt.Sprintf("%t", rv.Bool()), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", rv.Int()), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", rv.Uint()), nil

	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", rv.Float()), nil

	case reflect.Ptr:
		if rv.IsNil() {
			return "null", nil
		}
		return normalizeValue(rv.Elem().Interface())

	case reflect.Interface:
		if rv.IsNil() {
			return "null", nil
		}
		return normalizeValue(rv.Elem().Interface())

	case reflect.Struct:
		// For structs, marshal to JSON as fallback
		bytes, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(bytes), nil

	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.UnsafePointer:
		// These types are not typically JSON serializable, use fallback
		bytes, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(bytes), nil

	default:
		// For other types, use JSON marshaling as fallback
		bytes, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}
}
