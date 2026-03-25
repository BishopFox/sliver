package ai

import (
	"context"
	"testing"
)

type staticAgenticToolExecutor struct {
	definitions []AgenticToolDefinition
}

func (s staticAgenticToolExecutor) ToolDefinitions() []AgenticToolDefinition {
	return s.definitions
}

func (staticAgenticToolExecutor) CallTool(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}

func TestBuildAgenticToolParamsNormalizesStrictOpenAISchemas(t *testing.T) {
	params := buildAgenticToolParams(staticAgenticToolExecutor{
		definitions: []AgenticToolDefinition{
			{
				Name: "fs_ls",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session_id": map[string]any{"type": "string"},
						"beacon_id":  map[string]any{"type": "string"},
						"path":       map[string]any{"type": "string"},
					},
					"additionalProperties": false,
				},
			},
		},
	})
	if len(params) != 1 || params[0].OfFunction == nil {
		t.Fatalf("expected one function tool param, got %+v", params)
	}

	parameters := params[0].OfFunction.Parameters
	required := schemaRequiredSet(parameters["required"])
	for _, field := range []string{"session_id", "beacon_id", "path"} {
		if !required[field] {
			t.Fatalf("expected strict schema to require %q, got %#v", field, parameters["required"])
		}
	}

	properties, ok := parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected strict schema properties map, got %#v", parameters["properties"])
	}
	for _, field := range []string{"session_id", "beacon_id", "path"} {
		property, ok := properties[field].(map[string]any)
		if !ok {
			t.Fatalf("expected property %q to be a schema object, got %#v", field, properties[field])
		}
		typeValues, ok := property["type"].([]string)
		if !ok {
			t.Fatalf("expected property %q type to be []string, got %#v", field, property["type"])
		}
		if !stringSliceContains(typeValues, "string") || !stringSliceContains(typeValues, "null") {
			t.Fatalf("expected property %q to allow string and null, got %#v", field, typeValues)
		}
	}
}
