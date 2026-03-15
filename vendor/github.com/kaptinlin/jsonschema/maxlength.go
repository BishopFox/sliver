package jsonschema

import (
	"fmt"
	"unicode/utf8"
)

// evaluateMaxLength checks if the length of a string instance does not exceed the maxLength specified in the schema.
// According to the JSON Schema Draft 2020-12:
//   - The "maxLength" keyword must be a non-negative integer.
//   - A string instance is valid against this keyword if its length is less than or equal to the value of this keyword.
//   - The length of a string instance is defined as the number of its characters as defined by RFC 8259.
//
// This method ensures that the string data instance does not exceed the maximum length defined in the schema.
// If the instance exceeds this length, it returns a EvaluationError detailing the maximum allowed length and the actual length.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-maxlength
func evaluateMaxLength(schema *Schema, value string) *EvaluationError {
	if schema.MaxLength != nil {
		// Use utf8.RuneCountInString to correctly count the number of graphemes in the string
		length := utf8.RuneCountInString(value)
		if length > int(*schema.MaxLength) {
			// String exceeds the maximum length.
			return NewEvaluationError("maxLength", "string_too_long", "Value should be at most {max_length} characters", map[string]any{
				"max_length": fmt.Sprintf("%.0f", *schema.MaxLength),
				"length":     length,
			})
		}
	}
	return nil
}
