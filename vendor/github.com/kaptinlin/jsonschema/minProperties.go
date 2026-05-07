package jsonschema

// evaluateMinProperties checks if the number of properties in the object meets or exceeds the specified minimum.
// According to the JSON Schema Draft 2020-12:
//   - The "minProperties" keyword must be a non-negative integer.
//   - An object instance is valid against "minProperties" if its number of properties is greater than or equal to the value of this keyword.
//   - Omitting this keyword has the same behavior as a value of 0.
//
// This method ensures that the object instance conforms to the property count constraints defined in the schema.
// If the instance has fewer properties than the minimum required, it returns a EvaluationError detailing the expected minimum and the actual count.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-minProperties
func evaluateMinProperties(schema *Schema, object map[string]any) *EvaluationError {
	minProperties := float64(0) // Default value if minProperties is omitted
	if schema.MinProperties != nil {
		minProperties = *schema.MinProperties
	}

	actualCount := float64(len(object))
	if actualCount < minProperties {
		return NewEvaluationError("minProperties", "too_few_properties", "Value should have at least {min_properties} properties", map[string]any{
			"min_properties": minProperties,
		})
	}

	return nil
}
