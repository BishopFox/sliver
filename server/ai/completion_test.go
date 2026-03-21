package ai

import (
	"context"
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
		Body          string
	}

	requests := make(chan capturedRequest, 1)
	restoreClient := SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			defer r.Body.Close()

			payload, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}

			requests <- capturedRequest{
				Path:          r.URL.Path,
				Authorization: r.Header.Get("Authorization"),
				Body:          string(payload),
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
				APIKey:          "openai-key",
				BaseURL:         "https://openai.example/proxy/v1",
				UseResponsesAPI: boolPtr(true),
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
	for _, fragment := range []string{
		`"model":"gpt-5.2"`,
		`"effort":"high"`,
		`"Keep it brief."`,
		`"Earlier answer."`,
		`"Explain the workflow."`,
	} {
		if !strings.Contains(request.Body, fragment) {
			t.Fatalf("expected openai request body to contain %q, got %s", fragment, request.Body)
		}
	}

	if completion.Content != "OpenAI assistant reply" {
		t.Fatalf("unexpected completion content: %q", completion.Content)
	}
	if completion.ProviderMessageID != "resp_123" {
		t.Fatalf("unexpected provider message id: %q", completion.ProviderMessageID)
	}
}

func TestResolveRuntimeConfigAnthropicRequiresAnInstalledDriver(t *testing.T) {
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

	runtime, err := ResolveRuntimeConfig(cfg, &clientpb.AIConversation{Provider: ProviderAnthropic})
	if runtime.Model != defaultAnthropicModel {
		t.Fatalf("unexpected default anthropic model: got=%q want=%q", runtime.Model, defaultAnthropicModel)
	}
	if err == nil {
		t.Fatal("expected anthropic runtime resolution to require a dedicated driver")
	}
	if !strings.Contains(err.Error(), `provider "anthropic"`) || !strings.Contains(err.Error(), "SDK driver") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRuntimeConfigOpenAICompatAllowsBaseURLWithoutAPIKey(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider: ProviderOpenAICompat,
			Model:    "gpt-oss-120b",
			OpenAICompat: &configs.AIProviderConfig{
				BaseURL: "http://127.0.0.1:8080/v1",
			},
		},
	}

	runtime, err := ResolveRuntimeConfig(cfg, &clientpb.AIConversation{
		Provider: ProviderOpenAICompat,
	})
	if err != nil {
		t.Fatalf("resolve runtime config: %v", err)
	}
	if runtime.Provider != ProviderOpenAICompat {
		t.Fatalf("expected provider %q, got %q", ProviderOpenAICompat, runtime.Provider)
	}
	if runtime.BaseURL != "http://127.0.0.1:8080/v1" {
		t.Fatalf("expected openai-compat base url, got %q", runtime.BaseURL)
	}
	if runtime.APIKey != "" {
		t.Fatalf("expected openai-compat api key to remain empty, got %q", runtime.APIKey)
	}
}

func TestCompleteConversationOpenAICompatUsesBaseURLWithoutAuth(t *testing.T) {
	type capturedRequest struct {
		Path          string
		Authorization string
		Body          string
	}

	requests := make(chan capturedRequest, 1)
	restoreClient := SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			defer r.Body.Close()

			payload, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}

			requests <- capturedRequest{
				Path:          r.URL.Path,
				Authorization: r.Header.Get("Authorization"),
				Body:          string(payload),
			}

			return jsonResponse(http.StatusOK, `{
			"id": "chatcmpl_123",
			"object": "chat.completion",
			"model": "gpt-oss-120b",
			"choices": [
				{
					"index": 0,
					"finish_reason": "stop",
					"message": {
						"role": "assistant",
						"content": "OpenAI-compatible assistant reply"
					}
				}
			],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 5,
				"total_tokens": 15
			}
		}`), nil
		}),
	})
	defer restoreClient()

	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider: ProviderOpenAICompat,
			Model:    "gpt-oss-120b",
			OpenAICompat: &configs.AIProviderConfig{
				BaseURL: "http://127.0.0.1:8080/v1",
			},
		},
	}
	conversation := &clientpb.AIConversation{
		Provider: ProviderOpenAICompat,
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Say hi."},
		},
	}

	runtime, err := ResolveRuntimeConfig(cfg, conversation)
	if err != nil {
		t.Fatalf("resolve runtime config: %v", err)
	}

	completion, err := CompleteConversation(context.Background(), runtime, conversation)
	if err != nil {
		t.Fatalf("complete conversation: %v", err)
	}

	request := <-requests
	if request.Path != "/v1/chat/completions" {
		t.Fatalf("unexpected openai-compatible request path: got=%q want=%q", request.Path, "/v1/chat/completions")
	}
	if request.Authorization != "" {
		t.Fatalf("expected no authorization header for unauthenticated openai-compatible endpoint, got %q", request.Authorization)
	}
	if !strings.Contains(request.Body, `"Say hi."`) {
		t.Fatalf("expected openai-compatible request body to contain the user prompt, got %s", request.Body)
	}
	if completion.Content != "OpenAI-compatible assistant reply" {
		t.Fatalf("unexpected completion content: %q", completion.Content)
	}
	if completion.ProviderMessageID != "chatcmpl_123" {
		t.Fatalf("unexpected provider message id: %q", completion.ProviderMessageID)
	}
}

func TestCompleteConversationOpenRouterUsesDefaultBaseURL(t *testing.T) {
	type capturedRequest struct {
		Path          string
		Authorization string
		Body          string
	}

	requests := make(chan capturedRequest, 1)
	restoreClient := SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			defer r.Body.Close()

			payload, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}

			requests <- capturedRequest{
				Path:          r.URL.Path,
				Authorization: r.Header.Get("Authorization"),
				Body:          string(payload),
			}

			return jsonResponse(http.StatusOK, `{
			"id": "chatcmpl_or_123",
			"object": "chat.completion",
			"model": "openai/gpt-5",
			"choices": [
				{
					"index": 0,
					"finish_reason": "stop",
					"message": {
						"role": "assistant",
						"content": "OpenRouter assistant reply"
					}
				}
			]
		}`), nil
		}),
	})
	defer restoreClient()

	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ProviderOpenRouter,
			ThinkingLevel: "medium",
			OpenRouter: &configs.AIProviderConfig{
				APIKey: "openrouter-key",
			},
		},
	}
	conversation := &clientpb.AIConversation{
		Provider: ProviderOpenRouter,
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Say hi."},
		},
	}

	runtime, err := ResolveRuntimeConfig(cfg, conversation)
	if err != nil {
		t.Fatalf("resolve runtime config: %v", err)
	}
	if runtime.Model != defaultOpenRouterModel {
		t.Fatalf("unexpected default openrouter model: got=%q want=%q", runtime.Model, defaultOpenRouterModel)
	}

	completion, err := CompleteConversation(context.Background(), runtime, conversation)
	if err != nil {
		t.Fatalf("complete conversation: %v", err)
	}

	request := <-requests
	if request.Path != "/api/v1/chat/completions" {
		t.Fatalf("unexpected openrouter request path: got=%q want=%q", request.Path, "/api/v1/chat/completions")
	}
	if request.Authorization != "Bearer openrouter-key" {
		t.Fatalf("unexpected authorization header: %q", request.Authorization)
	}
	for _, fragment := range []string{
		`"model":"openai/gpt-5"`,
		`"Say hi."`,
	} {
		if !strings.Contains(request.Body, fragment) {
			t.Fatalf("expected openrouter request body to contain %q, got %s", fragment, request.Body)
		}
	}
	if completion.Content != "OpenRouter assistant reply" {
		t.Fatalf("unexpected completion content: %q", completion.Content)
	}
	if completion.ProviderMessageID != "chatcmpl_or_123" {
		t.Fatalf("unexpected provider message id: %q", completion.ProviderMessageID)
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

func boolPtr(value bool) *bool {
	return &value
}
