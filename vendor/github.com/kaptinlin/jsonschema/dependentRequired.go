package jsonschema

import (
	"github.com/go-json-experiment/json"
)

// evaluateDependentRequired checks that if a specified property is present, all its dependent properties are also present.
// According to the JSON Schema Draft 2020-12:
//   - The "dependentRequired" keyword specifies properties that are required if a specific other property is present.
//   - This keyword's value must be an object where each property is an array of strings, indicating properties required when the key property is present.
//   - Validation succeeds if, whenever a key property is present in the instance, all properties in its array are also present.
//
// This method ensures that property dependencies are respected in the data instance.
// If a dependency is not met, it returns a EvaluationError detailing the issue.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-dependentrequired
func evaluateDependentRequired(schema *Schema, object map[string]any) *EvaluationError {
	if schema.DependentRequired == nil {
		return nil // No dependent required properties defined, nothing to do.
	}

	dependentMissingProps := make(map[string][]string)

	for key, requiredProps := range schema.DependentRequired {
		if _, keyExists := object[key]; keyExists {
			var missingProps []string
			for _, reqProp := range requiredProps {
				if _, propExists := object[reqProp]; !propExists {
					missingProps = append(missingProps, reqProp)
				}
			}

			if len(missingProps) > 0 {
				dependentMissingProps[key] = missingProps
			}
		}
	}

	if len(dependentMissingProps) > 0 {
		missingPropsJSON, _ := json.Marshal(dependentMissingProps)
		return NewEvaluationError(
			"dependentRequired", "dependent_property_required",
			"Some required property dependencies are missing: {missing_properties}",
			map[string]any{"missing_properties": string(missingPropsJSON)},
		)
	}

	return nil
}
