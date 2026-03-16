package jsonschema

import (
	"fmt"
	"strings"
)

// evaluatePropertyNames checks if every property name in the object conforms to the schema specified by the propertyNames attribute.
// According to the JSON Schema Draft 2020-12:
//   - The "propertyNames" keyword must be a valid JSON Schema.
//   - If the instance is an object, this keyword validates if every property name in the instance validates against the provided schema.
//   - The property name that the schema is testing will always be a string.
//   - Omitting this keyword has the same behavior as an empty schema.
//
// This method ensures that each property name in the object instance conforms to the constraints defined in the propertyNames schema.
// If a property name does not conform, it returns a EvaluationError detailing the issue with that specific property name.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-core#name-propertynames
func evaluatePropertyNames(schema *Schema, object map[string]any, _ map[string]bool, _ map[int]bool, dynamicScope *DynamicScope) ([]*EvaluationResult, *EvaluationError) {
	if schema.PropertyNames == nil {
		// No propertyNames schema defined, equivalent to an empty schema, which means all property names are valid.
		return nil, nil
	}

	var invalidProperties []string
	var results []*EvaluationResult

	if schema.PropertyNames != nil {
		for propName := range object {
			result, _, _ := schema.PropertyNames.evaluate(propName, dynamicScope)

			if result != nil {
				//nolint:errcheck
				result.SetEvaluationPath(fmt.Sprintf("/propertyNames/%s", propName)).
					SetSchemaLocation(schema.GetSchemaLocation(fmt.Sprintf("/propertyNames/%s", propName))).
					SetInstanceLocation(fmt.Sprintf("/%s", propName))
			}

			results = append(results, result)

			if !result.IsValid() {
				invalidProperties = append(invalidProperties, propName)
			}
		}
	}

	if len(invalidProperties) == 1 {
		return results, NewEvaluationError("propertyNames", "property_name_mismatch", "Property name {property} does not match the schema", map[string]any{
			"property": fmt.Sprintf("'%s'", invalidProperties[0]),
		})
	}
	if len(invalidProperties) > 1 {
		quotedProperties := make([]string, len(invalidProperties))
		for i, prop := range invalidProperties {
			quotedProperties[i] = fmt.Sprintf("'%s'", prop)
		}
		return results, NewEvaluationError("propertyNames", "property_names_mismatch", "Property names {properties} do not match the schema", map[string]any{
			"properties": strings.Join(quotedProperties, ", "),
		})
	}

	return results, nil
}
