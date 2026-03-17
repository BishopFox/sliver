package jsonschema

import (
	"fmt"
	"strconv"
	"strings"
)

// evaluateUnevaluatedItems checks if the data's array items that have not been evaluated by 'items', 'prefixItems', or 'contains'
// conform to the subschema specified in the 'unevaluatedItems' attribute of the schema.
// According to the JSON Schema Draft 2020-12:
//   - The value of "unevaluatedItems" MUST be a valid JSON Schema.
//   - Validation depends on the annotation results of "prefixItems", "items", and "contains".
//   - If no relevant annotations are present, "unevaluatedItems" must apply to all locations in the array.
//   - If a boolean true value is present from any annotations, "unevaluatedItems" must be ignored.
//   - Otherwise, the subschema must be applied to any index greater than the largest evaluated index.
//
// This method ensures that any unevaluated array elements conform to the constraints defined in the unevaluatedItems attribute.
// If an unevaluated array element does not conform, it returns a EvaluationError detailing the issue.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-unevaluateditems
func evaluateUnevaluatedItems(
	schema *Schema, data any, _ map[string]bool,
	evaluatedItems map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	items, ok := data.([]any)
	if !ok {
		return nil, nil // If data is not an array, then skip the array-specific validations.
	}

	// If UnevaluatedItems is not set, all items are considered evaluated
	if schema.UnevaluatedItems == nil {
		return nil, nil
	}

	// If UnevaluatedItems is a boolean type
	if schema.UnevaluatedItems.Boolean != nil {
		if *schema.UnevaluatedItems.Boolean {
			// If true, all unevaluated items are valid
			for i := range items {
				evaluatedItems[i] = true
			}
			return nil, nil
		}
		// If false, any unevaluated item is invalid
		var unevaluatedIndexes []string
		for i := range items {
			if _, evaluated := evaluatedItems[i]; !evaluated {
				unevaluatedIndexes = append(unevaluatedIndexes, strconv.Itoa(i))
			}
		}
		if len(unevaluatedIndexes) > 0 {
			return nil, NewEvaluationError("unevaluatedItems", "unevaluated_items_not_allowed", "Unevaluated items are not allowed at indexes: {indexes}", map[string]any{
				"indexes": strings.Join(unevaluatedIndexes, ", "),
			})
		}
		return nil, nil
	}

	var results []*EvaluationResult
	var invalidIndexes []string

	// Evaluate unevaluated items
	for i, item := range items {
		if _, evaluated := evaluatedItems[i]; !evaluated {
			result, _, evaluatedMap := schema.UnevaluatedItems.evaluate(item, dynamicScope)
			if result != nil {
				//nolint:errcheck
				result.SetEvaluationPath(fmt.Sprintf("/unevaluatedItems/%d", i)).
					SetSchemaLocation(schema.GetSchemaLocation(fmt.Sprintf("/unevaluatedItems/%d", i))).
					SetInstanceLocation(fmt.Sprintf("/%d", i))

				results = append(results, result)
				if result.IsValid() {
					evaluatedItems[i] = true
				} else {
					invalidIndexes = append(invalidIndexes, strconv.Itoa(i))
				}
			}
			// Merge evaluation states
			for k, v := range evaluatedMap {
				evaluatedItems[k] = v
			}
		}
	}

	if len(invalidIndexes) == 1 {
		return results, NewEvaluationError("unevaluatedItems", "unevaluated_item_mismatch", "Item at index {index} does not match the unevaluatedItems schema", map[string]any{
			"index": invalidIndexes[0],
		})
	}
	if len(invalidIndexes) > 1 {
		return results, NewEvaluationError("unevaluatedItems", "unevaluated_items_mismatch", "Items at indexes {indexes} do not match the unevaluatedItems schema", map[string]any{
			"indexes": strings.Join(invalidIndexes, ", "),
		})
	}

	return results, nil
}
