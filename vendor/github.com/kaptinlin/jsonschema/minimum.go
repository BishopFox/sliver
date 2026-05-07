package jsonschema

// evaluateMinimum checks if the numeric data's value meets or exceeds the minimum value specified in the schema.
// According to the JSON Schema Draft 2020-12:
//   - The value of the "minimum" keyword must be a number, representing an inclusive lower limit for a numeric instance.
//   - This keyword validates only if the instance is greater than or exactly equal to "minimum".
//
// This method ensures that the numeric data instance conforms to the minimum constraints defined in the schema.
// If the instance is below the minimum value, it returns a EvaluationError detailing the expected minimum and actual value.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-minimum
func evaluateMinimum(schema *Schema, value *Rat) *EvaluationError {
	if schema.Minimum != nil {
		if value.Cmp(schema.Minimum.Rat) < 0 {
			// If the data value is below the minimum value, construct and return an error.
			return NewEvaluationError("minimum", "value_below_minimum", "{value} should be at least {minimum}", map[string]any{
				"value":   FormatRat(value),
				"minimum": FormatRat(schema.Minimum),
			})
		}
	}
	return nil
}
