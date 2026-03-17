package jsonschema

import (
	"fmt"
)

// evaluateAnyOf checks if the data conforms to at least one of the schemas specified in the anyOf attribute.
// According to the JSON Schema Draft 2020-12:
//   - The "anyOf" keyword's value must be a non-empty array, where each item is either a valid JSON Schema or a boolean.
//   - An instance validates successfully against this keyword if it validates successfully against at least one schema or is true for any boolean in this array.
//
// This function ensures that the data instance meets at least one of the specified constraints defined by the schemas or booleans in the anyOf array.
// If the instance fails to conform to all conditions in the array, it returns a EvaluationError detailing the specific failures.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-anyof
func evaluateAnyOf(
	schema *Schema, data any, evaluatedProps map[string]bool,
	evaluatedItems map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	if len(schema.AnyOf) == 0 {
		return nil, nil // No anyOf constraints to validate against.
	}

	var valid bool
	var results []*EvaluationResult

	for i, subSchema := range schema.AnyOf {
		if subSchema != nil {
			skipEval := false
			if subSchema.Boolean != nil && *subSchema.Boolean {
				// If the schema is `true`, skip updating evaluated properties and items.
				skipEval = true
			}
			result, schemaEvaluatedProps, schemaEvaluatedItems := subSchema.evaluate(data, dynamicScope)

			if result != nil {
				results = append(results, result.SetEvaluationPath(fmt.Sprintf("/anyOf/%d", i)).
					SetSchemaLocation(schema.GetSchemaLocation(fmt.Sprintf("/anyOf/%d", i))),
				)

				if result.IsValid() {
					valid = true
					// Merge maps only if the evaluation is successful
					if !skipEval {
						mergeStringMaps(evaluatedProps, schemaEvaluatedProps)
						mergeIntMaps(evaluatedItems, schemaEvaluatedItems)
					}
				}
			}
		}
	}

	if valid {
		return results, nil // Return nil only if at least one schema succeeds
	}
	return results, NewEvaluationError("anyOf", "any_of_item_mismatch", "Value does not match anyOf schema")
}
