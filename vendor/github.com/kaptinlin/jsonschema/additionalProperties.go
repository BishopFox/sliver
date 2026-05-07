package jsonschema

import (
	"fmt"
	"strings"
)

// evaluateAdditionalProperties checks if properties not explicitly defined or matched by patternProperties conform to the schema specified in additionalProperties.
// According to the JSON Schema Draft 2020-12:
//   - The value of "additionalProperties" must be a valid JSON Schema.
//   - This keyword validates child values of instance names that do not appear in the annotation results of either "properties" or "patternProperties".
//   - Validation succeeds for these properties if the child instance validates against the "additionalProperties" schema.
//   - Omitting "additionalProperties" has the same assertion behavior as an empty schema, which allows any type of value.
//
// This function ensures that all properties not explicitly mentioned or matched are validated according to a default schema or constraints.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-additionalproperties
func evaluateAdditionalProperties(
	schema *Schema, object map[string]any, evaluatedProps map[string]bool,
	_ map[int]bool, dynamicScope *DynamicScope,
) ([]*EvaluationResult, *EvaluationError) {
	var results []*EvaluationResult
	var invalidProperties []string

	properties := make(map[string]bool)
	if schema.Properties != nil {
		for propName := range *schema.Properties {
			properties[propName] = true
		}
	}
	if schema.PatternProperties != nil {
		for _, regex := range schema.compiledPatterns {
			for propName := range object {
				if regex.MatchString(propName) {
					properties[propName] = true
				}
			}
		}
	}

	// Evaluate additional properties
	if schema.AdditionalProperties != nil {
		for propName, propValue := range object {
			if !properties[propName] {
				result, _, _ := schema.AdditionalProperties.evaluate(propValue, dynamicScope)
				if result != nil {
					//nolint:errcheck
					result.SetEvaluationPath(fmt.Sprintf("/additionalProperties/%s", propName)).
						SetSchemaLocation(schema.GetSchemaLocation(fmt.Sprintf("/additionalProperties/%s", propName))).
						SetInstanceLocation(fmt.Sprintf("/%s", propName))

					results = append(results, result)
					if !result.IsValid() {
						invalidProperties = append(invalidProperties, propName)
					}
				}

				// Mark property as evaluated
				evaluatedProps[propName] = true
			}
		}
	}

	if len(invalidProperties) == 1 {
		return results, NewEvaluationError(
			"additionalProperties", "additional_property_mismatch",
			"Additional property {property} does not match the schema",
			map[string]any{"property": fmt.Sprintf("'%s'", invalidProperties[0])},
		)
	}
	if len(invalidProperties) > 1 {
		quotedProperties := make([]string, len(invalidProperties))
		for i, prop := range invalidProperties {
			quotedProperties[i] = fmt.Sprintf("'%s'", prop)
		}
		return results, NewEvaluationError(
			"additionalProperties", "additional_properties_mismatch",
			"Additional properties {properties} do not match the schema",
			map[string]any{"properties": strings.Join(quotedProperties, ", ")},
		)
	}

	return results, nil
}
