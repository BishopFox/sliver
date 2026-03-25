package jsonschema

// evaluateMaxProperties checks if the number of properties in the object does not exceed the specified maximum.
// According to the JSON Schema Draft 2020-12:
//   - The "maxProperties" keyword must be a non-negative integer.
//   - An object instance is valid against "maxProperties" if its number of properties is less than, or equal to, the value of this keyword.
//
// This method ensures that the object instance conforms to the property count constraints defined in the schema.
// If the instance exceeds the maximum number of properties, it returns a EvaluationError detailing the expected maximum and the actual count.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-maxProperties
func evaluateMaxProperties(schema *Schema, object map[string]any) *EvaluationError {
	if schema.MaxProperties != nil {
		actualCount := float64(len(object))
		if actualCount > *schema.MaxProperties {
			return NewEvaluationError("maxProperties", "too_many_properties", "Value should have at most {max_properties} properties", map[string]any{
				"max_properties": *schema.MaxProperties,
			})
		}
	}

	return nil
}
