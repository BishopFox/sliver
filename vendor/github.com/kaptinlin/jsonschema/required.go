package jsonschema

import (
	"fmt"
	"strings"
)

// evaluateRequired checks if all the required properties specified in the schema are present in the data object.
// According to the JSON Schema Draft 2020-12:
//   - The value of the "required" keyword must be an array of strings, where each string is a unique property name.
//   - An object instance is valid against this keyword if every item in the array is the name of a property in the instance.
//   - Omitting this keyword has the same behavior as an empty array, meaning no properties are required.
//
// This method ensures that all properties listed as required are present in the data instance.
// If a required property is missing, it returns a EvaluationError detailing the missing properties.
//
// Reference: https://json-schema.org/draft/2020-12/json-schema-validation#name-required
func evaluateRequired(schema *Schema, object map[string]any) *EvaluationError {
	if schema.Required == nil {
		// No required properties defined, nothing to do.
		return nil
	}

	// Proceed with checking for required properties only if it is indeed an object.
	var missingProps []string
	for _, propName := range schema.Required {
		if _, exists := object[propName]; !exists {
			missingProps = append(missingProps, propName)
		}
	}

	if len(missingProps) > 0 {
		if len(missingProps) == 1 {
			return NewEvaluationError("required", "missing_required_property", "Required property {property} is missing", map[string]any{
				"property": fmt.Sprintf("'%s'", missingProps[0]),
			})
		}
		quotedProperties := make([]string, len(missingProps))
		for i, prop := range missingProps {
			quotedProperties[i] = fmt.Sprintf("'%s'", prop)
		}
		return NewEvaluationError("required", "missing_required_properties", "Required properties {properties} are missing", map[string]any{
			"properties": strings.Join(quotedProperties, ", "),
		})
	}

	return nil
}
