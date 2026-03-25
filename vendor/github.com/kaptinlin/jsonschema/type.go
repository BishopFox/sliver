package jsonschema

import (
	"strings"
)

// evaluateType checks if the data's type matches the type specified in the schema.
// According to the JSON Schema Draft 2020-12:
//   - The value of the "type" keyword must be either a string or an array of unique strings.
//   - Valid string values are the six primitive types ("null", "boolean", "object", "array", "number", "string")
//     and "integer", which matches any number with a zero fractional part.
//   - If "type" is a single string, the data matches if its type corresponds to that string.
//   - If "type" is an array, the data matches if its type corresponds to any string in that array.
//
// This method ensures that the data instance conforms to the type constraints defined in the schema.
// If the instance does not conform, it returns a EvaluationResult detailing the expected and actual types.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-type
func evaluateType(schema *Schema, instance any) *EvaluationError {
	// Evaluate only if type constraints are specified in the schema
	if len(schema.Type) == 0 {
		return nil // No type constraints, so no validation needed
	}

	instanceType := getDataType(instance) // Determine the type of the provided instance

	for _, schemaType := range schema.Type {
		if schemaType == "number" && instanceType == "integer" {
			// Special case: integers are valid numbers per JSON Schema specification
			return nil
		}
		if instanceType == schemaType {
			return nil // Correct type found, validation successful
		}
	}

	// If no valid type match is found, generate a EvaluationResult
	return NewEvaluationError("type", "type_mismatch", "Value is {received} but should be {expected}", map[string]any{
		"expected": strings.Join(schema.Type, ", "), // Expected types
		"received": instanceType,                    // Actual type of the input data
	})
}
