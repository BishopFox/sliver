package jsonschema

import (
	"fmt"
	"reflect"
	"strings"
)

// valuesEqual checks if two values are equal, handling type conversions for numeric types
func valuesEqual(a, b any) bool {
	// Try direct comparison first
	if reflect.DeepEqual(a, b) {
		return true
	}

	// Handle numeric type conversions
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	// If both are numeric, convert to float64 for comparison
	if isNumeric(va) && isNumeric(vb) {
		fa, ok1 := toFloat64(a)
		fb, ok2 := toFloat64(b)
		if ok1 && ok2 {
			return fa == fb
		}
	}

	return false
}

// isNumeric checks if a reflect.Value represents a numeric type
func isNumeric(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Invalid, reflect.Bool, reflect.Uintptr,
		reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan,
		reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr,
		reflect.Slice, reflect.String, reflect.Struct, reflect.UnsafePointer:
		return false
	default:
		return false
	}
}

// toFloat64 converts various numeric types to float64
func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

// evaluateEnum checks if the data's value matches one of the enumerated values specified in the schema.
// According to the JSON Schema Draft 2020-12:
//   - The value of the "enum" keyword must be an array.
//   - This array should have at least one element, and all elements should be unique.
//   - An instance validates successfully against this keyword if its value is equal to one of the elements in the array.
//   - Elements in the array might be of any type, including null.
//
// This method ensures that the data instance conforms to the enumerated values defined in the schema.
// If the instance does not match any of the enumerated values, it returns a EvaluationError detailing the allowed values.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-enum
func evaluateEnum(schema *Schema, instance any) *EvaluationError {
	if len(schema.Enum) == 0 {
		return nil // No enum values, so no validation needed
	}

	allowed := make([]string, 0, len(schema.Enum))

	for _, enumValue := range schema.Enum {
		if valuesEqual(instance, enumValue) {
			return nil // Match found.
		}

		allowed = append(allowed, fmt.Sprintf("%v", enumValue))
	}

	// No match found.
	return NewEvaluationError("enum", "value_not_in_enum", "Value {received} should be one of the allowed values: {expected}", map[string]any{
		"expected": strings.Join(allowed, ", "),
		"received": instance,
	})
}
