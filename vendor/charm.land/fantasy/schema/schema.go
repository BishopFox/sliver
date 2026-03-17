// Package schema provides JSON schema generation and validation utilities.
// It supports automatic schema generation from Go types and validation of parsed objects.
package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"charm.land/fantasy/jsonrepair"
	"github.com/kaptinlin/jsonschema"
)

// ObjectRepairFunc is a function that attempts to repair invalid JSON output.
// It receives the raw text and the error encountered during parsing or validation,
// and returns repaired text or an error if repair is not possible.
type ObjectRepairFunc func(ctx context.Context, text string, err error) (string, error)

// ParseError is returned when object generation fails
// due to parsing errors, validation errors, or model failures.
type ParseError struct {
	RawText         string
	ParseError      error
	ValidationError error
}

// Schema represents a JSON schema for tool input validation.
type Schema struct {
	Type        string             `json:"type,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Description string             `json:"description,omitempty"`
	Enum        []any              `json:"enum,omitempty"`
	Format      string             `json:"format,omitempty"`
	Minimum     *float64           `json:"minimum,omitempty"`
	Maximum     *float64           `json:"maximum,omitempty"`
	MinLength   *int               `json:"minLength,omitempty"`
	MaxLength   *int               `json:"maxLength,omitempty"`
}

// ParseState represents the state of JSON parsing.
type ParseState string

const (
	// ParseStateUndefined means input was undefined/empty.
	ParseStateUndefined ParseState = "undefined"

	// ParseStateSuccessful means JSON parsed without repair.
	ParseStateSuccessful ParseState = "successful"

	// ParseStateRepaired means JSON parsed after repair.
	ParseStateRepaired ParseState = "repaired"

	// ParseStateFailed means JSON could not be parsed even after repair.
	ParseStateFailed ParseState = "failed"
)

// ToParameters converts a Schema to the parameters map format expected by ToolInfo.
func ToParameters(s Schema) map[string]any {
	if s.Properties == nil {
		return make(map[string]any)
	}

	result := make(map[string]any)
	for name, propSchema := range s.Properties {
		result[name] = ToMap(*propSchema)
	}
	return result
}

// Generate generates a JSON schema from a reflect.Type.
// It recursively processes struct fields, arrays, maps, and primitive types.
func Generate(t reflect.Type) Schema {
	return generateSchemaRecursive(t, make(map[reflect.Type]bool))
}

func generateSchemaRecursive(t reflect.Type, visited map[reflect.Type]bool) Schema {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if visited[t] {
		return Schema{Type: "object"}
	}
	visited[t] = true
	defer delete(visited, t)

	switch t.Kind() {
	case reflect.String:
		return Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return Schema{Type: "number"}
	case reflect.Bool:
		return Schema{Type: "boolean"}
	case reflect.Slice, reflect.Array:
		itemSchema := generateSchemaRecursive(t.Elem(), visited)
		return Schema{
			Type:  "array",
			Items: &itemSchema,
		}
	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			valueSchema := generateSchemaRecursive(t.Elem(), visited)
			schema := Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"*": &valueSchema,
				},
			}
			return schema
		}
		return Schema{Type: "object"}
	case reflect.Struct:
		schema := Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
		}
		for i := range t.NumField() {
			field := t.Field(i)

			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			fieldName := field.Name
			required := true

			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					fieldName = parts[0]
				}

				if slices.Contains(parts[1:], "omitempty") {
					required = false
				}
			} else {
				fieldName = toSnakeCase(fieldName)
			}

			fieldSchema := generateSchemaRecursive(field.Type, visited)

			if desc := field.Tag.Get("description"); desc != "" {
				fieldSchema.Description = desc
			}

			if enumTag := field.Tag.Get("enum"); enumTag != "" {
				enumValues := strings.Split(enumTag, ",")
				fieldSchema.Enum = make([]any, len(enumValues))
				for i, v := range enumValues {
					fieldSchema.Enum[i] = strings.TrimSpace(v)
				}
			}

			schema.Properties[fieldName] = &fieldSchema

			if required {
				schema.Required = append(schema.Required, fieldName)
			}
		}

		return schema
	case reflect.Interface:
		return Schema{Type: "object"}
	default:
		return Schema{Type: "object"}
	}
}

// ToMap converts a Schema to a map representation suitable for JSON Schema.
func ToMap(schema Schema) map[string]any {
	result := make(map[string]any)

	if schema.Type != "" {
		result["type"] = schema.Type
	}

	if schema.Description != "" {
		result["description"] = schema.Description
	}

	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}

	if schema.Format != "" {
		result["format"] = schema.Format
	}

	if schema.Minimum != nil {
		result["minimum"] = *schema.Minimum
	}

	if schema.Maximum != nil {
		result["maximum"] = *schema.Maximum
	}

	if schema.MinLength != nil {
		result["minLength"] = *schema.MinLength
	}

	if schema.MaxLength != nil {
		result["maxLength"] = *schema.MaxLength
	}

	if schema.Properties != nil {
		props := make(map[string]any)
		for name, propSchema := range schema.Properties {
			props[name] = ToMap(*propSchema)
		}
		result["properties"] = props
	}

	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	if schema.Items != nil {
		itemsMap := ToMap(*schema.Items)
		// Ensure type is always set for items, even if it was blank for llama.cpp compatibility
		if _, hasType := itemsMap["type"]; !hasType && schema.Items.Type == "" {
			if len(schema.Items.Properties) > 0 {
				itemsMap["type"] = "object"
			}
		}
		result["items"] = itemsMap
	}

	return result
}

// ParsePartialJSON attempts to parse potentially incomplete JSON.
// It first tries standard JSON parsing, then attempts repair if that fails.
//
// Returns:
//   - result: The parsed JSON value (map, slice, or primitive)
//   - state: Indicates whether parsing succeeded, needed repair, or failed
//   - err: The error if parsing failed completely
//
// Example:
//
//	obj, state, err := ParsePartialJSON(`{"name": "John", "age": 25`)
//	// Result: map[string]any{"name": "John", "age": 25}, ParseStateRepaired, nil
func ParsePartialJSON(text string) (any, ParseState, error) {
	if text == "" {
		return nil, ParseStateUndefined, nil
	}

	var result any
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result, ParseStateSuccessful, nil
	}

	repaired, err := jsonrepair.RepairJSON(text)
	if err != nil {
		return nil, ParseStateFailed, fmt.Errorf("json repair failed: %w", err)
	}

	if err := json.Unmarshal([]byte(repaired), &result); err != nil {
		return nil, ParseStateFailed, fmt.Errorf("failed to parse repaired json: %w", err)
	}

	return result, ParseStateRepaired, nil
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.ValidationError != nil {
		return fmt.Sprintf("object validation failed: %v", e.ValidationError)
	}
	if e.ParseError != nil {
		return fmt.Sprintf("failed to parse object: %v", e.ParseError)
	}
	return "failed to generate object"
}

// ParseAndValidate combines JSON parsing and validation.
// Returns the parsed object if both parsing and validation succeed.
func ParseAndValidate(text string, schema Schema) (any, error) {
	obj, state, err := ParsePartialJSON(text)
	if state == ParseStateFailed {
		return nil, &ParseError{
			RawText:    text,
			ParseError: err,
		}
	}

	if err := validateAgainstSchema(obj, schema); err != nil {
		return nil, &ParseError{
			RawText:         text,
			ValidationError: err,
		}
	}

	return obj, nil
}

// ValidateAgainstSchema validates a parsed object against a Schema.
func ValidateAgainstSchema(obj any, schema Schema) error {
	return validateAgainstSchema(obj, schema)
}

func validateAgainstSchema(obj any, schema Schema) error {
	jsonSchemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	validator, err := compiler.Compile(jsonSchemaBytes)
	if err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	result := validator.Validate(obj)
	if !result.IsValid() {
		var errMsgs []string
		for field, validationErr := range result.Errors {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", field, validationErr.Message))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// ParseAndValidateWithRepair attempts parsing, validation, and custom repair.
func ParseAndValidateWithRepair(
	ctx context.Context,
	text string,
	schema Schema,
	repair ObjectRepairFunc,
) (any, error) {
	obj, state, parseErr := ParsePartialJSON(text)

	if state == ParseStateSuccessful || state == ParseStateRepaired {
		validationErr := validateAgainstSchema(obj, schema)
		if validationErr == nil {
			return obj, nil
		}

		if repair != nil {
			repairedText, repairErr := repair(ctx, text, validationErr)
			if repairErr == nil {
				obj2, state2, _ := ParsePartialJSON(repairedText)
				if state2 == ParseStateSuccessful || state2 == ParseStateRepaired {
					if err := validateAgainstSchema(obj2, schema); err == nil {
						return obj2, nil
					}
				}
			}
		}

		return nil, &ParseError{
			RawText:         text,
			ValidationError: validationErr,
		}
	}

	if repair != nil {
		repairedText, repairErr := repair(ctx, text, parseErr)
		if repairErr == nil {
			obj2, state2, parseErr2 := ParsePartialJSON(repairedText)
			if state2 == ParseStateSuccessful || state2 == ParseStateRepaired {
				if err := validateAgainstSchema(obj2, schema); err == nil {
					return obj2, nil
				}
			}
			return nil, &ParseError{
				RawText:    repairedText,
				ParseError: parseErr2,
			}
		}
	}

	return nil, &ParseError{
		RawText:    text,
		ParseError: parseErr,
	}
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Normalize recursively normalizes a raw JSON Schema map so it is
// compatible with providers that reject type-arrays (e.g. OpenAI). Type
// arrays are converted to anyOf and bare "array" types get "items":{}.
func Normalize(node map[string]any) {
	for _, child := range node {
		switch v := child.(type) {
		case map[string]any:
			Normalize(v)
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					Normalize(m)
				}
			}
		}
	}

	typeArr, ok := node["type"].([]any)
	if !ok {
		if node["type"] == "array" {
			if _, has := node["items"]; !has {
				node["items"] = map[string]any{}
			}
		}
		return
	}

	anyOf := make([]any, 0, len(typeArr))
	for _, t := range typeArr {
		variant := map[string]any{"type": t}
		if t == "array" {
			variant["items"] = map[string]any{}
		}
		anyOf = append(anyOf, variant)
	}
	delete(node, "type")
	node["anyOf"] = anyOf
}
