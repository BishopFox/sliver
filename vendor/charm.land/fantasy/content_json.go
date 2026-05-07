package fantasy

import (
	"encoding/json"
	"errors"
	"fmt"
)

// contentJSON is a helper type for JSON serialization of Content in Response.
type contentJSON struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// messagePartJSON is a helper type for JSON serialization of MessagePart.
type messagePartJSON struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// toolResultOutputJSON is a helper type for JSON serialization of ToolResultOutputContent.
type toolResultOutputJSON struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// toolJSON is a helper type for JSON serialization of Tool.
type toolJSON struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// MarshalJSON implements json.Marshaler for TextContent.
func (t TextContent) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		Text             string           `json:"text"`
		ProviderMetadata ProviderMetadata `json:"provider_metadata,omitempty"`
	}{
		Text:             t.Text,
		ProviderMetadata: t.ProviderMetadata,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(contentJSON{
		Type: string(ContentTypeText),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for TextContent.
func (t *TextContent) UnmarshalJSON(data []byte) error {
	var cj contentJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return err
	}

	var aux struct {
		Text             string                     `json:"text"`
		ProviderMetadata map[string]json.RawMessage `json:"provider_metadata,omitempty"`
	}

	if err := json.Unmarshal(cj.Data, &aux); err != nil {
		return err
	}

	t.Text = aux.Text

	if len(aux.ProviderMetadata) > 0 {
		metadata, err := UnmarshalProviderMetadata(aux.ProviderMetadata)
		if err != nil {
			return err
		}
		t.ProviderMetadata = metadata
	}

	return nil
}

// MarshalJSON implements json.Marshaler for ReasoningContent.
func (r ReasoningContent) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		Text             string           `json:"text"`
		ProviderMetadata ProviderMetadata `json:"provider_metadata,omitempty"`
	}{
		Text:             r.Text,
		ProviderMetadata: r.ProviderMetadata,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(contentJSON{
		Type: string(ContentTypeReasoning),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ReasoningContent.
func (r *ReasoningContent) UnmarshalJSON(data []byte) error {
	var cj contentJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return err
	}

	var aux struct {
		Text             string                     `json:"text"`
		ProviderMetadata map[string]json.RawMessage `json:"provider_metadata,omitempty"`
	}

	if err := json.Unmarshal(cj.Data, &aux); err != nil {
		return err
	}

	r.Text = aux.Text

	if len(aux.ProviderMetadata) > 0 {
		metadata, err := UnmarshalProviderMetadata(aux.ProviderMetadata)
		if err != nil {
			return err
		}
		r.ProviderMetadata = metadata
	}

	return nil
}

// MarshalJSON implements json.Marshaler for FileContent.
func (f FileContent) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		MediaType        string           `json:"media_type"`
		Data             []byte           `json:"data"`
		ProviderMetadata ProviderMetadata `json:"provider_metadata,omitempty"`
	}{
		MediaType:        f.MediaType,
		Data:             f.Data,
		ProviderMetadata: f.ProviderMetadata,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(contentJSON{
		Type: string(ContentTypeFile),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for FileContent.
func (f *FileContent) UnmarshalJSON(data []byte) error {
	var cj contentJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return err
	}

	var aux struct {
		MediaType        string                     `json:"media_type"`
		Data             []byte                     `json:"data"`
		ProviderMetadata map[string]json.RawMessage `json:"provider_metadata,omitempty"`
	}

	if err := json.Unmarshal(cj.Data, &aux); err != nil {
		return err
	}

	f.MediaType = aux.MediaType
	f.Data = aux.Data

	if len(aux.ProviderMetadata) > 0 {
		metadata, err := UnmarshalProviderMetadata(aux.ProviderMetadata)
		if err != nil {
			return err
		}
		f.ProviderMetadata = metadata
	}

	return nil
}

// MarshalJSON implements json.Marshaler for SourceContent.
func (s SourceContent) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		SourceType       SourceType       `json:"source_type"`
		ID               string           `json:"id"`
		URL              string           `json:"url,omitempty"`
		Title            string           `json:"title,omitempty"`
		MediaType        string           `json:"media_type,omitempty"`
		Filename         string           `json:"filename,omitempty"`
		ProviderMetadata ProviderMetadata `json:"provider_metadata,omitempty"`
	}{
		SourceType:       s.SourceType,
		ID:               s.ID,
		URL:              s.URL,
		Title:            s.Title,
		MediaType:        s.MediaType,
		Filename:         s.Filename,
		ProviderMetadata: s.ProviderMetadata,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(contentJSON{
		Type: string(ContentTypeSource),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for SourceContent.
func (s *SourceContent) UnmarshalJSON(data []byte) error {
	var cj contentJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return err
	}

	var aux struct {
		SourceType       SourceType                 `json:"source_type"`
		ID               string                     `json:"id"`
		URL              string                     `json:"url,omitempty"`
		Title            string                     `json:"title,omitempty"`
		MediaType        string                     `json:"media_type,omitempty"`
		Filename         string                     `json:"filename,omitempty"`
		ProviderMetadata map[string]json.RawMessage `json:"provider_metadata,omitempty"`
	}

	if err := json.Unmarshal(cj.Data, &aux); err != nil {
		return err
	}

	s.SourceType = aux.SourceType
	s.ID = aux.ID
	s.URL = aux.URL
	s.Title = aux.Title
	s.MediaType = aux.MediaType
	s.Filename = aux.Filename

	if len(aux.ProviderMetadata) > 0 {
		metadata, err := UnmarshalProviderMetadata(aux.ProviderMetadata)
		if err != nil {
			return err
		}
		s.ProviderMetadata = metadata
	}

	return nil
}

// MarshalJSON implements json.Marshaler for ToolCallContent.
func (t ToolCallContent) MarshalJSON() ([]byte, error) {
	var validationErrMsg *string
	if t.ValidationError != nil {
		msg := t.ValidationError.Error()
		validationErrMsg = &msg
	}
	dataBytes, err := json.Marshal(struct {
		ToolCallID       string           `json:"tool_call_id"`
		ToolName         string           `json:"tool_name"`
		Input            string           `json:"input"`
		ProviderExecuted bool             `json:"provider_executed"`
		ProviderMetadata ProviderMetadata `json:"provider_metadata,omitempty"`
		Invalid          bool             `json:"invalid,omitempty"`
		ValidationError  *string          `json:"validation_error,omitempty"`
	}{
		ToolCallID:       t.ToolCallID,
		ToolName:         t.ToolName,
		Input:            t.Input,
		ProviderExecuted: t.ProviderExecuted,
		ProviderMetadata: t.ProviderMetadata,
		Invalid:          t.Invalid,
		ValidationError:  validationErrMsg,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(contentJSON{
		Type: string(ContentTypeToolCall),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ToolCallContent.
func (t *ToolCallContent) UnmarshalJSON(data []byte) error {
	var cj contentJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return err
	}

	var aux struct {
		ToolCallID       string                     `json:"tool_call_id"`
		ToolName         string                     `json:"tool_name"`
		Input            string                     `json:"input"`
		ProviderExecuted bool                       `json:"provider_executed"`
		ProviderMetadata map[string]json.RawMessage `json:"provider_metadata,omitempty"`
		Invalid          bool                       `json:"invalid,omitempty"`
		ValidationError  *string                    `json:"validation_error,omitempty"`
	}

	if err := json.Unmarshal(cj.Data, &aux); err != nil {
		return err
	}

	t.ToolCallID = aux.ToolCallID
	t.ToolName = aux.ToolName
	t.Input = aux.Input
	t.ProviderExecuted = aux.ProviderExecuted
	t.Invalid = aux.Invalid
	if aux.ValidationError != nil {
		t.ValidationError = errors.New(*aux.ValidationError)
	}

	if len(aux.ProviderMetadata) > 0 {
		metadata, err := UnmarshalProviderMetadata(aux.ProviderMetadata)
		if err != nil {
			return err
		}
		t.ProviderMetadata = metadata
	}

	return nil
}

// MarshalJSON implements json.Marshaler for ToolResultContent.
func (t ToolResultContent) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		ToolCallID       string                  `json:"tool_call_id"`
		ToolName         string                  `json:"tool_name"`
		Result           ToolResultOutputContent `json:"result"`
		ClientMetadata   string                  `json:"client_metadata,omitempty"`
		ProviderExecuted bool                    `json:"provider_executed"`
		ProviderMetadata ProviderMetadata        `json:"provider_metadata,omitempty"`
	}{
		ToolCallID:       t.ToolCallID,
		ToolName:         t.ToolName,
		Result:           t.Result,
		ClientMetadata:   t.ClientMetadata,
		ProviderExecuted: t.ProviderExecuted,
		ProviderMetadata: t.ProviderMetadata,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(contentJSON{
		Type: string(ContentTypeToolResult),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ToolResultContent.
func (t *ToolResultContent) UnmarshalJSON(data []byte) error {
	var cj contentJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return err
	}

	var aux struct {
		ToolCallID       string                     `json:"tool_call_id"`
		ToolName         string                     `json:"tool_name"`
		Result           json.RawMessage            `json:"result"`
		ClientMetadata   string                     `json:"client_metadata,omitempty"`
		ProviderExecuted bool                       `json:"provider_executed"`
		ProviderMetadata map[string]json.RawMessage `json:"provider_metadata,omitempty"`
	}

	if err := json.Unmarshal(cj.Data, &aux); err != nil {
		return err
	}

	t.ToolCallID = aux.ToolCallID
	t.ToolName = aux.ToolName
	t.ClientMetadata = aux.ClientMetadata
	t.ProviderExecuted = aux.ProviderExecuted

	// Unmarshal the Result field
	result, err := UnmarshalToolResultOutputContent(aux.Result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal tool result output: %w", err)
	}
	t.Result = result

	if len(aux.ProviderMetadata) > 0 {
		metadata, err := UnmarshalProviderMetadata(aux.ProviderMetadata)
		if err != nil {
			return err
		}
		t.ProviderMetadata = metadata
	}

	return nil
}

// MarshalJSON implements json.Marshaler for ToolResultOutputContentText.
func (t ToolResultOutputContentText) MarshalJSON() ([]byte, error) {
	type alias ToolResultOutputContentText
	dataBytes, err := json.Marshal(alias(t))
	if err != nil {
		return nil, err
	}

	return json.Marshal(toolResultOutputJSON{
		Type: string(ToolResultContentTypeText),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ToolResultOutputContentText.
func (t *ToolResultOutputContentText) UnmarshalJSON(data []byte) error {
	var tr toolResultOutputJSON
	if err := json.Unmarshal(data, &tr); err != nil {
		return err
	}

	type alias ToolResultOutputContentText
	var temp alias

	if err := json.Unmarshal(tr.Data, &temp); err != nil {
		return err
	}

	*t = ToolResultOutputContentText(temp)
	return nil
}

// MarshalJSON implements json.Marshaler for ToolResultOutputContentError.
func (t ToolResultOutputContentError) MarshalJSON() ([]byte, error) {
	errMsg := ""
	if t.Error != nil {
		errMsg = t.Error.Error()
	}
	dataBytes, err := json.Marshal(struct {
		Error string `json:"error"`
	}{
		Error: errMsg,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(toolResultOutputJSON{
		Type: string(ToolResultContentTypeError),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ToolResultOutputContentError.
func (t *ToolResultOutputContentError) UnmarshalJSON(data []byte) error {
	var tr toolResultOutputJSON
	if err := json.Unmarshal(data, &tr); err != nil {
		return err
	}

	var temp struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(tr.Data, &temp); err != nil {
		return err
	}
	if temp.Error != "" {
		t.Error = errors.New(temp.Error)
	}
	return nil
}

// MarshalJSON implements json.Marshaler for ToolResultOutputContentMedia.
func (t ToolResultOutputContentMedia) MarshalJSON() ([]byte, error) {
	type alias ToolResultOutputContentMedia
	dataBytes, err := json.Marshal(alias(t))
	if err != nil {
		return nil, err
	}

	return json.Marshal(toolResultOutputJSON{
		Type: string(ToolResultContentTypeMedia),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ToolResultOutputContentMedia.
func (t *ToolResultOutputContentMedia) UnmarshalJSON(data []byte) error {
	var tr toolResultOutputJSON
	if err := json.Unmarshal(data, &tr); err != nil {
		return err
	}

	type alias ToolResultOutputContentMedia
	var temp alias

	if err := json.Unmarshal(tr.Data, &temp); err != nil {
		return err
	}

	*t = ToolResultOutputContentMedia(temp)
	return nil
}

// MarshalJSON implements json.Marshaler for TextPart.
func (t TextPart) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		Text            string          `json:"text"`
		ProviderOptions ProviderOptions `json:"provider_options,omitempty"`
	}{
		Text:            t.Text,
		ProviderOptions: t.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(messagePartJSON{
		Type: string(ContentTypeText),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for TextPart.
func (t *TextPart) UnmarshalJSON(data []byte) error {
	var mpj messagePartJSON
	if err := json.Unmarshal(data, &mpj); err != nil {
		return err
	}

	var aux struct {
		Text            string                     `json:"text"`
		ProviderOptions map[string]json.RawMessage `json:"provider_options,omitempty"`
	}

	if err := json.Unmarshal(mpj.Data, &aux); err != nil {
		return err
	}

	t.Text = aux.Text

	if len(aux.ProviderOptions) > 0 {
		options, err := UnmarshalProviderOptions(aux.ProviderOptions)
		if err != nil {
			return err
		}
		t.ProviderOptions = options
	}

	return nil
}

// MarshalJSON implements json.Marshaler for ReasoningPart.
func (r ReasoningPart) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		Text            string          `json:"text"`
		ProviderOptions ProviderOptions `json:"provider_options,omitempty"`
	}{
		Text:            r.Text,
		ProviderOptions: r.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(messagePartJSON{
		Type: string(ContentTypeReasoning),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ReasoningPart.
func (r *ReasoningPart) UnmarshalJSON(data []byte) error {
	var mpj messagePartJSON
	if err := json.Unmarshal(data, &mpj); err != nil {
		return err
	}

	var aux struct {
		Text            string                     `json:"text"`
		ProviderOptions map[string]json.RawMessage `json:"provider_options,omitempty"`
	}

	if err := json.Unmarshal(mpj.Data, &aux); err != nil {
		return err
	}

	r.Text = aux.Text

	if len(aux.ProviderOptions) > 0 {
		options, err := UnmarshalProviderOptions(aux.ProviderOptions)
		if err != nil {
			return err
		}
		r.ProviderOptions = options
	}

	return nil
}

// MarshalJSON implements json.Marshaler for FilePart.
func (f FilePart) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		Filename        string          `json:"filename"`
		Data            []byte          `json:"data"`
		MediaType       string          `json:"media_type"`
		ProviderOptions ProviderOptions `json:"provider_options,omitempty"`
	}{
		Filename:        f.Filename,
		Data:            f.Data,
		MediaType:       f.MediaType,
		ProviderOptions: f.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(messagePartJSON{
		Type: string(ContentTypeFile),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for FilePart.
func (f *FilePart) UnmarshalJSON(data []byte) error {
	var mpj messagePartJSON
	if err := json.Unmarshal(data, &mpj); err != nil {
		return err
	}

	var aux struct {
		Filename        string                     `json:"filename"`
		Data            []byte                     `json:"data"`
		MediaType       string                     `json:"media_type"`
		ProviderOptions map[string]json.RawMessage `json:"provider_options,omitempty"`
	}

	if err := json.Unmarshal(mpj.Data, &aux); err != nil {
		return err
	}

	f.Filename = aux.Filename
	f.Data = aux.Data
	f.MediaType = aux.MediaType

	if len(aux.ProviderOptions) > 0 {
		options, err := UnmarshalProviderOptions(aux.ProviderOptions)
		if err != nil {
			return err
		}
		f.ProviderOptions = options
	}

	return nil
}

// MarshalJSON implements json.Marshaler for ToolCallPart.
func (t ToolCallPart) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		ToolCallID       string          `json:"tool_call_id"`
		ToolName         string          `json:"tool_name"`
		Input            string          `json:"input"`
		ProviderExecuted bool            `json:"provider_executed"`
		ProviderOptions  ProviderOptions `json:"provider_options,omitempty"`
	}{
		ToolCallID:       t.ToolCallID,
		ToolName:         t.ToolName,
		Input:            t.Input,
		ProviderExecuted: t.ProviderExecuted,
		ProviderOptions:  t.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(messagePartJSON{
		Type: string(ContentTypeToolCall),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ToolCallPart.
func (t *ToolCallPart) UnmarshalJSON(data []byte) error {
	var mpj messagePartJSON
	if err := json.Unmarshal(data, &mpj); err != nil {
		return err
	}

	var aux struct {
		ToolCallID       string                     `json:"tool_call_id"`
		ToolName         string                     `json:"tool_name"`
		Input            string                     `json:"input"`
		ProviderExecuted bool                       `json:"provider_executed"`
		ProviderOptions  map[string]json.RawMessage `json:"provider_options,omitempty"`
	}

	if err := json.Unmarshal(mpj.Data, &aux); err != nil {
		return err
	}

	t.ToolCallID = aux.ToolCallID
	t.ToolName = aux.ToolName
	t.Input = aux.Input
	t.ProviderExecuted = aux.ProviderExecuted

	if len(aux.ProviderOptions) > 0 {
		options, err := UnmarshalProviderOptions(aux.ProviderOptions)
		if err != nil {
			return err
		}
		t.ProviderOptions = options
	}

	return nil
}

// MarshalJSON implements json.Marshaler for ToolResultPart.
func (t ToolResultPart) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		ToolCallID      string                  `json:"tool_call_id"`
		Output          ToolResultOutputContent `json:"output"`
		ProviderOptions ProviderOptions         `json:"provider_options,omitempty"`
	}{
		ToolCallID:      t.ToolCallID,
		Output:          t.Output,
		ProviderOptions: t.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(messagePartJSON{
		Type: string(ContentTypeToolResult),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ToolResultPart.
func (t *ToolResultPart) UnmarshalJSON(data []byte) error {
	var mpj messagePartJSON
	if err := json.Unmarshal(data, &mpj); err != nil {
		return err
	}

	var aux struct {
		ToolCallID      string                     `json:"tool_call_id"`
		Output          json.RawMessage            `json:"output"`
		ProviderOptions map[string]json.RawMessage `json:"provider_options,omitempty"`
	}

	if err := json.Unmarshal(mpj.Data, &aux); err != nil {
		return err
	}

	t.ToolCallID = aux.ToolCallID

	// Unmarshal the Output field
	output, err := UnmarshalToolResultOutputContent(aux.Output)
	if err != nil {
		return fmt.Errorf("failed to unmarshal tool result output: %w", err)
	}
	t.Output = output

	if len(aux.ProviderOptions) > 0 {
		options, err := UnmarshalProviderOptions(aux.ProviderOptions)
		if err != nil {
			return err
		}
		t.ProviderOptions = options
	}

	return nil
}

// UnmarshalJSON implements json.Unmarshaler for Message.
func (m *Message) UnmarshalJSON(data []byte) error {
	var aux struct {
		Role            MessageRole                `json:"role"`
		Content         []json.RawMessage          `json:"content"`
		ProviderOptions map[string]json.RawMessage `json:"provider_options"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	m.Role = aux.Role

	m.Content = make([]MessagePart, len(aux.Content))
	for i, rawPart := range aux.Content {
		part, err := UnmarshalMessagePart(rawPart)
		if err != nil {
			return fmt.Errorf("failed to unmarshal message part at index %d: %w", i, err)
		}
		m.Content[i] = part
	}

	if len(aux.ProviderOptions) > 0 {
		options, err := UnmarshalProviderOptions(aux.ProviderOptions)
		if err != nil {
			return err
		}
		m.ProviderOptions = options
	}

	return nil
}

// MarshalJSON implements json.Marshaler for FunctionTool.
func (f FunctionTool) MarshalJSON() ([]byte, error) {
	dataBytes, err := json.Marshal(struct {
		Name            string          `json:"name"`
		Description     string          `json:"description"`
		InputSchema     map[string]any  `json:"input_schema"`
		ProviderOptions ProviderOptions `json:"provider_options,omitempty"`
	}{
		Name:            f.Name,
		Description:     f.Description,
		InputSchema:     f.InputSchema,
		ProviderOptions: f.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(toolJSON{
		Type: string(ToolTypeFunction),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for FunctionTool.
func (f *FunctionTool) UnmarshalJSON(data []byte) error {
	var tj toolJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return err
	}

	var aux struct {
		Name            string                     `json:"name"`
		Description     string                     `json:"description"`
		InputSchema     map[string]any             `json:"input_schema"`
		ProviderOptions map[string]json.RawMessage `json:"provider_options,omitempty"`
	}

	if err := json.Unmarshal(tj.Data, &aux); err != nil {
		return err
	}

	f.Name = aux.Name
	f.Description = aux.Description
	f.InputSchema = aux.InputSchema

	if len(aux.ProviderOptions) > 0 {
		options, err := UnmarshalProviderOptions(aux.ProviderOptions)
		if err != nil {
			return err
		}
		f.ProviderOptions = options
	}

	return nil
}

// MarshalJSON implements json.Marshaler for ProviderDefinedTool.
func (p ProviderDefinedTool) MarshalJSON() ([]byte, error) {
	type alias ProviderDefinedTool
	dataBytes, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}

	return json.Marshal(toolJSON{
		Type: string(ToolTypeProviderDefined),
		Data: json.RawMessage(dataBytes),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ProviderDefinedTool.
func (p *ProviderDefinedTool) UnmarshalJSON(data []byte) error {
	var tj toolJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return err
	}

	type alias ProviderDefinedTool
	var aux alias

	if err := json.Unmarshal(tj.Data, &aux); err != nil {
		return err
	}

	*p = ProviderDefinedTool(aux)

	return nil
}

// UnmarshalTool unmarshals JSON into the appropriate Tool type.
func UnmarshalTool(data []byte) (Tool, error) {
	var tj toolJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, err
	}

	switch ToolType(tj.Type) {
	case ToolTypeFunction:
		var tool FunctionTool
		if err := tool.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return tool, nil
	case ToolTypeProviderDefined:
		var tool ProviderDefinedTool
		if err := tool.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return tool, nil
	default:
		return nil, fmt.Errorf("unknown tool type: %s", tj.Type)
	}
}

// UnmarshalContent unmarshals JSON into the appropriate Content type.
func UnmarshalContent(data []byte) (Content, error) {
	var cj contentJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return nil, err
	}

	switch ContentType(cj.Type) {
	case ContentTypeText:
		var content TextContent
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	case ContentTypeReasoning:
		var content ReasoningContent
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	case ContentTypeFile:
		var content FileContent
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	case ContentTypeSource:
		var content SourceContent
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	case ContentTypeToolCall:
		var content ToolCallContent
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	case ContentTypeToolResult:
		var content ToolResultContent
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	default:
		return nil, fmt.Errorf("unknown content type: %s", cj.Type)
	}
}

// UnmarshalMessagePart unmarshals JSON into the appropriate MessagePart type.
func UnmarshalMessagePart(data []byte) (MessagePart, error) {
	var mpj messagePartJSON
	if err := json.Unmarshal(data, &mpj); err != nil {
		return nil, err
	}

	switch ContentType(mpj.Type) {
	case ContentTypeText:
		var part TextPart
		if err := part.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return part, nil
	case ContentTypeReasoning:
		var part ReasoningPart
		if err := part.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return part, nil
	case ContentTypeFile:
		var part FilePart
		if err := part.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return part, nil
	case ContentTypeToolCall:
		var part ToolCallPart
		if err := part.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return part, nil
	case ContentTypeToolResult:
		var part ToolResultPart
		if err := part.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return part, nil
	default:
		return nil, fmt.Errorf("unknown message part type: %s", mpj.Type)
	}
}

// UnmarshalToolResultOutputContent unmarshals JSON into the appropriate ToolResultOutputContent type.
func UnmarshalToolResultOutputContent(data []byte) (ToolResultOutputContent, error) {
	var troj toolResultOutputJSON
	if err := json.Unmarshal(data, &troj); err != nil {
		return nil, err
	}

	switch ToolResultContentType(troj.Type) {
	case ToolResultContentTypeText:
		var content ToolResultOutputContentText
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	case ToolResultContentTypeError:
		var content ToolResultOutputContentError
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	case ToolResultContentTypeMedia:
		var content ToolResultOutputContentMedia
		if err := content.UnmarshalJSON(data); err != nil {
			return nil, err
		}
		return content, nil
	default:
		return nil, fmt.Errorf("unknown tool result output content type: %s", troj.Type)
	}
}
