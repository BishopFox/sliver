package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	openAIFrontierContextWindowTokens     = 400000
	openAIChatContextWindowTokens         = 128000
	openAI4_1ContextWindowTokens          = 1047576
	openAI5_4ContextWindowTokens          = 1050000
	openAIReasoningContextWindowTokens    = 200000
	openAIRealtimeContextWindowTokens     = 32000
	openAIRealtimeMiniContextWindowTokens = 16000
	openRouterContextWindowCacheTTL       = 6 * time.Hour
)

// ContextWindowUsage captures the provider-reported token usage for a turn and,
// when available, the model context window used to render the client-side meter.
type ContextWindowUsage struct {
	InputTokens                  int64
	OutputTokens                 int64
	TotalTokens                  int64
	ContextWindowTokens          int64
	ContextWindowTokensEstimated bool
}

func newContextWindowUsage(inputTokens, outputTokens, totalTokens, windowTokens int64, estimated bool) *ContextWindowUsage {
	if inputTokens == 0 && outputTokens == 0 && totalTokens == 0 {
		return nil
	}
	return &ContextWindowUsage{
		InputTokens:                  inputTokens,
		OutputTokens:                 outputTokens,
		TotalTokens:                  totalTokens,
		ContextWindowTokens:          windowTokens,
		ContextWindowTokensEstimated: estimated,
	}
}

func cloneContextWindowUsage(usage *ContextWindowUsage) *ContextWindowUsage {
	if usage == nil {
		return nil
	}
	cloned := *usage
	return &cloned
}

func contextWindowUsageRank(usage *ContextWindowUsage) (int64, int64, int64) {
	if usage == nil {
		return 0, 0, 0
	}
	return usage.TotalTokens, usage.InputTokens, usage.OutputTokens
}

func mergeContextWindowUsage(current *ContextWindowUsage, candidate *ContextWindowUsage) *ContextWindowUsage {
	if candidate == nil {
		return cloneContextWindowUsage(current)
	}
	if current == nil {
		return cloneContextWindowUsage(candidate)
	}

	result := cloneContextWindowUsage(current)
	currentTotal, currentInput, currentOutput := contextWindowUsageRank(result)
	candidateTotal, candidateInput, candidateOutput := contextWindowUsageRank(candidate)
	if candidateTotal > currentTotal ||
		(candidateTotal == currentTotal && candidateInput > currentInput) ||
		(candidateTotal == currentTotal && candidateInput == currentInput && candidateOutput > currentOutput) {
		result = cloneContextWindowUsage(candidate)
	}

	if result.ContextWindowTokens == 0 && candidate.ContextWindowTokens > 0 {
		result.ContextWindowTokens = candidate.ContextWindowTokens
		result.ContextWindowTokensEstimated = candidate.ContextWindowTokensEstimated
	}
	if result.ContextWindowTokens > 0 && result.ContextWindowTokensEstimated &&
		candidate.ContextWindowTokens > 0 && !candidate.ContextWindowTokensEstimated {
		result.ContextWindowTokens = candidate.ContextWindowTokens
		result.ContextWindowTokensEstimated = false
	}

	return result
}

func resolveContextWindowUsage(ctx context.Context, runtime *RuntimeConfig, model string, inputTokens, outputTokens, totalTokens int64) *ContextWindowUsage {
	windowTokens, estimated := resolveContextWindowTokens(ctx, runtime, model)
	return newContextWindowUsage(inputTokens, outputTokens, totalTokens, windowTokens, estimated)
}

func resolveContextWindowTokens(ctx context.Context, runtime *RuntimeConfig, model string) (int64, bool) {
	model = normalizeContextWindowModelID(model)
	if model == "" {
		return 0, false
	}

	if tokens := openAIKnownContextWindowTokens(model); tokens > 0 {
		return tokens, true
	}

	if openRouterCompatibleRuntime(runtime) {
		if tokens, ok := openRouterContextWindowTokens(ctx, runtime, model); ok {
			return tokens, false
		}
	}

	return 0, false
}

func openAIKnownContextWindowTokens(model string) int64 {
	model = normalizeContextWindowModelID(model)
	switch {
	case model == "codex-mini-latest":
		return openAIReasoningContextWindowTokens
	case strings.HasPrefix(model, "o3"), strings.HasPrefix(model, "o4-"):
		return openAIReasoningContextWindowTokens
	case strings.HasPrefix(model, "gpt-5.4"):
		return openAI5_4ContextWindowTokens
	case strings.HasPrefix(model, "gpt-4.1"):
		return openAI4_1ContextWindowTokens
	case strings.HasPrefix(model, "gpt-5.1-chat"), strings.HasPrefix(model, "gpt-5-chat"):
		return openAIChatContextWindowTokens
	case strings.HasPrefix(model, "gpt-5"):
		return openAIFrontierContextWindowTokens
	case strings.HasPrefix(model, "gpt-4o-mini-realtime"):
		return openAIRealtimeMiniContextWindowTokens
	case strings.HasPrefix(model, "gpt-4o-realtime"):
		return openAIRealtimeContextWindowTokens
	case strings.HasPrefix(model, "gpt-4o"), strings.HasPrefix(model, "chatgpt-4o"):
		return openAIChatContextWindowTokens
	case strings.Contains(model, "gpt-4-turbo"):
		return openAIChatContextWindowTokens
	case strings.HasPrefix(model, "gpt-4"):
		return 8192
	default:
		return 0
	}
}

func normalizeContextWindowModelID(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if idx := strings.Index(model, "/"); idx >= 0 {
		model = model[idx+1:]
	}
	if idx := strings.Index(model, ":"); idx >= 0 {
		model = model[:idx]
	}
	return strings.TrimSpace(model)
}

func openRouterCompatibleRuntime(runtime *RuntimeConfig) bool {
	if runtime == nil {
		return false
	}
	switch NormalizeProviderName(runtime.Provider) {
	case ProviderOpenRouter:
		return true
	case ProviderOpenAICompat:
		baseURL := strings.ToLower(strings.TrimSpace(openAIBaseURL(runtime)))
		return strings.Contains(baseURL, "openrouter.ai")
	default:
		return false
	}
}

type openRouterModelsResponse struct {
	Data []openRouterModel `json:"data"`
}

type openRouterModel struct {
	ID          string `json:"id"`
	TopProvider struct {
		ContextLength int64 `json:"context_length"`
	} `json:"top_provider"`
}

type cachedOpenRouterContextWindow struct {
	tokens    int64
	expiresAt time.Time
}

var openRouterContextWindowCache = struct {
	sync.RWMutex
	entries map[string]cachedOpenRouterContextWindow
}{
	entries: map[string]cachedOpenRouterContextWindow{},
}

func openRouterContextWindowTokens(ctx context.Context, runtime *RuntimeConfig, model string) (int64, bool) {
	model = normalizeContextWindowModelID(model)
	if model == "" {
		return 0, false
	}

	cacheKey := openRouterContextWindowCacheKey(runtime, model)
	if tokens, ok := cachedOpenRouterContextWindowTokens(cacheKey); ok {
		return tokens, true
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterModelsURL(runtime), nil)
	if err != nil {
		return 0, false
	}
	if runtime != nil && !runtime.SkipAuth && strings.TrimSpace(runtime.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(runtime.APIKey))
	}
	for key, value := range providerHeaders(runtime) {
		req.Header.Set(key, value)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false
	}

	payload := &openRouterModelsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(payload); err != nil {
		return 0, false
	}

	for _, candidate := range payload.Data {
		candidateID := normalizeContextWindowModelID(candidate.ID)
		if candidateID != model {
			continue
		}
		if candidate.TopProvider.ContextLength <= 0 {
			return 0, false
		}
		storeOpenRouterContextWindowTokens(cacheKey, candidate.TopProvider.ContextLength)
		return candidate.TopProvider.ContextLength, true
	}

	return 0, false
}

func openRouterModelsURL(runtime *RuntimeConfig) string {
	baseURL := strings.TrimSpace(openAIBaseURL(runtime))
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed == nil {
		return defaultOpenRouterBaseURL + "/models"
	}

	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		path = strings.TrimSuffix(path, "/chat/completions")
	case strings.HasSuffix(path, "/responses"):
		path = strings.TrimSuffix(path, "/responses")
	case path == "":
		path = "/api/v1"
	}
	if !strings.HasSuffix(path, "/models") {
		path += "/models"
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func openRouterContextWindowCacheKey(runtime *RuntimeConfig, model string) string {
	return openRouterModelsURL(runtime) + "|" + normalizeContextWindowModelID(model)
}

func cachedOpenRouterContextWindowTokens(cacheKey string) (int64, bool) {
	if cacheKey == "" {
		return 0, false
	}

	now := time.Now()
	openRouterContextWindowCache.RLock()
	cached, ok := openRouterContextWindowCache.entries[cacheKey]
	openRouterContextWindowCache.RUnlock()
	if !ok || cached.tokens <= 0 || now.After(cached.expiresAt) {
		if ok && now.After(cached.expiresAt) {
			openRouterContextWindowCache.Lock()
			delete(openRouterContextWindowCache.entries, cacheKey)
			openRouterContextWindowCache.Unlock()
		}
		return 0, false
	}

	return cached.tokens, true
}

func storeOpenRouterContextWindowTokens(cacheKey string, tokens int64) {
	if cacheKey == "" || tokens <= 0 {
		return
	}

	openRouterContextWindowCache.Lock()
	openRouterContextWindowCache.entries[cacheKey] = cachedOpenRouterContextWindow{
		tokens:    tokens,
		expiresAt: time.Now().Add(openRouterContextWindowCacheTTL),
	}
	openRouterContextWindowCache.Unlock()
}
