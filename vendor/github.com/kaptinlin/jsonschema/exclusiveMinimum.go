package jsonschema

// evaluateExclusiveMinimum checks if a numeric instance is strictly greater than the value specified by exclusiveMinimum.
// According to the JSON Schema Draft 2020-12:
//   - The value of the "exclusiveMinimum" keyword must be a number.
//   - The instance is valid only if it is strictly greater than (not equal to) the value specified by "exclusiveMinimum".
//
// This method ensures that the numeric data instance meets the exclusive minimum limit defined in the schema.
// If the instance does not meet this limit, it returns a EvaluationError detailing the expected minimum value and the actual value.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-exclusiveminimum
func evaluateExclusiveMinimum(schema *Schema, value *Rat) *EvaluationError {
	if schema.ExclusiveMinimum != nil {
		if value.Cmp(schema.ExclusiveMinimum.Rat) <= 0 {
			// Data does not meet the exclusive minimum value.
			return NewEvaluationError("exclusiveMinimum", "exclusive_minimum_mismatch", "{value} should be greater than {exclusive_minimum}", map[string]any{
				"exclusive_minimum": FormatRat(schema.ExclusiveMinimum),
				"value":             FormatRat(value),
			})
		}
	}
	return nil
}
