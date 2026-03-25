package jsonschema

// evaluateMinItems checks if the array data contains at least the minimum number of items specified in the "minItems" schema attribute.
// According to the JSON Schema Draft 2020-12:
//   - The value of "minItems" must be a non-negative integer.
//   - An array instance is valid against "minItems" if its size is greater than, or equal to, the value of this keyword.
//   - Omitting this keyword has the same behavior as a value of 0, which means no minimum size constraint unless explicitly specified.
//
// This method ensures that the array data instance conforms to the minimum items constraints defined in the schema.
// If the instance violates this constraint, it returns a EvaluationError detailing the required minimum and the actual size.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-minitems
func evaluateMinItems(schema *Schema, array []any) *EvaluationError {
	if schema.MinItems != nil {
		if float64(len(array)) < *schema.MinItems {
			// If the array size is less than the minimum required, construct and return an error.
			return NewEvaluationError("minItems", "items_too_short", "Value should have at least {min_items} items", map[string]any{
				"min_items": *schema.MinItems,
				"count":     len(array),
			})
		}
	}
	return nil
}
