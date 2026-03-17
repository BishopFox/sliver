package jsonschema

// evaluateNot checks if the data fails to conform to the schema or boolean specified in the not attribute.
// According to the JSON Schema Draft 2020-12:
//   - The "not" keyword's value must be either a boolean or a valid JSON Schema.
//   - If "not" is a schema, an instance is valid against this keyword if it fails to validate successfully against the schema.
//   - If "not" is a boolean, the boolean value dictates the negation directly (true forbids validation, false allows any data).
//
// This function ensures that the data instance does not meet the constraints defined by the schema or respects the boolean in the not attribute.
// If the instance fails to conform to the schema or the boolean logic dictates failure, it returns a EvaluationError.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-not
func evaluateNot(schema *Schema, instance any, _ map[string]bool, _ map[int]bool, dynamicScope *DynamicScope) (*EvaluationResult, *EvaluationError) {
	if schema.Not == nil {
		return nil, nil // No 'not' constraints to validate against
	}

	result, _, _ := schema.Not.evaluate(instance, dynamicScope)

	if result != nil {
		//nolint:errcheck
		result.SetEvaluationPath("/not").
			SetSchemaLocation(schema.GetSchemaLocation("/not"))

		if result.IsValid() {
			return result, NewEvaluationError("not", "not_schema_mismatch", "Value should not match the not schema")
		}
	}

	return result, nil
}
