package jsonschema

import (
	"fmt"
	"strconv"
	"strings"
)

// evaluateItems checks if the data's array items conform to the subschema or boolean condition specified in the 'items' attribute of the schema.
// According to the JSON Schema Draft 2020-12:
//   - The value of "items" MUST be either a valid JSON Schema or a boolean.
//   - If "items" is a Schema, each element of the instance array must conform to this subschema.
//   - If "items" is boolean and is true, any array elements are valid.
//   - If "items" is boolean and is false, no array elements are valid unless the array is empty.
//
// This method ensures that array elements conform to the constraints defined in the items attribute.
// If any array element does not conform, it returns a EvaluationError detailing the issue.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-items
func evaluateItems(
	schema *Schema, array []any, _ map[string]bool,
	evaluatedItems map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	if schema.Items == nil {
		return nil, nil // // No 'items' constraints to validate against
	}

	var invalidIndexes []string
	var results []*EvaluationResult

	// Number of prefix items to skip before regular item validation
	startIndex := len(schema.PrefixItems)

	// Check if the general 'items' schema is available and proceed with validation if it's not explicitly false
	if schema.Items != nil {
		// Ensure that we only access indices within the range of existing array elements
		for i := startIndex; i < len(array); i++ {
			item := array[i]
			result, _, _ := schema.Items.evaluate(item, dynamicScope)
			if result != nil {
				//nolint:errcheck
				result.SetEvaluationPath(fmt.Sprintf("/items/%d", i)).
					SetSchemaLocation(schema.GetSchemaLocation(fmt.Sprintf("/items/%d", i))).
					SetInstanceLocation(fmt.Sprintf("/%d", i))

				if result.IsValid() {
					evaluatedItems[i] = true // Mark the item as evaluated if it passes schema validation.
				} else {
					results = append(results, result)
					invalidIndexes = append(invalidIndexes, strconv.Itoa(i))
				}
			}
		}
	}

	if len(invalidIndexes) == 1 {
		return results, NewEvaluationError("items", "item_mismatch", "Item at index {index} does not match the schema", map[string]any{
			"index": invalidIndexes[0],
		})
	}
	if len(invalidIndexes) > 1 {
		return results, NewEvaluationError("items", "items_mismatch", "Items at index {indexs} do not match the schema", map[string]any{
			"indexs": strings.Join(invalidIndexes, ", "),
		})
	}
	return results, nil
}
