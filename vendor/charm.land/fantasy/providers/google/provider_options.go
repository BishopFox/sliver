// Package google provides an implementation of the fantasy AI SDK for Google's language models.
package google

import (
	"encoding/json"

	"charm.land/fantasy"
)

// Global type identifiers for Google-specific provider data.
const (
	TypeProviderOptions   = Name + ".options"
	TypeReasoningMetadata = Name + ".reasoning_metadata"
)

// Register Google provider-specific types with the global registry.
func init() {
	fantasy.RegisterProviderType(TypeProviderOptions, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ProviderOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
	fantasy.RegisterProviderType(TypeReasoningMetadata, func(data []byte) (fantasy.ProviderOptionsData, error) {
		var v ReasoningMetadata
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return &v, nil
	})
}

// ThinkingConfig represents thinking configuration for the Google provider.
type ThinkingConfig struct {
	ThinkingBudget  *int64 `json:"thinking_budget"`
	IncludeThoughts *bool  `json:"include_thoughts"`
}

// ReasoningMetadata represents reasoning metadata for the Google provider.
type ReasoningMetadata struct {
	Signature string `json:"signature"`
	ToolID    string `json:"tool_id"`
}

// Options implements the ProviderOptionsData interface for ReasoningMetadata.
func (m *ReasoningMetadata) Options() {}

// MarshalJSON implements custom JSON marshaling with type info for ReasoningMetadata.
func (m ReasoningMetadata) MarshalJSON() ([]byte, error) {
	type plain ReasoningMetadata
	return fantasy.MarshalProviderType(TypeReasoningMetadata, plain(m))
}

// UnmarshalJSON implements custom JSON unmarshaling with type info for ReasoningMetadata.
func (m *ReasoningMetadata) UnmarshalJSON(data []byte) error {
	type plain ReasoningMetadata
	var p plain
	if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
		return err
	}
	*m = ReasoningMetadata(p)
	return nil
}

// SafetySetting represents safety settings for the Google provider.
type SafetySetting struct {
	// 'HARM_CATEGORY_UNSPECIFIED',
	// 'HARM_CATEGORY_HATE_SPEECH',
	// 'HARM_CATEGORY_DANGEROUS_CONTENT',
	// 'HARM_CATEGORY_HARASSMENT',
	// 'HARM_CATEGORY_SEXUALLY_EXPLICIT',
	// 'HARM_CATEGORY_CIVIC_INTEGRITY',
	Category string `json:"category"`

	// 'HARM_BLOCK_THRESHOLD_UNSPECIFIED',
	// 'BLOCK_LOW_AND_ABOVE',
	// 'BLOCK_MEDIUM_AND_ABOVE',
	// 'BLOCK_ONLY_HIGH',
	// 'BLOCK_NONE',
	// 'OFF',
	Threshold string `json:"threshold"`
}

// ProviderOptions represents additional options for the Google provider.
type ProviderOptions struct {
	ThinkingConfig *ThinkingConfig `json:"thinking_config"`

	// Optional.
	// The name of the cached content used as context to serve the prediction.
	// Format: cachedContents/{cachedContent}
	CachedContent string `json:"cached_content"`

	// Optional. A list of unique safety settings for blocking unsafe content.
	SafetySettings []SafetySetting `json:"safety_settings"`
	// 'HARM_BLOCK_THRESHOLD_UNSPECIFIED',
	// 'BLOCK_LOW_AND_ABOVE',
	// 'BLOCK_MEDIUM_AND_ABOVE',
	// 'BLOCK_ONLY_HIGH',
	// 'BLOCK_NONE',
	// 'OFF',
	Threshold string `json:"threshold"`
}

// Options implements the ProviderOptionsData interface for ProviderOptions.
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

// ParseOptions parses provider options from a map for the Google provider.
func ParseOptions(data map[string]any) (*ProviderOptions, error) {
	var options ProviderOptions
	if err := fantasy.ParseOptions(data, &options); err != nil {
		return nil, err
	}
	return &options, nil
}
