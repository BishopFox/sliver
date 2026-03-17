package jsonschema

import (
	"fmt"
	"strings"
)

// evaluateUnevaluatedProperties checks if the unevaluated properties of the data object conform to the unevaluatedProperties schema specified in the schema.
// This is in accordance with the JSON Schema Draft 2020-12 specification which dictates that:
// - The value of "unevaluatedProperties" must be a valid JSON Schema.
// - This keyword's behavior is contingent on the annotations from "properties", "patternProperties", and "additionalProperties".
// - Validation applies only to properties that do not appear in the annotations resulting from the aforementioned keywords.
// - The validation succeeds if such properties conform to the "unevaluatedProperties" schema.
// - The annotations influence the evaluation order, meaning all related properties and applicators must be processed first.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-unevaluatedproperties
func evaluateUnevaluatedProperties(
	schema *Schema, data any, evaluatedProps map[string]bool,
	_ map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	if schema.UnevaluatedProperties == nil {
		return nil, nil // If "unevaluatedProperties" is not defined, all properties are considered evaluated.
	}

	var invalidProperties []string
	var results []*EvaluationResult

	object, ok := data.(map[string]any)
	if !ok {
		return nil, nil // If data is not an object, then skip the unevaluatedProperties validation.
	}

	// Loop through all properties of the object to find unevaluated properties.
	for propName, propValue := range object {
		if _, evaluated := evaluatedProps[propName]; !evaluated {
			// If property has not been evaluated, validate it against the "unevaluatedProperties" schema.
			result, _, _ := schema.UnevaluatedProperties.evaluate(propValue, dynamicScope)
			if result != nil {
				//nolint:errcheck
				result.SetEvaluationPath("/unevaluatedProperties").
					SetSchemaLocation(schema.GetSchemaLocation("/unevaluatedProperties")).
					SetInstanceLocation(fmt.Sprintf("/%s", propName))

				results = append(results, result)

				if !result.IsValid() {
					invalidProperties = append(invalidProperties, propName)
				}
			}
			evaluatedProps[propName] = true
		}
	}

	if len(invalidProperties) == 1 {
		return results, NewEvaluationError("properties", "unevaluated_property_mismatch", "Property {property} does not match the unevaluatedProperties schema", map[string]any{
			"property": fmt.Sprintf("'%s'", invalidProperties[0]),
		})
	}
	if len(invalidProperties) > 1 {
		quotedProperties := make([]string, len(invalidProperties))
		for i, prop := range invalidProperties {
			quotedProperties[i] = fmt.Sprintf("'%s'", prop)
		}
		return results, NewEvaluationError("properties", "unevaluated_properties_mismatch", "Properties {properties} do not match the unevaluatedProperties schema", map[string]any{
			"properties": strings.Join(quotedProperties, ", "),
		})
	}

	return results, nil
}
