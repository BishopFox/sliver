// Package openai provides an implementation of the fantasy AI SDK for OpenAI's language models.
package openai

import (
	"encoding/json"

	"charm.land/fantasy"
	"github.com/openai/openai-go/v2"
)

// ReasoningEffort represents the reasoning effort level for OpenAI models.
type ReasoningEffort string

const (
	// ReasoningEffortMinimal represents minimal reasoning effort.
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	// ReasoningEffortLow represents low reasoning effort.
	ReasoningEffortLow ReasoningEffort = "low"
	// ReasoningEffortMedium represents medium reasoning effort.
	ReasoningEffortMedium ReasoningEffort = "medium"
	// ReasoningEffortHigh represents high reasoning effort.
	ReasoningEffortHigh ReasoningEffort = "high"
)

// Global type identifiers for OpenAI-specific provider data.
const (
	TypeProviderOptions     = Name + ".options"
	TypeProviderFileOptions = Name + ".file_options"
	TypeProviderMetadata    = Name + ".metadata"
)

// Register OpenAI provider-specific types with the global registry.
func init() {
	fantasy.RegisterProviderType(TypeProviderOptions, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ProviderOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
	fantasy.RegisterProviderType(TypeProviderFileOptions, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ProviderFileOptions
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

// ProviderMetadata represents additional metadata from OpenAI provider.
type ProviderMetadata struct {
	Logprobs                 []openai.ChatCompletionTokenLogprob `json:"logprobs"`
	AcceptedPredictionTokens int64                               `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int64                               `json:"rejected_prediction_tokens"`
}

// Options implements the ProviderOptions interface.
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

// ProviderOptions represents additional options for OpenAI provider.
type ProviderOptions struct {
	LogitBias           map[string]int64 `json:"logit_bias"`
	LogProbs            *bool            `json:"log_probs"`
	TopLogProbs         *int64           `json:"top_log_probs"`
	ParallelToolCalls   *bool            `json:"parallel_tool_calls"`
	User                *string          `json:"user"`
	ReasoningEffort     *ReasoningEffort `json:"reasoning_effort"`
	MaxCompletionTokens *int64           `json:"max_completion_tokens"`
	TextVerbosity       *string          `json:"text_verbosity"`
	Prediction          map[string]any   `json:"prediction"`
	Store               *bool            `json:"store"`
	Metadata            map[string]any   `json:"metadata"`
	PromptCacheKey      *string          `json:"prompt_cache_key"`
	SafetyIdentifier    *string          `json:"safety_identifier"`
	ServiceTier         *string          `json:"service_tier"`
	StructuredOutputs   *bool            `json:"structured_outputs"`
}

// Options implements the ProviderOptions interface.
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

// ProviderFileOptions represents file options for OpenAI provider.
type ProviderFileOptions struct {
	ImageDetail string `json:"image_detail"`
}

// Options implements the ProviderOptions interface.
func (*ProviderFileOptions) Options() {}

// MarshalJSON implements custom JSON marshaling with type info for ProviderFileOptions.
func (o ProviderFileOptions) MarshalJSON() ([]byte, error) {
	type plain ProviderFileOptions
	return fantasy.MarshalProviderType(TypeProviderFileOptions, plain(o))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info for ProviderFileOptions.
func (o *ProviderFileOptions) UnmarshalJSON(data []byte) error {
	type plain ProviderFileOptions
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*o = ProviderFileOptions(p)
	return nil
}

// ReasoningEffortOption creates a pointer to a ReasoningEffort value.
func ReasoningEffortOption(e ReasoningEffort) *ReasoningEffort {
	return &e
}

// NewProviderOptions creates new provider options for OpenAI.
func NewProviderOptions(opts *ProviderOptions) fantasy.ProviderOptions {
	return fantasy.ProviderOptions{
		Name: opts,
	}
}

// NewProviderFileOptions creates new file options for OpenAI.
func NewProviderFileOptions(opts *ProviderFileOptions) fantasy.ProviderOptions {
	return fantasy.ProviderOptions{
		Name: opts,
	}
}

// ParseOptions parses provider options from a map.
func ParseOptions(data map[string]any) (*ProviderOptions, error) {
	var options ProviderOptions
	if err := fantasy.ParseOptions(data, &options); err != nil {
		return nil, err
	}
	return &options, nil
}
