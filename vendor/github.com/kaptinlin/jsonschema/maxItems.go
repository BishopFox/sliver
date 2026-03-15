package jsonschema

import "fmt"

// evaluateMaxItems checks if the array data contains no more items than the maximum specified in the "maxItems" schema attribute.
// According to the JSON Schema Draft 2020-12:
//   - The value of "maxItems" must be a non-negative integer.
//   - An array instance is valid against "maxItems" if its size is less than, or equal to, the value of this keyword.
//   - If the data is not an array, the "maxItems" constraint does not apply and should be ignored.
//
// This method ensures that the array data instance conforms to the maximum items constraints defined in the schema.
// If the instance violates this constraint, it returns a EvaluationError detailing the allowed maximum and the actual size.
// If the data is not an array, it returns nil, indicating the data is valid for this constraint.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-maxitems
func evaluateMaxItems(schema *Schema, array []any) *EvaluationError {
	if schema.MaxItems != nil {
		if float64(len(array)) > *schema.MaxItems {
			// If the array size exceeds the maximum allowed, construct and return an error.
			return NewEvaluationError("maxItems", "items_too_long", "Value should have at most {max_items} items", map[string]any{
				"max_items": fmt.Sprintf("%.0f", *schema.MaxItems),
				"count":     len(array),
			})
		}
	}
	return nil
}
