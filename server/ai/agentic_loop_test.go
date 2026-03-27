package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
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

func TestCompleteConversationAgenticRequestsReasoningSummaryForOpenAI(t *testing.T) {
	type capturedRequest struct {
		Body string
	}

	requests := make(chan capturedRequest, 1)
	restoreClient := SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			defer r.Body.Close()

			payload, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}

			requests <- capturedRequest{Body: string(payload)}
			return jsonResponse(http.StatusOK, `{
				"id": "resp_123",
				"model": "gpt-5.4",
				"status": "completed",
				"output": [
					{
						"type": "message",
						"id": "msg_123",
						"role": "assistant",
						"content": [
							{"type": "output_text", "text": "Done."}
						]
					}
				]
			}`), nil
		}),
	})
	defer restoreClient()

	completion, err := CompleteConversationAgentic(context.Background(), &RuntimeConfig{
		Provider:        ProviderOpenAI,
		Model:           "gpt-5.4",
		UseResponsesAPI: true,
		ThinkingLevel:   "high",
		APIKey:          "openai-key",
	}, &clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Say hi."},
		},
	}, nil, nil)
	if err != nil {
		t.Fatalf("complete agentic conversation: %v", err)
	}

	request := <-requests
	for _, fragment := range []string{`"effort":"high"`, `"summary":"concise"`} {
		if !strings.Contains(request.Body, fragment) {
			t.Fatalf("expected agentic request body to contain %q, got %s", fragment, request.Body)
		}
	}
	if completion.Content != "Done." {
		t.Fatalf("unexpected completion content: %q", completion.Content)
	}
}
