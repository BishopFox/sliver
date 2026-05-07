package jsonschema

// evaluateExclusiveMaximum checks if a numeric instance is strictly less than the value specified by exclusiveMaximum.
// According to the JSON Schema Draft 2020-12:
//   - The value of the "exclusiveMaximum" keyword must be a number.
//   - The instance is valid if it is strictly less than (not equal to) the value specified by "exclusiveMaximum".
//
// This method ensures that the numeric data instance does not exceed the exclusive maximum limit defined in the schema.
// If the instance exceeds this limit, it returns a EvaluationError detailing the expected maximum value and the actual value.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-exclusivemaximum
func evaluateExclusiveMaximum(schema *Schema, value *Rat) *EvaluationError {
	if schema.ExclusiveMaximum != nil {
		if value.Cmp(schema.ExclusiveMaximum.Rat) >= 0 {
			// Data exceeds the exclusive maximum value.
			return NewEvaluationError("exclusiveMaximum", "exclusive_maximum_mismatch", "{value} should be less than {exclusive_maximum}", map[string]any{
				"exclusive_maximum": FormatRat(schema.ExclusiveMaximum),
				"value":             FormatRat(value),
			})
		}
	}
	return nil
}
