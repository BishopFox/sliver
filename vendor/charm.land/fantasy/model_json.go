package fantasy

import (
	"encoding/json"
	"fmt"
)

// UnmarshalJSON implements json.Unmarshaler for Call.
func (c *Call) UnmarshalJSON(data []byte) error {
	var aux struct {
		Prompt           Prompt                     `json:"prompt"`
		MaxOutputTokens  *int64                     `json:"max_output_tokens"`
		Temperature      *float64                   `json:"temperature"`
		TopP             *float64                   `json:"top_p"`
		TopK             *int64                     `json:"top_k"`
		PresencePenalty  *float64                   `json:"presence_penalty"`
		FrequencyPenalty *float64                   `json:"frequency_penalty"`
		Tools            []json.RawMessage          `json:"tools"`
		ToolChoice       *ToolChoice                `json:"tool_choice"`
		ProviderOptions  map[string]json.RawMessage `json:"provider_options"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.Prompt = aux.Prompt
	c.MaxOutputTokens = aux.MaxOutputTokens
	c.Temperature = aux.Temperature
	c.TopP = aux.TopP
	c.TopK = aux.TopK
	c.PresencePenalty = aux.PresencePenalty
	c.FrequencyPenalty = aux.FrequencyPenalty
	c.ToolChoice = aux.ToolChoice

	// Unmarshal Tools slice
	c.Tools = make([]Tool, len(aux.Tools))
	for i, rawTool := range aux.Tools {
		tool, err := UnmarshalTool(rawTool)
		if err != nil {
			return fmt.Errorf("failed to unmarshal tool at index %d: %w", i, err)
		}
		c.Tools[i] = tool
	}

	// Unmarshal ProviderOptions
	if len(aux.ProviderOptions) > 0 {
		options, err := UnmarshalProviderOptions(aux.ProviderOptions)
		if err != nil {
			return err
		}
		c.ProviderOptions = options
	}

	return nil
}

// UnmarshalJSON implements json.Unmarshaler for Response.
func (r *Response) UnmarshalJSON(data []byte) error {
	var aux struct {
		Content          json.RawMessage            `json:"content"`
		FinishReason     FinishReason               `json:"finish_reason"`
		Usage            Usage                      `json:"usage"`
		Warnings         []CallWarning              `json:"warnings"`
		ProviderMetadata map[string]json.RawMessage `json:"provider_metadata"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	r.FinishReason = aux.FinishReason
	r.Usage = aux.Usage
	r.Warnings = aux.Warnings

	// Unmarshal ResponseContent (need to know the type definition)
	// If ResponseContent is []Content:
	var rawContent []json.RawMessage
	if err := json.Unmarshal(aux.Content, &rawContent); err != nil {
		return err
	}

	content := make([]Content, len(rawContent))
	for i, rawItem := range rawContent {
		item, err := UnmarshalContent(rawItem)
		if err != nil {
			return fmt.Errorf("failed to unmarshal content at index %d: %w", i, err)
		}
		content[i] = item
	}
	r.Content = content

	// Unmarshal ProviderMetadata
	if len(aux.ProviderMetadata) > 0 {
		metadata, err := UnmarshalProviderMetadata(aux.ProviderMetadata)
		if err != nil {
			return err
		}
		r.ProviderMetadata = metadata
	}

	return nil
}

// MarshalJSON implements json.Marshaler for StreamPart.
func (s StreamPart) MarshalJSON() ([]byte, error) {
	type alias StreamPart
	aux := struct {
		alias
		Error string `json:"error,omitempty"`
	}{
		alias: (alias)(s),
	}

	// Marshal error to string
	if s.Error != nil {
		aux.Error = s.Error.Error()
	}

	// Clear the original Error field to avoid duplicate marshaling
	aux.alias.Error = nil

	return json.Marshal(aux)
}

// UnmarshalJSON implements json.Unmarshaler for StreamPart.
func (s *StreamPart) UnmarshalJSON(data []byte) error {
	type alias StreamPart
	aux := struct {
		*alias
		Error            string                     `json:"error"`
		ProviderMetadata map[string]json.RawMessage `json:"provider_metadata"`
	}{
		alias: (*alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Unmarshal error string back to error type
	if aux.Error != "" {
		s.Error = fmt.Errorf("%s", aux.Error)
	}

	// Unmarshal ProviderMetadata
	if len(aux.ProviderMetadata) > 0 {
		metadata, err := UnmarshalProviderMetadata(aux.ProviderMetadata)
		if err != nil {
			return err
		}
		s.ProviderMetadata = metadata
	}

	return nil
}
