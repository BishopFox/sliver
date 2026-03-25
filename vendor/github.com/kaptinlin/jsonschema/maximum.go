package jsonschema

// evaluateMaximum checks if the numeric data's value does not exceed the maximum value specified in the schema.
// According to the JSON Schema Draft 2020-12:
//   - The value of the "maximum" keyword must be a number, representing an inclusive upper limit for a numeric instance.
//   - This keyword validates only if the instance is less than or exactly equal to "maximum".
//
// This method ensures that the numeric data instance conforms to the maximum constraints defined in the schema.
// If the instance exceeds the maximum value, it returns a EvaluationError detailing the expected maximum and actual value.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-maximum
func evaluateMaximum(schema *Schema, value *Rat) *EvaluationError {
	if schema.Maximum.Rat != nil {
		if value.Cmp(schema.Maximum.Rat) > 0 {
			// If the data value exceeds the maximum value, construct and return an error.
			return NewEvaluationError("maximum", "value_above_maximum", "{value} should be at most {maximum}", map[string]any{
				"value":   FormatRat(value),
				"maximum": FormatRat(schema.Maximum),
			})
		}
	}
	return nil
}
