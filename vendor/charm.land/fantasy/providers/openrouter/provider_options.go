// Package openrouter provides an implementation of the fantasy AI SDK for OpenRouter's language models.
package openrouter

import (
	"encoding/json"

	"charm.land/fantasy"
)

// ReasoningEffort represents the reasoning effort level for OpenRouter models.
type ReasoningEffort string

const (
	// ReasoningEffortLow represents low reasoning effort.
	ReasoningEffortLow ReasoningEffort = "low"
	// ReasoningEffortMedium represents medium reasoning effort.
	ReasoningEffortMedium ReasoningEffort = "medium"
	// ReasoningEffortHigh represents high reasoning effort.
	ReasoningEffortHigh ReasoningEffort = "high"
)

// Global type identifiers for OpenRouter-specific provider data.
const (
	TypeProviderOptions  = Name + ".options"
	TypeProviderMetadata = Name + ".metadata"
)

// Register OpenRouter provider-specific types with the global registry.
func init() {
	fantasy.RegisterProviderType(TypeProviderOptions, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ProviderOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
	fantasy.RegisterProviderType(TypeProviderMetadata, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ProviderMetadata
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
}

// PromptTokensDetails represents details about prompt tokens for OpenRouter.
type PromptTokensDetails struct {
	CachedTokens int64 `json:"cached_tokens"`
}

// CompletionTokensDetails represents details about completion tokens for OpenRouter.
type CompletionTokensDetails struct {
	ReasoningTokens int64 `json:"reasoning_tokens"`
}

// CostDetails represents cost details for OpenRouter.
type CostDetails struct {
	UpstreamInferenceCost            float64 `json:"upstream_inference_cost"`
	UpstreamInferencePromptCost      float64 `json:"upstream_inference_prompt_cost"`
	UpstreamInferenceCompletionsCost float64 `json:"upstream_inference_completions_cost"`
}

// UsageAccounting represents usage accounting details for OpenRouter.
type UsageAccounting struct {
	PromptTokens            int64                   `json:"prompt_tokens"`
	PromptTokensDetails     PromptTokensDetails     `json:"prompt_tokens_details"`
	CompletionTokens        int64                   `json:"completion_tokens"`
	CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details"`
	TotalTokens             int64                   `json:"total_tokens"`
	Cost                    float64                 `json:"cost"`
	CostDetails             CostDetails             `json:"cost_details"`
}

// ProviderMetadata represents metadata from OpenRouter provider.
type ProviderMetadata struct {
	Provider string          `json:"provider"`
	Usage    UsageAccounting `json:"usage"`
}

// Options implements the ProviderOptionsData interface for ProviderMetadata.
func (*ProviderMetadata) Options() {}

// MarshalJSON implements custom JSON marshaling with type info for ProviderMetadata.
func (m ProviderMetadata) MarshalJSON() ([]byte, error) {
	type plain ProviderMetadata
	return fantasy.MarshalProviderType(TypeProviderMetadata, plain(m))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info for ProviderMetadata.
func (m *ProviderMetadata) UnmarshalJSON(data []byte) error {
	type plain ProviderMetadata
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*m = ProviderMetadata(p)
	return nil
}

// ReasoningOptions represents reasoning options for OpenRouter.
type ReasoningOptions struct {
	// Whether reasoning is enabled
	Enabled *bool `json:"enabled,omitempty"`
	// Whether to exclude reasoning from the response
	Exclude *bool `json:"exclude,omitempty"`
	// Maximum number of tokens to use for reasoning
	MaxTokens *int64 `json:"max_tokens,omitempty"`
	// Reasoning effort level: "low" | "medium" | "high"
	Effort *ReasoningEffort `json:"effort,omitempty"`
}

// Provider represents provider routing preferences for OpenRouter.
type Provider struct {
	// List of provider slugs to try in order (e.g. ["anthropic", "openai"])
	Order []string `json:"order,omitempty"`
	// Whether to allow backup providers when primary is unavailable (default: true)
	AllowFallbacks *bool `json:"allow_fallbacks,omitempty"`
	// Only use providers that support all parameters in your request (default: false)
	RequireParameters *bool `json:"require_parameters,omitempty"`
	// Control whether to use providers that may store data: "allow" | "deny"
	DataCollection *string `json:"data_collection,omitempty"`
	// List of provider slugs to allow for this request
	Only []string `json:"only,omitempty"`
	// List of provider slugs to skip for this request
	Ignore []string `json:"ignore,omitempty"`
	// List of quantization levels to filter by (e.g. ["int4", "int8"])
	Quantizations []string `json:"quantizations,omitempty"`
	// Sort providers by "price" | "throughput" | "latency"
	Sort *string `json:"sort,omitempty"`
}

// ProviderOptions represents additional options for OpenRouter provider.
type ProviderOptions struct {
	Reasoning    *ReasoningOptions `json:"reasoning,omitempty"`
	ExtraBody    map[string]any    `json:"extra_body,omitempty"`
	IncludeUsage *bool             `json:"include_usage,omitempty"`
	// Modify the likelihood of specified tokens appearing in the completion.
	// Accepts a map that maps tokens (specified by their token ID) to an associated bias value from -100 to 100.
	// The bias is added to the logits generated by the model prior to sampling.
	LogitBias map[string]int64 `json:"logit_bias,omitempty"`
	// Return the log probabilities of the tokens. Including logprobs will increase the response size.
	// Setting to true will return the log probabilities of the tokens that were generated.
	LogProbs *bool `json:"log_probs,omitempty"`
	// Whether to enable parallel function calling during tool use. Default to true.
	ParallelToolCalls *bool `json:"parallel_tool_calls,omitempty"`
	// A unique identifier representing your end-user, which can help OpenRouter to monitor and detect abuse.
	User *string `json:"user,omitempty"`
	// Provider routing preferences to control request routing behavior
	Provider *Provider `json:"provider,omitempty"`
	// TODO: add the web search plugin config
}

// Options implements the ProviderOptionsData interface for ProviderOptions.
func (*ProviderOptions) Options() {}

// MarshalJSON implements custom JSON marshaling with type info for ProviderOptions.
func (o ProviderOptions) MarshalJSON() ([]byte, error) {
	type plain ProviderOptions
	return fantasy.MarshalProviderType(TypeProviderOptions, plain(o))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info for ProviderOptions.
func (o *ProviderOptions) UnmarshalJSON(data []byte) error {
	type plain ProviderOptions
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*o = ProviderOptions(p)
	return nil
}

// ReasoningDetail represents a reasoning detail for OpenRouter.
type ReasoningDetail struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	Text      string `json:"text,omitempty"`
	Data      string `json:"data,omitempty"`
	Format    string `json:"format,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Signature string `json:"signature,omitempty"`
	Index     int    `json:"index"`
}

// ReasoningData represents reasoning data for OpenRouter.
type ReasoningData struct {
	Reasoning        string            `json:"reasoning"`
	ReasoningDetails []ReasoningDetail `json:"reasoning_details"`
}

// ReasoningEffortOption creates a pointer to a ReasoningEffort value for OpenRouter.
func ReasoningEffortOption(e ReasoningEffort) *ReasoningEffort {
	return &e
}

// NewProviderOptions creates new provider options for OpenRouter.
func NewProviderOptions(opts *ProviderOptions) fantasy.ProviderOptions {
	return fantasy.ProviderOptions{
		Name: opts,
	}
}

// ParseOptions parses provider options from a map for OpenRouter.
func ParseOptions(data map[string]any) (*ProviderOptions, error) {
	var options ProviderOptions
	if err := fantasy.ParseOptions(data, &options); err != nil {
		return nil, err
	}
	return &options, nil
}
