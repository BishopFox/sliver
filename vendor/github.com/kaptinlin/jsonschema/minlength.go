package jsonschema

import (
	"unicode/utf8"
)

// evaluateMinLength checks if the length of a string instance meets or exceeds the minLength specified in the schema.
// According to the JSON Schema Draft 2020-12:
//   - The "minLength" keyword must be a non-negative integer.
//   - A string instance is valid against this keyword if its length is greater than or equal to the value of this keyword.
//   - The length of a string instance is defined as the number of its characters as defined by RFC 8259.
//   - Omitting this keyword has the same behavior as a value of 0.
//
// This method ensures that the string data instance does not fall short of the minimum length defined in the schema.
// If the instance is shorter than this length, it returns a EvaluationError detailing the minimum required length and the actual length.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-minlength
func evaluateMinLength(schema *Schema, value string) *EvaluationError {
	if schema.MinLength != nil {
		// Use utf8.RuneCountInString to correctly count the number of graphemes in the string
		length := utf8.RuneCountInString(value)
		if length < int(*schema.MinLength) {
			// String does not meet the minimum length.
			return NewEvaluationError("minLength", "string_too_short", "Value should be at least {min_length} characters", map[string]any{
				"min_length": *schema.MinLength,
				"length":     length,
			})
		}
	}
	return nil
}
