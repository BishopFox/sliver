package jsonschema

import (
	"fmt"
	"strconv"
	"strings"
)

// evaluateAllOf checks if the data conforms to all schemas specified in the allOf attribute.
// According to the JSON Schema Draft 2020-12:
//   - The "allOf" keyword's value must be a non-empty array, where each item is a valid JSON Schema or a boolean.
//   - An instance validates successfully against this keyword if it validates successfully against all schemas or booleans in this array.
//
// This function ensures that the data instance meets all the specified constraints collectively defined by the schemas or booleans in the allOf array.
// If the instance fails to conform to any schema or boolean in the array, it returns a EvaluationError detailing the specific failure.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-allof
func evaluateAllOf(
	schema *Schema, instance any, evaluatedProps map[string]bool,
	evaluatedItems map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	if len(schema.AllOf) == 0 {
		return nil, nil // No allOf constraints to validate against.
	}

	var invalidIndexes []string
	var results []*EvaluationResult

	for i, subSchema := range schema.AllOf {
		if subSchema != nil {
			skipEval := false
			if subSchema.Boolean != nil && *subSchema.Boolean {
				// If the schema is `true`, skip updating evaluated properties and items.
				skipEval = true
			}

			result, schemaEvaluatedProps, schemaEvaluatedItems := subSchema.evaluate(instance, dynamicScope)
			if !skipEval {
				mergeStringMaps(evaluatedProps, schemaEvaluatedProps)
				mergeIntMaps(evaluatedItems, schemaEvaluatedItems)
			}

			if result != nil {
				results = append(results, result.SetEvaluationPath(fmt.Sprintf("/allOf/%d", i)).
					SetSchemaLocation(schema.GetSchemaLocation(fmt.Sprintf("/allOf/%d", i))),
				)

				if !result.IsValid() {
					invalidIndexes = append(invalidIndexes, strconv.Itoa(i))
				}
			}
		}
	}

	if len(invalidIndexes) == 0 {
		return results, nil
	}

	return results, NewEvaluationError("allOf", "all_of_item_mismatch", "Value does not match the allOf schema at index {indexs}", map[string]any{
		"indexs": strings.Join(invalidIndexes, ", "),
	})
}
