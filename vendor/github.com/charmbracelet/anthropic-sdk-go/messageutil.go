package anthropic

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/anthropic-sdk-go/internal/paramutil"
	"github.com/charmbracelet/anthropic-sdk-go/packages/param"
)

// Accumulate builds up the Message incrementally from a MessageStreamEvent. The Message then can be used as
// any other Message, except with the caveat that the Message.JSON field which normally can be used to inspect
// the JSON sent over the network may not be populated fully.
//
//	message := anthropic.Message{}
//	for stream.Next() {
//		event := stream.Current()
//		message.Accumulate(event)
//	}
func (acc *Message) Accumulate(event MessageStreamEventUnion) error {
	if acc == nil {
		return fmt.Errorf("accumulate: cannot accumulate into nil Message")
	}

	switch event := event.AsAny().(type) {
	case MessageStartEvent:
		*acc = event.Message
	case MessageDeltaEvent:
		acc.StopReason = event.Delta.StopReason
		acc.StopSequence = event.Delta.StopSequence
		acc.Usage.OutputTokens = event.Usage.OutputTokens
	case ContentBlockStartEvent:
		acc.Content = append(acc.Content, ContentBlockUnion{})
		err := acc.Content[len(acc.Content)-1].UnmarshalJSON([]byte(event.ContentBlock.RawJSON()))
		if err != nil {
			return err
		}
	case ContentBlockDeltaEvent:
		if len(acc.Content) == 0 {
			return fmt.Errorf("received event of type %s but there was no content block", event.Type)
		}
		cb := &acc.Content[len(acc.Content)-1]
		switch delta := event.Delta.AsAny().(type) {
		case TextDelta:
			cb.Text += delta.Text
		case InputJSONDelta:
			if len(delta.PartialJSON) != 0 {
				if string(cb.Input) == "{}" {
					cb.Input = []byte(delta.PartialJSON)
				} else {
					cb.Input = append(cb.Input, []byte(delta.PartialJSON)...)
				}
			}
		case ThinkingDelta:
			cb.Thinking += delta.Thinking
		case SignatureDelta:
			cb.Signature += delta.Signature
		case CitationsDelta:
			citation := TextCitationUnion{}
			err := citation.UnmarshalJSON([]byte(delta.Citation.RawJSON()))
			if err != nil {
				return fmt.Errorf("could not unmarshal citation delta into citation type: %w", err)
			}
			cb.Citations = append(cb.Citations, citation)
		}
	case MessageStopEvent:
		// Re-marshal the accumulated message to update JSON.raw so that AsAny()
		// returns the accumulated data rather than the original stream data
		accJSON, err := json.Marshal(acc)
		if err != nil {
			return fmt.Errorf("error converting accumulated message to JSON: %w", err)
		}
		acc.JSON.raw = string(accJSON)
	case ContentBlockStopEvent:
		// Re-marshal the content block to update JSON.raw so that AsAny()
		// returns the accumulated data rather than the original stream data
		if len(acc.Content) == 0 {
			return fmt.Errorf("received event of type %s but there was no content block", event.Type)
		}
		contentBlock := &acc.Content[len(acc.Content)-1]
		cbJSON, err := json.Marshal(contentBlock)
		if err != nil {
			return fmt.Errorf("error converting content block to JSON: %w", err)
		}
		contentBlock.JSON.raw = string(cbJSON)
	}

	return nil
}

// ToParam converters

func (r Message) ToParam() MessageParam {
	var p MessageParam
	p.Role = MessageParamRole(r.Role)
	p.Content = make([]ContentBlockParamUnion, len(r.Content))
	for i, c := range r.Content {
		p.Content[i] = c.ToParam()
	}
	return p
}

func (r ContentBlockUnion) ToParam() ContentBlockParamUnion {
	return r.AsAny().toParamUnion()
}

func (variant TextBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfText: &p}
}

func (variant ToolUseBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfToolUse: &p}
}

func (variant WebSearchToolResultBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfWebSearchToolResult: &p}
}

func (variant ServerToolUseBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfServerToolUse: &p}
}

func (variant ThinkingBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfThinking: &p}
}

func (variant RedactedThinkingBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfRedactedThinking: &p}
}

func (r RedactedThinkingBlock) ToParam() RedactedThinkingBlockParam {
	var p RedactedThinkingBlockParam
	p.Type = r.Type
	p.Data = r.Data
	return p
}

func (r ToolUseBlock) ToParam() ToolUseBlockParam {
	var toolUse ToolUseBlockParam
	toolUse.Type = r.Type
	toolUse.ID = r.ID
	toolUse.Input = r.Input
	toolUse.Name = r.Name
	return toolUse
}

func (citationVariant CitationCharLocation) toParamUnion() TextCitationParamUnion {
	var citationParam CitationCharLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.DocumentTitle = paramutil.ToOpt(citationVariant.DocumentTitle, citationVariant.JSON.DocumentTitle)
	citationParam.CitedText = citationVariant.CitedText
	citationParam.DocumentIndex = citationVariant.DocumentIndex
	citationParam.EndCharIndex = citationVariant.EndCharIndex
	citationParam.StartCharIndex = citationVariant.StartCharIndex
	return TextCitationParamUnion{OfCharLocation: &citationParam}
}

func (citationVariant CitationPageLocation) toParamUnion() TextCitationParamUnion {
	var citationParam CitationPageLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.DocumentTitle = paramutil.ToOpt(citationVariant.DocumentTitle, citationVariant.JSON.DocumentTitle)
	citationParam.DocumentIndex = citationVariant.DocumentIndex
	citationParam.EndPageNumber = citationVariant.EndPageNumber
	citationParam.StartPageNumber = citationVariant.StartPageNumber
	return TextCitationParamUnion{OfPageLocation: &citationParam}
}

func (citationVariant CitationContentBlockLocation) toParamUnion() TextCitationParamUnion {
	var citationParam CitationContentBlockLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.DocumentTitle = paramutil.ToOpt(citationVariant.DocumentTitle, citationVariant.JSON.DocumentTitle)
	citationParam.CitedText = citationVariant.CitedText
	citationParam.DocumentIndex = citationVariant.DocumentIndex
	citationParam.EndBlockIndex = citationVariant.EndBlockIndex
	citationParam.StartBlockIndex = citationVariant.StartBlockIndex
	return TextCitationParamUnion{OfContentBlockLocation: &citationParam}
}

func (citationVariant CitationsSearchResultLocation) toParamUnion() TextCitationParamUnion {
	var citationParam CitationSearchResultLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.CitedText = citationVariant.CitedText
	citationParam.Title = paramutil.ToOpt(citationVariant.Title, citationVariant.JSON.Title)
	return TextCitationParamUnion{OfSearchResultLocation: &citationParam}
}

func (citationVariant CitationsWebSearchResultLocation) toParamUnion() TextCitationParamUnion {
	var citationParam CitationWebSearchResultLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.CitedText = citationVariant.CitedText
	citationParam.Title = paramutil.ToOpt(citationVariant.Title, citationVariant.JSON.Title)
	return TextCitationParamUnion{OfWebSearchResultLocation: &citationParam}
}

func (r TextBlock) ToParam() TextBlockParam {
	var p TextBlockParam
	p.Type = r.Type
	p.Text = r.Text

	// Distinguish between a nil and zero length slice, since some compatible
	// APIs may not require citations.
	if r.Citations != nil {
		p.Citations = make([]TextCitationParamUnion, 0, len(r.Citations))
	}

	for _, citation := range r.Citations {
		p.Citations = append(p.Citations, citation.AsAny().toParamUnion())
	}

	return p
}

func (r ThinkingBlock) ToParam() ThinkingBlockParam {
	var p ThinkingBlockParam
	p.Type = r.Type
	p.Signature = r.Signature
	p.Thinking = r.Thinking
	return p
}

func (r ServerToolUseBlock) ToParam() ServerToolUseBlockParam {
	var p ServerToolUseBlockParam
	p.Type = r.Type
	p.ID = r.ID
	p.Input = r.Input
	p.Name = ServerToolUseBlockParamName(r.Name)
	return p
}

func (r WebSearchToolResultBlock) ToParam() WebSearchToolResultBlockParam {
	var p WebSearchToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	p.Content = r.Content.ToParam()
	return p
}

func (r WebSearchResultBlock) ToParam() WebSearchResultBlockParam {
	var p WebSearchResultBlockParam
	p.Type = r.Type
	p.EncryptedContent = r.EncryptedContent
	p.Title = r.Title
	p.URL = r.URL
	p.PageAge = paramutil.ToOpt(r.PageAge, r.JSON.PageAge)
	return p
}

func (r WebSearchToolResultBlockContentUnion) ToParam() WebSearchToolResultBlockParamContentUnion {
	var p WebSearchToolResultBlockParamContentUnion

	if len(r.OfWebSearchResultBlockArray) > 0 {
		for _, block := range r.OfWebSearchResultBlockArray {
			p.OfWebSearchToolResultBlockItem = append(p.OfWebSearchToolResultBlockItem, block.ToParam())
		}
		return p
	}

	p.OfRequestWebSearchToolResultError = &WebSearchToolRequestErrorParam{
		ErrorCode: WebSearchToolResultErrorCode(r.ErrorCode),
	}
	return p
}

func (variant WebFetchToolResultBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfWebFetchToolResult: &p}
}

func (variant CodeExecutionToolResultBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfCodeExecutionToolResult: &p}
}

func (variant BashCodeExecutionToolResultBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfBashCodeExecutionToolResult: &p}
}

func (variant TextEditorCodeExecutionToolResultBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfTextEditorCodeExecutionToolResult: &p}
}

func (variant ToolSearchToolResultBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfToolSearchToolResult: &p}
}

func (variant ContainerUploadBlock) toParamUnion() ContentBlockParamUnion {
	p := variant.ToParam()
	return ContentBlockParamUnion{OfContainerUpload: &p}
}

func (r WebFetchToolResultBlock) ToParam() WebFetchToolResultBlockParam {
	var p WebFetchToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	return p
}

func (r ContainerUploadBlock) ToParam() ContainerUploadBlockParam {
	var p ContainerUploadBlockParam
	p.Type = r.Type
	p.FileID = r.FileID
	return p
}

func (r BashCodeExecutionToolResultBlock) ToParam() BashCodeExecutionToolResultBlockParam {
	var p BashCodeExecutionToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID

	if r.Content.JSON.ErrorCode.Valid() {
		p.Content.OfRequestBashCodeExecutionToolResultError = &BashCodeExecutionToolResultErrorParam{
			ErrorCode: BashCodeExecutionToolResultErrorCode(r.Content.ErrorCode),
		}
	} else {
		requestBashContentResult := &BashCodeExecutionResultBlockParam{
			ReturnCode: r.Content.ReturnCode,
			Stderr:     r.Content.Stderr,
			Stdout:     r.Content.Stdout,
		}
		for _, block := range r.Content.Content {
			requestBashContentResult.Content = append(requestBashContentResult.Content, block.ToParam())
		}
		p.Content.OfRequestBashCodeExecutionResultBlock = requestBashContentResult
	}

	return p
}

func (r BashCodeExecutionOutputBlock) ToParam() BashCodeExecutionOutputBlockParam {
	var p BashCodeExecutionOutputBlockParam
	p.Type = r.Type
	p.FileID = r.FileID
	return p
}

func (r CodeExecutionToolResultBlock) ToParam() CodeExecutionToolResultBlockParam {
	var p CodeExecutionToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	if r.Content.JSON.ErrorCode.Valid() {
		p.Content.OfRequestCodeExecutionToolResultError = &CodeExecutionToolResultErrorParam{
			ErrorCode: r.Content.ErrorCode,
		}
	} else {
		p.Content.OfRequestCodeExecutionResultBlock = &CodeExecutionResultBlockParam{
			ReturnCode: r.Content.ReturnCode,
			Stderr:     r.Content.Stderr,
			Stdout:     r.Content.Stdout,
		}
		for _, block := range r.Content.Content {
			p.Content.OfRequestCodeExecutionResultBlock.Content = append(p.Content.OfRequestCodeExecutionResultBlock.Content, block.ToParam())
		}
	}
	return p
}

func (r CodeExecutionOutputBlock) ToParam() CodeExecutionOutputBlockParam {
	var p CodeExecutionOutputBlockParam
	p.Type = r.Type
	p.FileID = r.FileID
	return p
}

func (r TextEditorCodeExecutionToolResultBlock) ToParam() TextEditorCodeExecutionToolResultBlockParam {
	var p TextEditorCodeExecutionToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	if r.Content.JSON.ErrorCode.Valid() {
		p.Content.OfRequestTextEditorCodeExecutionToolResultError = &TextEditorCodeExecutionToolResultErrorParam{
			ErrorCode:    TextEditorCodeExecutionToolResultErrorCode(r.Content.ErrorCode),
			ErrorMessage: paramutil.ToOpt(r.Content.ErrorMessage, r.Content.JSON.ErrorMessage),
		}
	} else {
		p.Content = param.Override[TextEditorCodeExecutionToolResultBlockParamContentUnion](r.Content.RawJSON())
	}
	return p
}

func (r ToolSearchToolResultBlock) ToParam() ToolSearchToolResultBlockParam {
	var p ToolSearchToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	if r.Content.JSON.ErrorCode.Valid() {
		p.Content.OfRequestToolSearchToolResultError = &ToolSearchToolResultErrorParam{
			ErrorCode: ToolSearchToolResultErrorCode(r.Content.ErrorCode),
		}
	} else {
		p.Content.OfRequestToolSearchToolSearchResultBlock = &ToolSearchToolSearchResultBlockParam{}
		for _, block := range r.Content.ToolReferences {
			p.Content.OfRequestToolSearchToolSearchResultBlock.ToolReferences = append(
				p.Content.OfRequestToolSearchToolSearchResultBlock.ToolReferences,
				block.ToParam(),
			)
		}
	}
	return p
}

func (r ToolReferenceBlock) ToParam() ToolReferenceBlockParam {
	var p ToolReferenceBlockParam
	p.Type = r.Type
	p.ToolName = r.ToolName
	return p
}
