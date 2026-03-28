package ai

import (
	"context"
	"net/http"
	"testing"
)

func TestResolveContextWindowUsageOpenRouterUsesModelsAPIForUnknownModels(t *testing.T) {
	openRouterContextWindowCache.Lock()
	openRouterContextWindowCache.entries = map[string]cachedOpenRouterContextWindow{}
	openRouterContextWindowCache.Unlock()

	requests := make(chan string, 1)
	restoreClient := SetHTTPClientForTests(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests <- r.URL.Path
			return jsonResponse(http.StatusOK, `{
				"data": [
					{
						"id": "anthropic/claude-sonnet-4-0",
						"top_provider": {
							"context_length": 200000
						}
					}
				]
			}`), nil
		}),
	})
	defer restoreClient()

	usage := resolveContextWindowUsage(context.Background(), &RuntimeConfig{
		Provider: ProviderOpenRouter,
		APIKey:   "openrouter-key",
	}, "anthropic/claude-sonnet-4-0", 24000, 1000, 25000)
	if usage == nil {
		t.Fatal("expected context window usage")
	}
	if usage.ContextWindowTokens != 200000 {
		t.Fatalf("unexpected context window tokens: %d", usage.ContextWindowTokens)
	}
	if usage.ContextWindowTokensEstimated {
		t.Fatal("expected OpenRouter model lookup to be treated as exact")
	}

	requestPath := <-requests
	if requestPath != "/api/v1/models" {
		t.Fatalf("unexpected models lookup path: %q", requestPath)
	}
}
