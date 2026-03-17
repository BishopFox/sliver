package jsonschema

import (
	"reflect"
)

// evaluateConst checks if the data matches exactly the value specified in the schema's 'const' keyword.
// According to the JSON Schema Draft 2020-12:
//   - The value of the "const" keyword may be of any type, including null.
//   - An instance validates successfully against this keyword if its value is equal to the value of the keyword.
//
// This function performs an equality check between the data and the constant value specified.
// If they do not match, it returns a EvaluationError detailing the expected and actual values.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-const
func evaluateConst(schema *Schema, instance any) *EvaluationError {
	if schema.Const == nil || !schema.Const.IsSet {
		return nil
	}

	// Special handling for null value comparison
	if schema.Const.Value == nil {
		if instance != nil {
			return NewEvaluationError("const", "const_mismatch_null", "Value should be null")
		}
		return nil
	}

	// Handle numeric type comparisons
	switch constVal := schema.Const.Value.(type) {
	case float64:
		switch instVal := instance.(type) {
		case float64:
			if constVal == instVal {
				return nil
			}
		case int:
			if constVal == float64(instVal) {
				return nil
			}
		}
		return NewEvaluationError("const", "const_mismatch", "Value does not match the constant value")
	case int:
		switch instVal := instance.(type) {
		case float64:
			if float64(constVal) == instVal {
				return nil
			}
		case int:
			if constVal == instVal {
				return nil
			}
		}
		return NewEvaluationError("const", "const_mismatch", "Value does not match the constant value")
	}

	// Use deep comparison for other types
	if !reflect.DeepEqual(instance, schema.Const.Value) {
		return NewEvaluationError("const", "const_mismatch", "Value does not match the constant value")
	}
	return nil
}
