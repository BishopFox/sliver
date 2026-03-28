package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/openai/openai-go/v2/shared"
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
			"usage": {
				"input_tokens": 12000,
				"input_tokens_details": {},
				"output_tokens": 6000,
				"output_tokens_details": {},
				"total_tokens": 18000
			},
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
		`"type":"web_search"`,
		`"search_context_size":"medium"`,
		`"effort":"high"`,
		`"summary":"auto"`,
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
	if completion.ContextWindowUsage == nil {
		t.Fatal("expected context window usage to be captured")
	}
	if completion.ContextWindowUsage.TotalTokens != 18000 {
		t.Fatalf("unexpected total tokens: %d", completion.ContextWindowUsage.TotalTokens)
	}
	if completion.ContextWindowUsage.ContextWindowTokens != openAIFrontierContextWindowTokens {
		t.Fatalf("unexpected context window tokens: %d", completion.ContextWindowUsage.ContextWindowTokens)
	}
}

func TestCompleteConversationOpenAIUsesDefaultBaseURLWhenUnset(t *testing.T) {
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
			"id": "resp_default_base",
			"model": "gpt-5.4",
			"status": "completed",
			"output": [
				{
					"type": "message",
					"role": "assistant",
					"content": [
						{"type": "output_text", "text": "OpenAI default-base reply"}
					]
				}
			]
		}`), nil
		}),
	})
	defer restoreClient()

	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider: ProviderOpenAI,
			Model:    "gpt-5.4",
			OpenAI: &configs.AIProviderConfig{
				APIKey:          "openai-key",
				UseResponsesAPI: boolPtr(true),
			},
			Anthropic: &configs.AIProviderConfig{},
		},
	}
	conversation := &clientpb.AIConversation{
		Provider: ProviderOpenAI,
		Model:    "gpt-5.4",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Say hi."},
		},
	}

	runtime, err := ResolveRuntimeConfig(cfg, conversation)
	if err != nil {
		t.Fatalf("resolve runtime config: %v", err)
	}
	if runtime.BaseURL != "" {
		t.Fatalf("expected openai runtime base url to remain unset, got %q", runtime.BaseURL)
	}

	completion, err := CompleteConversation(context.Background(), runtime, conversation)
	if err != nil {
		t.Fatalf("complete conversation: %v", err)
	}

	request := <-requests
	if request.Path != "/v1/responses" {
		t.Fatalf("unexpected openai default-base request path: got=%q want=%q", request.Path, "/v1/responses")
	}
	if request.Authorization != "Bearer openai-key" {
		t.Fatalf("unexpected authorization header: %q", request.Authorization)
	}
	if !strings.Contains(request.Body, `"Say hi."`) {
		t.Fatalf("expected openai request body to contain the user prompt, got %s", request.Body)
	}
	if completion.Content != "OpenAI default-base reply" {
		t.Fatalf("unexpected completion content: %q", completion.Content)
	}
	if completion.ProviderMessageID != "resp_default_base" {
		t.Fatalf("unexpected provider message id: %q", completion.ProviderMessageID)
	}
}

func TestCompleteConversationOpenAIChatEnablesWebSearch(t *testing.T) {
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
			"id": "chatcmpl_openai_123",
			"object": "chat.completion",
			"model": "gpt-5.4",
			"usage": {
				"prompt_tokens": 32000,
				"completion_tokens": 4000,
				"total_tokens": 36000
			},
			"choices": [
				{
					"index": 0,
					"finish_reason": "stop",
					"message": {
						"role": "assistant",
						"content": "OpenAI chat reply"
					}
				}
			]
		}`), nil
		}),
	})
	defer restoreClient()

	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider: ProviderOpenAI,
			Model:    "gpt-5.4",
			OpenAI: &configs.AIProviderConfig{
				APIKey:          "openai-key",
				UseResponsesAPI: boolPtr(false),
			},
			Anthropic: &configs.AIProviderConfig{},
		},
	}
	conversation := &clientpb.AIConversation{
		Provider: ProviderOpenAI,
		Model:    "gpt-5.4",
		Messages: []*clientpb.AIConversationMessage{
			{Role: "user", Content: "Say hi."},
		},
	}

	runtime, err := ResolveRuntimeConfig(cfg, conversation)
	if err != nil {
		t.Fatalf("resolve runtime config: %v", err)
	}
	if runtime.UseResponsesAPI {
		t.Fatal("expected openai runtime to use chat completions for this test")
	}

	completion, err := CompleteConversation(context.Background(), runtime, conversation)
	if err != nil {
		t.Fatalf("complete conversation: %v", err)
	}

	request := <-requests
	if request.Path != "/v1/chat/completions" {
		t.Fatalf("unexpected openai chat request path: got=%q want=%q", request.Path, "/v1/chat/completions")
	}
	if request.Authorization != "Bearer openai-key" {
		t.Fatalf("unexpected authorization header: %q", request.Authorization)
	}
	for _, fragment := range []string{
		`"model":"gpt-5.4"`,
		`"web_search_options":{"search_context_size":"medium"}`,
		`"Say hi."`,
	} {
		if !strings.Contains(request.Body, fragment) {
			t.Fatalf("expected openai chat request body to contain %q, got %s", fragment, request.Body)
		}
	}
	if completion.Content != "OpenAI chat reply" {
		t.Fatalf("unexpected completion content: %q", completion.Content)
	}
	if completion.ProviderMessageID != "chatcmpl_openai_123" {
		t.Fatalf("unexpected provider message id: %q", completion.ProviderMessageID)
	}
	if completion.ContextWindowUsage == nil {
		t.Fatal("expected openai chat usage to be captured")
	}
	if completion.ContextWindowUsage.TotalTokens != 36000 {
		t.Fatalf("unexpected openai chat total tokens: %d", completion.ContextWindowUsage.TotalTokens)
	}
	if completion.ContextWindowUsage.ContextWindowTokens != openAI5_4ContextWindowTokens {
		t.Fatalf("unexpected openai chat context window tokens: %d", completion.ContextWindowUsage.ContextWindowTokens)
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

func TestResolveRuntimeConfigUsesConversationThinkingOverride(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ProviderOpenAI,
			Model:         "gpt-5.4",
			ThinkingLevel: "high",
			OpenAI: &configs.AIProviderConfig{
				APIKey:          "openai-key",
				UseResponsesAPI: boolPtr(true),
			},
			Anthropic: &configs.AIProviderConfig{},
		},
	}

	runtime, err := ResolveRuntimeConfig(cfg, &clientpb.AIConversation{
		Provider:      ProviderOpenAI,
		ThinkingLevel: "disabled",
	})
	if err != nil {
		t.Fatalf("resolve runtime config: %v", err)
	}
	if runtime.ThinkingLevel != "disabled" {
		t.Fatalf("expected conversation thinking override %q, got %q", "disabled", runtime.ThinkingLevel)
	}
}

func TestReasoningEffortForThinkingLevelSupportsXHigh(t *testing.T) {
	effort, ok := reasoningEffortForThinkingLevel("xhigh")
	if !ok {
		t.Fatal("expected xhigh reasoning effort to be supported")
	}
	if string(effort) != "xhigh" {
		t.Fatalf("expected reasoning effort %q, got %q", "xhigh", effort)
	}
}

func TestResponseReasoningParamSkipsSummaryForOpenAICompat(t *testing.T) {
	reasoning, ok := responseReasoningParam(&RuntimeConfig{
		Provider:      ProviderOpenAICompat,
		ThinkingLevel: "high",
	})
	if !ok {
		t.Fatal("expected response reasoning params to be available")
	}
	if reasoning.Effort != shared.ReasoningEffortHigh {
		t.Fatalf("expected reasoning effort %q, got %q", shared.ReasoningEffortHigh, reasoning.Effort)
	}
	if reasoning.Summary != "" {
		t.Fatalf("expected openai-compatible runtime to skip reasoning summary, got %q", reasoning.Summary)
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
			"usage": {
				"prompt_tokens": 14000,
				"completion_tokens": 2000,
				"total_tokens": 16000
			},
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
	if completion.ContextWindowUsage == nil {
		t.Fatal("expected openrouter usage to be captured")
	}
	if completion.ContextWindowUsage.TotalTokens != 16000 {
		t.Fatalf("unexpected openrouter total tokens: %d", completion.ContextWindowUsage.TotalTokens)
	}
	if completion.ContextWindowUsage.ContextWindowTokens != openAIFrontierContextWindowTokens {
		t.Fatalf("unexpected openrouter context window tokens: %d", completion.ContextWindowUsage.ContextWindowTokens)
	}
}

func TestConversationHistoryUsesExplicitContextFlagIndependentlyOfVisibility(t *testing.T) {
	systemPrompt, messages, err := conversationHistory(&clientpb.AIConversation{
		SystemPrompt: "Stay concise.",
		Messages: []*clientpb.AIConversationMessage{
			{
				Role:             "assistant",
				Content:          "Visible in the transcript but excluded from context.",
				Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
				Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
				IncludeInContext: boolPtr(false),
			},
			{
				Role:             "assistant",
				Content:          "UI-only but still included in context.",
				Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
				Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY,
				IncludeInContext: boolPtr(true),
			},
			{
				Role:             "user",
				Content:          "What changed?",
				Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
				Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
				IncludeInContext: boolPtr(true),
			},
		},
	})
	if err != nil {
		t.Fatalf("conversationHistory: %v", err)
	}
	if systemPrompt != "Stay concise." {
		t.Fatalf("unexpected system prompt: %q", systemPrompt)
	}
	if len(messages) != 2 {
		t.Fatalf("unexpected message count: got=%d want=%d", len(messages), 2)
	}
	if messages[0].Content != "UI-only but still included in context." {
		t.Fatalf("unexpected retained assistant message: %+v", messages[0])
	}
	if messages[1].Role != "user" || messages[1].Content != "What changed?" {
		t.Fatalf("unexpected trailing user message: %+v", messages[1])
	}
}

func TestConversationHistoryTreatsLeadingSystemMessageAsSystemPrompt(t *testing.T) {
	systemPrompt, messages, err := conversationHistory(&clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{
				Role:             "system",
				Content:          "Stay concise.",
				Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
				Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
				IncludeInContext: boolPtr(true),
			},
			{
				Role:             "user",
				Content:          "What changed?",
				Kind:             clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
				Visibility:       clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
				IncludeInContext: boolPtr(true),
			},
		},
	})
	if err != nil {
		t.Fatalf("conversationHistory: %v", err)
	}
	if systemPrompt != "Stay concise." {
		t.Fatalf("unexpected system prompt: %q", systemPrompt)
	}
	if len(messages) != 1 {
		t.Fatalf("unexpected message count: got=%d want=%d", len(messages), 1)
	}
	if messages[0].Role != "user" || messages[0].Content != "What changed?" {
		t.Fatalf("unexpected trailing user message: %+v", messages[0])
	}
}

func TestConversationHistoryFallsBackToVisibilityWhenContextFlagIsUnset(t *testing.T) {
	_, messages, err := conversationHistory(&clientpb.AIConversation{
		Messages: []*clientpb.AIConversationMessage{
			{
				Role:       "assistant",
				Content:    "Legacy context-visible message.",
				Kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
				Visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			},
			{
				Role:       "user",
				Content:    "Legacy prompt.",
				Kind:       clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT,
				Visibility: clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT,
			},
		},
	})
	if err != nil {
		t.Fatalf("conversationHistory: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("unexpected legacy message count: got=%d want=%d", len(messages), 2)
	}
	if messages[0].Content != "Legacy context-visible message." {
		t.Fatalf("unexpected legacy assistant message: %+v", messages[0])
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
