package jsonschema

import (
	"fmt"
	"strconv"
	"strings"
)

// evaluatePrefixItems checks if each element in an array instance matches the schema specified at the same index in the 'prefixItems' array.
// According to the JSON Schema Draft 2020-12:
//   - The value of "prefixItems" MUST be a non-empty array of valid JSON Schemas.
//   - Validation succeeds if each element of the instance validates against the schema at the same position, if any.
//   - This keyword does not constrain the length of the array; it validates only the prefix of the array up to the length of 'prefixItems'.
//   - Produces an annotation value of the largest index validated if all items up to that index are valid. The value may be true if all items are valid.
//   - Omitting this keyword implies an empty array behavior, meaning no validation is enforced on the array items.
//
// If validation fails, it returns a EvaluationError detailing the index and discrepancy.
func evaluatePrefixItems(
	schema *Schema, array []any, _ map[string]bool,
	evaluatedItems map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	if len(schema.PrefixItems) == 0 {
		return nil, nil // If no prefixItems are defined, there is nothing to validate against.
	}

	var invalidIndexes []string
	var results []*EvaluationResult

	for i, itemSchema := range schema.PrefixItems {
		if i >= len(array) {
			break // Stop validation if there are more schemas than array items.
		}

		result, _, _ := itemSchema.evaluate(array[i], dynamicScope)
		if result != nil {
			results = append(results, result.SetEvaluationPath(fmt.Sprintf("/prefixItems/%d", i)).
				SetSchemaLocation(schema.GetSchemaLocation(fmt.Sprintf("/prefixItems/%d", i))).
				SetInstanceLocation(fmt.Sprintf("/%d", i)),
			)

			if result.IsValid() {
				evaluatedItems[i] = true // Mark the item as evaluated if it passes schema validation.
			} else {
				invalidIndexes = append(invalidIndexes, strconv.Itoa(i))
			}
		}
	}

	if len(invalidIndexes) == 1 {
		return results, NewEvaluationError("prefixItems", "prefix_item_mismatch", "Item at index {index} does not match the prefixItems schema", map[string]any{
			"index": invalidIndexes[0],
		})
	}
	if len(invalidIndexes) > 1 {
		return results, NewEvaluationError("prefixItems", "prefix_items_mismatch", "Items at index {indexs} do not match the prefixItems schemas", map[string]any{
			"indexs": strings.Join(invalidIndexes, ", "),
		})
	}

	return results, nil
}
