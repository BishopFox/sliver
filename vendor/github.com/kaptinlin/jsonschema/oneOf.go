package jsonschema

import (
	"fmt"
	"strconv"
	"strings"
)

// evaluateOneOf checks if the data conforms to exactly one of the schemas specified in the oneOf attribute.
// According to the JSON Schema Draft 2020-12:
//   - The "oneOf" keyword's value must be a non-empty array, where each item is either a valid JSON Schema or a boolean.
//   - An instance validates successfully against this keyword if it validates successfully against exactly one schema or is true for exactly one boolean in this array.
//
// This function ensures that the data instance meets exactly one of the specified constraints defined by the schemas or booleans in the oneOf array.
// If the instance conforms to more than one or none of the schemas, it returns a EvaluationError detailing the specific failures or the lack of a valid schema.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-oneof
func evaluateOneOf(
	schema *Schema, instance any, evaluatedProps map[string]bool,
	evaluatedItems map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	if len(schema.OneOf) == 0 {
		return nil, nil // No oneOf constraints to validate against.
	}

	var validIndexes []string
	var results []*EvaluationResult
	var tempEvaluatedProps map[string]bool
	var tempEvaluatedItems map[int]bool

	for i, subSchema := range schema.OneOf {
		if subSchema != nil {
			result, schemaEvaluatedProps, schemaEvaluatedItems := subSchema.evaluate(instance, dynamicScope)
			if result != nil {
				results = append(results, result.SetEvaluationPath(fmt.Sprintf("/oneOf/%d", i)).
					SetSchemaLocation(schema.GetSchemaLocation(fmt.Sprintf("/oneOf/%d", i))),
				)

				if result.IsValid() {
					validIndexes = append(validIndexes, strconv.Itoa(i))
					tempEvaluatedProps = schemaEvaluatedProps
					tempEvaluatedItems = schemaEvaluatedItems
				}
			}
		}
	}

	if len(validIndexes) == 1 {
		// Merge maps only if exactly one schema or boolean condition is successfully validated
		mergeStringMaps(evaluatedProps, tempEvaluatedProps)
		mergeIntMaps(evaluatedItems, tempEvaluatedItems)
		return results, nil
	}

	if len(validIndexes) > 1 {
		return results, NewEvaluationError("oneOf", "one_of_multiple_matches", "Value should match exactly one schema but matches multiple at indexes {matches}", map[string]any{
			"matches": strings.Join(validIndexes, ", "),
		})
	}
	// If no conditions are met, return error
	return results, NewEvaluationError("oneOf", "one_of_item_mismatch", "Value does not match the oneOf schema")
}
