package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
)

func TestCompleteConversationOpenAIUsesConfiguredCredentialsAndSettings(t *testing.T) {
	type capturedRequest struct {
		Path          string
		Authorization string
		Body          openAIRequest
	}

	requests := make(chan capturedRequest, 1)
	restoreClient := SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			defer r.Body.Close()

			var body openAIRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}

			requests <- capturedRequest{
				Path:          r.URL.Path,
				Authorization: r.Header.Get("Authorization"),
				Body:          body,
			}

			return jsonResponse(http.StatusOK, `{
			"id": "resp_123",
			"model": "gpt-5.2",
			"status": "completed",
			"output": [
				{
					"type": "message",
					"role": "assistant",
					"content": [
						{"type": "output_text", "text": "OpenAI assistant reply"}
					]
				}
			]
		}`), nil
		}),
	})
	defer restoreClient()

	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ProviderOpenAI,
			ThinkingLevel: "high",
			OpenAI: &configs.AIProviderConfig{
				APIKey:  "openai-key",
				BaseURL: "https://openai.example/proxy/v1",
			},
			Anthropic: &configs.AIProviderConfig{},
		},
	}
	conversation := &clientpb.AIConversation{
		Provider:     ProviderOpenAI,
		SystemPrompt: "Keep it brief.",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "assistant", Content: "Earlier answer."},
			{Role: "user", Content: "Explain the workflow."},
		},
	}

	runtime, err := ResolveRuntimeConfig(cfg, conversation)
	if err != nil {
		t.Fatalf("resolve runtime config: %v", err)
	}
	if runtime.Model != defaultOpenAIModel {
		t.Fatalf("unexpected default openai model: got=%q want=%q", runtime.Model, defaultOpenAIModel)
	}

	completion, err := CompleteConversation(context.Background(), runtime, conversation)
	if err != nil {
		t.Fatalf("complete conversation: %v", err)
	}

	request := <-requests
	if request.Path != "/proxy/v1/responses" {
		t.Fatalf("unexpected openai request path: got=%q want=%q", request.Path, "/proxy/v1/responses")
	}
	if request.Authorization != "Bearer openai-key" {
		t.Fatalf("unexpected authorization header: %q", request.Authorization)
	}
	if request.Body.Model != defaultOpenAIModel {
		t.Fatalf("unexpected openai model: got=%q want=%q", request.Body.Model, defaultOpenAIModel)
	}
	if request.Body.Reasoning == nil || request.Body.Reasoning.Effort != "high" {
		t.Fatalf("unexpected reasoning config: %+v", request.Body.Reasoning)
	}
	if len(request.Body.Input) != 3 {
		t.Fatalf("unexpected input message count: got=%d want=%d", len(request.Body.Input), 3)
	}
	if request.Body.Input[0].Role != "system" || request.Body.Input[0].Content != "Keep it brief." {
		t.Fatalf("unexpected system prompt in request: %+v", request.Body.Input[0])
	}
	if request.Body.Input[1].Role != "assistant" || request.Body.Input[1].Content != "Earlier answer." {
		t.Fatalf("unexpected assistant history in request: %+v", request.Body.Input[1])
	}
	if request.Body.Input[2].Role != "user" || request.Body.Input[2].Content != "Explain the workflow." {
		t.Fatalf("unexpected user prompt in request: %+v", request.Body.Input[2])
	}

	if completion.Content != "OpenAI assistant reply" {
		t.Fatalf("unexpected completion content: %q", completion.Content)
	}
	if completion.ProviderMessageID != "resp_123" {
		t.Fatalf("unexpected provider message id: %q", completion.ProviderMessageID)
	}
}

func TestCompleteConversationAnthropicUsesConfiguredCredentialsAndSettings(t *testing.T) {
	type capturedRequest struct {
		Path             string
		APIKey           string
		AnthropicVersion string
		Body             anthropicRequest
	}

	requests := make(chan capturedRequest, 1)
	restoreClient := SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			defer r.Body.Close()

			var body anthropicRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}

			requests <- capturedRequest{
				Path:             r.URL.Path,
				APIKey:           r.Header.Get("x-api-key"),
				AnthropicVersion: r.Header.Get("anthropic-version"),
				Body:             body,
			}

			return jsonResponse(http.StatusOK, `{
			"id": "msg_123",
			"model": "claude-sonnet-4-0",
			"stop_reason": "end_turn",
			"content": [
				{"type": "thinking", "text": "internal reasoning"},
				{"type": "text", "text": "Anthropic assistant reply"}
			]
		}`), nil
		}),
	})
	defer restoreClient()

	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ProviderAnthropic,
			ThinkingLevel: "medium",
			Anthropic: &configs.AIProviderConfig{
				APIKey:  "anthropic-key",
				BaseURL: "https://anthropic.example/edge",
			},
			OpenAI: &configs.AIProviderConfig{},
		},
	}
	conversation := &clientpb.AIConversation{
		Provider:     ProviderAnthropic,
		SystemPrompt: "Use short answers.",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
			{Role: "user", Content: "What changed?"},
		},
	}

	runtime, err := ResolveRuntimeConfig(cfg, conversation)
	if err != nil {
		t.Fatalf("resolve runtime config: %v", err)
	}
	if runtime.Model != defaultAnthropicModel {
		t.Fatalf("unexpected default anthropic model: got=%q want=%q", runtime.Model, defaultAnthropicModel)
	}

	completion, err := CompleteConversation(context.Background(), runtime, conversation)
	if err != nil {
		t.Fatalf("complete conversation: %v", err)
	}

	request := <-requests
	if request.Path != "/edge/v1/messages" {
		t.Fatalf("unexpected anthropic request path: got=%q want=%q", request.Path, "/edge/v1/messages")
	}
	if request.APIKey != "anthropic-key" {
		t.Fatalf("unexpected anthropic api key header: %q", request.APIKey)
	}
	if request.AnthropicVersion != anthropicAPIVersion {
		t.Fatalf("unexpected anthropic version header: got=%q want=%q", request.AnthropicVersion, anthropicAPIVersion)
	}
	if request.Body.Model != defaultAnthropicModel {
		t.Fatalf("unexpected anthropic model: got=%q want=%q", request.Body.Model, defaultAnthropicModel)
	}
	if request.Body.System != "Use short answers." {
		t.Fatalf("unexpected anthropic system prompt: %q", request.Body.System)
	}
	if request.Body.Thinking == nil || request.Body.Thinking.BudgetTokens != anthropicThinkingBudget("medium") {
		t.Fatalf("unexpected anthropic thinking config: %+v", request.Body.Thinking)
	}
	if len(request.Body.Messages) != 3 {
		t.Fatalf("unexpected anthropic message count: got=%d want=%d", len(request.Body.Messages), 3)
	}
	if request.Body.Messages[1].Role != "assistant" || request.Body.Messages[1].Content != "Hi there" {
		t.Fatalf("unexpected anthropic assistant history: %+v", request.Body.Messages[1])
	}

	if completion.Content != "Anthropic assistant reply" {
		t.Fatalf("unexpected completion content: %q", completion.Content)
	}
	if completion.ProviderMessageID != "msg_123" {
		t.Fatalf("unexpected provider message id: %q", completion.ProviderMessageID)
	}
	if completion.FinishReason != "end_turn" {
		t.Fatalf("unexpected finish reason: %q", completion.FinishReason)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
