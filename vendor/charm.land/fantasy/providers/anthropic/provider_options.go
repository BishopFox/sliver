// Package anthropic provides an implementation of the fantasy AI SDK for Anthropic's language models.
package anthropic

import (
	"encoding/json"

	"charm.land/fantasy"
)

// Global type identifiers for Anthropic-specific provider data.
const (
	TypeProviderOptions         = Name + ".options"
	TypeReasoningOptionMetadata = Name + ".reasoning_metadata"
	TypeProviderCacheControl    = Name + ".cache_control_options"
)

// Register Anthropic provider-specific types with the global registry.
func init() {
	fantasy.RegisterProviderType(TypeProviderOptions, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ProviderOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
	fantasy.RegisterProviderType(TypeReasoningOptionMetadata, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ReasoningOptionMetadata
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
	fantasy.RegisterProviderType(TypeProviderCacheControl, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ProviderCacheControlOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
}

// ProviderOptions represents additional options for the Anthropic provider.
type ProviderOptions struct {
	SendReasoning          *bool                   `json:"send_reasoning"`
	Thinking               *ThinkingProviderOption `json:"thinking"`
	DisableParallelToolUse *bool                   `json:"disable_parallel_tool_use"`
}

// Options implements the ProviderOptions interface.
func (o *ProviderOptions) Options() {}

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

// ThinkingProviderOption represents thinking options for the Anthropic provider.
type ThinkingProviderOption struct {
	BudgetTokens int64 `json:"budget_tokens"`
}

// ReasoningOptionMetadata represents reasoning metadata for the Anthropic provider.
type ReasoningOptionMetadata struct {
	Signature    string `json:"signature"`
	RedactedData string `json:"redacted_data"`
}

// Options implements the ProviderOptions interface.
func (*ReasoningOptionMetadata) Options() {}

// MarshalJSON implements custom JSON marshaling with type info for ReasoningOptionMetadata.
func (m ReasoningOptionMetadata) MarshalJSON() ([]byte, error) {
	type plain ReasoningOptionMetadata
	return fantasy.MarshalProviderType(TypeReasoningOptionMetadata, plain(m))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info for ReasoningOptionMetadata.
func (m *ReasoningOptionMetadata) UnmarshalJSON(data []byte) error {
	type plain ReasoningOptionMetadata
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*m = ReasoningOptionMetadata(p)
	return nil
}

// ProviderCacheControlOptions represents cache control options for the Anthropic provider.
type ProviderCacheControlOptions struct {
	CacheControl CacheControl `json:"cache_control"`
}

// Options implements the ProviderOptions interface.
func (*ProviderCacheControlOptions) Options() {}

// MarshalJSON implements custom JSON marshaling with type info for ProviderCacheControlOptions.
func (o ProviderCacheControlOptions) MarshalJSON() ([]byte, error) {
	type plain ProviderCacheControlOptions
	return fantasy.MarshalProviderType(TypeProviderCacheControl, plain(o))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info for ProviderCacheControlOptions.
func (o *ProviderCacheControlOptions) UnmarshalJSON(data []byte) error {
	type plain ProviderCacheControlOptions
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*o = ProviderCacheControlOptions(p)
	return nil
}

// CacheControl represents cache control settings for the Anthropic provider.
type CacheControl struct {
	Type string `json:"type"`
}

// NewProviderOptions creates new provider options for the Anthropic provider.
func NewProviderOptions(opts *ProviderOptions) fantasy.ProviderOptions {
	return fantasy.ProviderOptions{
		Name: opts,
	}
}

// NewProviderCacheControlOptions creates new cache control options for the Anthropic provider.
func NewProviderCacheControlOptions(opts *ProviderCacheControlOptions) fantasy.ProviderOptions {
	return fantasy.ProviderOptions{
		Name: opts,
	}
}

// ParseOptions parses provider options from a map for the Anthropic provider.
func ParseOptions(data map[string]any) (*ProviderOptions, error) {
	var options ProviderOptions
	if err := fantasy.ParseOptions(data, &options); err != nil {
		return nil, err
	}
	return &options, nil
}
