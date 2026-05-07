package jsonschema

import "fmt"

// evaluateContains checks if at least one element in an array meets the conditions specified by the 'contains' keyword.
// It follows the JSON Schema Draft 2020-12:
//   - "contains" must be associated with a valid JSON Schema.
//   - An array is valid if at least one of its elements matches the given schema, unless "minContains" is 0.
//   - When "minContains" is 0, the array is considered valid even if no elements match the schema.
//   - Produces annotations of indexes where the schema validates successfully.
//   - If every element validates, it produces a boolean "true".
//   - If the array is empty, annotations reflect the validation state.
//   - Influences "unevaluatedItems" and may be used to implement "minContains" and "maxContains".
//
// This function provides detailed feedback on element validation and gathers comprehensive annotations.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-contains
func evaluateContains(
	schema *Schema, data []any, _ map[string]bool,
	evaluatedItems map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	if schema.Contains == nil {
		// No 'contains' constraint is defined, skip further checks.
		return nil, nil
	}

	var results []*EvaluationResult

	var validCount int
	for i, item := range data {
		result, _, _ := schema.Contains.evaluate(item, dynamicScope)

		if result != nil {
			//nolint:errcheck
			result.SetEvaluationPath("/contains").
				SetSchemaLocation(schema.GetSchemaLocation("/contains")).
				SetInstanceLocation(fmt.Sprintf("/%d", i))

			if result.IsValid() {
				validCount++
				evaluatedItems[i] = true // Mark this item as evaluated
			}
		}
	}

	// Handle 'minContains' logic
	minContains := 1 // Default value if 'minContains' is not specified
	if schema.MinContains != nil {
		minContains = int(*schema.MinContains)
	}

	// Check minContains validation (skip if minContains is 0 and no valid items found - valid scenario)
	if (minContains != 0 || validCount != 0) && validCount < minContains {
		return results, NewEvaluationError(
			"minContains", "contains_too_few_items",
			"Value should contain at least {min_contains} matching items",
			map[string]any{"min_contains": minContains, "count": validCount},
		)
	}

	// Handle 'maxContains' logic
	if schema.MaxContains != nil && validCount > int(*schema.MaxContains) {
		return results, NewEvaluationError(
			"maxContains", "contains_too_many_items",
			"Value should contain no more than {max_contains} matching items",
			map[string]any{"max_contains": *schema.MaxContains, "count": validCount},
		)
	}

	return results, nil
}
