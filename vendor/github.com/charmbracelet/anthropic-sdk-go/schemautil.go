package anthropic

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
)

// BetaJSONSchemaOutputFormat creates a BetaJSONOutputFormatParam from a JSON schema map.
// It transforms the schema to ensure compatibility with Anthropic's JSON schema requirements.
//
// Example:
//
//	schema := map[string]any{
//	    "type": "object",
//	    "properties": map[string]any{
//	        "name": map[string]any{"type": "string"},
//	        "age": map[string]any{"type": "integer", "minimum": 0},
//	    },
//	    "required": []string{"name"},
//	}
//	outputFormat := BetaJSONSchemaOutputFormat(schema)
//
//	msg, _ := client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
//	    Model: anthropic.Model("claude-sonnet-4-5"),
//	    Messages: anthropic.F([]anthropic.BetaMessageParam{...}),
//	    MaxTokens: 1024,
//	    OutputFormat: outputFormat,
//	})
func BetaJSONSchemaOutputFormat(jsonSchema map[string]any) BetaJSONOutputFormatParam {
	return BetaJSONOutputFormatParam{Schema: transformSchema(jsonSchema)}
}

// BetaToolInputSchema creates a BetaToolInputSchemaParam from a JSON schema map.
// It transforms the schema to ensure compatibility with Anthropic's tool calling requirements.
func BetaToolInputSchema(jsonSchema map[string]any) BetaToolInputSchemaParam {
	return BetaToolInputSchemaParam{ExtraFields: transformSchema(jsonSchema)}
}

var supportedStringFormats = []string{
	"date-time",
	"time",
	"date",
	"duration",
	"email",
	"hostname",
	"uri",
	"ipv4",
	"ipv6",
	"uuid",
}
var supportedSchemaKeys = []string{
	// Top-level schema keys
	"$ref",
	"$defs",
	"type",
	"anyOf",
	"oneOf",
	"description",
	"title",

	// Object-specific keys
	"properties",
	"additionalProperties",
	"required",

	// Array-specific keys
	"items",
	"minItems",

	// String-specific keys
	"format",
}

// TransformSchema transforms a JSON schema to ensure it conforms to the Anthropic API's expectations.
// It returns nil if the transformed schema is empty.
//
// The transformation process:
// - Preserves $ref references
// - Transforms $defs recursively
// - Handles anyOf/oneOf by converting oneOf to anyOf
// - Ensures objects have additionalProperties: false
// - Filters string formats to only supported ones
// - Limits array minItems to 0 or 1
// - Appends unsupported properties to the description
//
// Example:
//
//	schema := map[string]any{
//	    "type": "integer",
//	    "minimum": 1,
//	    "maximum": 10,
//	    "description": "A number",
//	}
//	transformed := TransformSchema(schema)
//	// Result: {"type": "integer", "description": "A number\n\n{minimum: 1, maximum: 10}"}
func transformSchema(jsonSchema map[string]any) map[string]any {
	if jsonSchema == nil {
		return nil
	}

	strictSchema := make(map[string]any)

	// Create a copy to avoid modifying the original
	schemaCopy := make(map[string]any)
	maps.Copy(schemaCopy, jsonSchema)

	// $ref is not supported alongside other properties
	if ref, ok := schemaCopy["$ref"]; ok {
		strictSchema["$ref"] = ref
		return strictSchema
	}

	for _, key := range supportedSchemaKeys {
		value, exists := schemaCopy[key]
		if exists {
			delete(schemaCopy, key)
			strictSchema[key] = value
		}
	}

	if defs, ok := strictSchema["$defs"]; ok {
		if defsMap, ok := defs.(map[string]any); ok {
			strictDefs := make(map[string]any)
			strictSchema["$defs"] = strictDefs

			for name, schema := range defsMap {
				if schemaMap, ok := schema.(map[string]any); ok {
					strictDefs[name] = transformSchema(schemaMap)
				}
			}
		}
	}

	typeValue, _ := strictSchema["type"]
	anyOf, _ := strictSchema["anyOf"]
	oneOf, _ := strictSchema["oneOf"]

	if anyOfSlice, ok := anyOf.([]any); ok {
		transformedVariants := make([]any, 0, len(anyOfSlice))
		for _, variant := range anyOfSlice {
			variantMap, ok := variant.(map[string]any)
			if !ok {
				continue
			}
			if transformed := transformSchema(variantMap); transformed != nil {
				transformedVariants = append(transformedVariants, transformed)
			}
		}
		strictSchema["anyOf"] = transformedVariants
	} else if oneOfSlice, ok := oneOf.([]any); ok {
		transformedVariants := make([]any, 0, len(oneOfSlice))
		for _, variant := range oneOfSlice {
			if variantMap, ok := variant.(map[string]any); ok {
				if transformed := transformSchema(variantMap); transformed != nil {
					transformedVariants = append(transformedVariants, transformed)
				}
			}
		}
		delete(strictSchema, "oneOf")
		strictSchema["anyOf"] = transformedVariants
	} else {
		if typeValue == nil {
			// schema is completely invalid, we have to bail
			return nil
		}

		strictSchema["type"] = typeValue
	}

	typeStr, _ := typeValue.(string)
	switch typeStr {
	case "object":
		if properties, ok := strictSchema["properties"]; ok {
			if propsMap, ok := properties.(map[string]any); ok {
				transformedProps := make(map[string]any)
				for key, propSchema := range propsMap {
					if propSchemaMap, ok := propSchema.(map[string]any); ok {
						transformedProps[key] = transformSchema(propSchemaMap)
					}
				}
				strictSchema["properties"] = transformedProps
			}
		} else {
			strictSchema["properties"] = make(map[string]any)
		}

		strictSchema["additionalProperties"] = false

	case "string":
		if format, ok := strictSchema["format"]; ok {
			if formatStr, ok := format.(string); ok {
				if !slices.Contains(supportedStringFormats, formatStr) {
					schemaCopy["format"] = format
					delete(strictSchema, "format")
				}
			}
		}

	case "array":
		if items, ok := strictSchema["items"]; ok {
			if itemsMap, ok := items.(map[string]any); ok {
				strictSchema["items"] = transformSchema(itemsMap)
			}
		}

		if minItems, ok := strictSchema["minItems"]; ok {
			if minItems != 0 && minItems != 1 {
				schemaCopy["minItems"] = minItems
				delete(strictSchema, "minItems")
			}
		}

	case "boolean", "integer", "number", "null":
		// These types are supported as-is
	}

	if len(schemaCopy) > 0 {
		description := strictSchema["description"]
		descStr, _ := description.(string)

		// Sort keys for deterministic output
		keys := make([]string, 0, len(schemaCopy))
		for key := range schemaCopy {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		extraProps := make([]string, 0, len(keys))
		for _, key := range keys {
			extraProps = append(extraProps, fmt.Sprintf("%s: %v", key, schemaCopy[key]))
		}

		if descStr != "" {
			strictSchema["description"] = descStr + "\n\n{" + strings.Join(extraProps, ", ") + "}"
		} else {
			strictSchema["description"] = "{" + strings.Join(extraProps, ", ") + "}"
		}
	}

	return strictSchema
}
