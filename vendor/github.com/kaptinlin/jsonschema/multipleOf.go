package jsonschema

import "math/big"

// evaluateMultipleOf checks if the numeric data is a multiple of the value specified in the "multipleOf" schema attribute.
// According to the JSON Schema Draft 2020-12:
//   - The value of "multipleOf" must be a number, strictly greater than 0.
//   - A numeric instance is valid only if division by this keyword's value results in an integer.
//
// This method ensures that the numeric data instance conforms to the divisibility constraints defined in the schema.
// If the instance does not conform, it returns a EvaluationError detailing the expected divisor and the actual remainder.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-multipleof
func evaluateMultipleOf(schema *Schema, value *Rat) *EvaluationError {
	if schema.MultipleOf != nil {
		if schema.MultipleOf.Sign() == 0 || schema.MultipleOf.Sign() < 0 {
			// If the divisor is 0, return an error.
			return NewEvaluationError("multipleOf", "invalid_multiple_of", "Multiple of {divisor} should be greater than 0", map[string]any{
				"divisor": FormatRat(schema.MultipleOf),
			})
		}

		// Calculate the division result to check if it's an integer.
		resultRat := new(big.Rat).Quo(value.Rat, schema.MultipleOf.Rat)
		if !resultRat.IsInt() {
			// If the division result is not an integer, construct and return an error.
			return NewEvaluationError("multipleOf", "not_multiple_of", "{value} should be a multiple of {divisor}", map[string]any{
				"divisor": FormatRat(schema.MultipleOf),
				"value":   FormatRat(value),
			})
		}
	}
	return nil
}
