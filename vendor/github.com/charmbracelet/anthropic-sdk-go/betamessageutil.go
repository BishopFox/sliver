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
func (acc *BetaMessage) Accumulate(event BetaRawMessageStreamEventUnion) error {
	if acc == nil {
		return fmt.Errorf("accumulate: cannot accumulate into nil Message")
	}

	switch event := event.AsAny().(type) {
	case BetaRawMessageStartEvent:
		*acc = event.Message
	case BetaRawMessageDeltaEvent:
		acc.StopReason = event.Delta.StopReason
		acc.StopSequence = event.Delta.StopSequence
		acc.Usage.OutputTokens = event.Usage.OutputTokens
		acc.Usage.Iterations = event.Usage.Iterations
		acc.ContextManagement = event.ContextManagement
	case BetaRawContentBlockStartEvent:
		acc.Content = append(acc.Content, BetaContentBlockUnion{})
		err := acc.Content[len(acc.Content)-1].UnmarshalJSON([]byte(event.ContentBlock.RawJSON()))
		if err != nil {
			return err
		}
	case BetaRawContentBlockDeltaEvent:
		if len(acc.Content) == 0 {
			return fmt.Errorf("received event of type %s but there was no content block", event.Type)
		}
		cb := &acc.Content[len(acc.Content)-1]
		switch delta := event.Delta.AsAny().(type) {
		case BetaTextDelta:
			cb.Text += delta.Text
		case BetaInputJSONDelta:
			if len(delta.PartialJSON) != 0 {
				if string(cb.Input) == "{}" {
					cb.Input = []byte(delta.PartialJSON)
				} else {
					cb.Input = append(cb.Input, []byte(delta.PartialJSON)...)
				}
			}
		case BetaThinkingDelta:
			cb.Thinking += delta.Thinking
		case BetaSignatureDelta:
			cb.Signature += delta.Signature
		case BetaCitationsDelta:
			citation := BetaTextCitationUnion{}
			err := citation.UnmarshalJSON([]byte(delta.Citation.RawJSON()))
			if err != nil {
				return fmt.Errorf("could not unmarshal citation delta into citation type: %w", err)
			}
			cb.Citations = append(cb.Citations, citation)
		case BetaCompactionContentBlockDelta:
			cb.Content.OfString = delta.Content
		}
	case BetaRawMessageStopEvent:
		// Re-marshal the accumulated message to update JSON.raw so that AsAny()
		// returns the accumulated data rather than the original stream data
		accJSON, err := json.Marshal(acc)
		if err != nil {
			return fmt.Errorf("error converting accumulated message to JSON: %w", err)
		}
		acc.JSON.raw = string(accJSON)
	case BetaRawContentBlockStopEvent:
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

// Param converters

func (r BetaContentBlockUnion) ToParam() BetaContentBlockParamUnion {
	return r.AsAny().toParamUnion()
}

func (variant BetaTextBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfText: &p}
}

func (variant BetaToolUseBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfToolUse: &p}
}

func (variant BetaThinkingBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfThinking: &p}
}

func (variant BetaRedactedThinkingBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfRedactedThinking: &p}
}

func (variant BetaWebSearchToolResultBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfWebSearchToolResult: &p}
}

func (variant BetaBashCodeExecutionToolResultBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfBashCodeExecutionToolResult: &p}
}

func (variant BetaCodeExecutionToolResultBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfCodeExecutionToolResult: &p}
}

func (variant BetaContainerUploadBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfContainerUpload: &p}
}

func (variant BetaMCPToolResultBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfMCPToolResult: &p}
}

func (variant BetaMCPToolUseBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfMCPToolUse: &p}
}

func (variant BetaServerToolUseBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfServerToolUse: &p}
}

func (variant BetaTextEditorCodeExecutionToolResultBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfTextEditorCodeExecutionToolResult: &p}
}

func (variant BetaWebFetchToolResultBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfWebFetchToolResult: &p}
}

func (variant BetaToolSearchToolResultBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfToolSearchToolResult: &p}
}

func (variant BetaCompactionBlock) toParamUnion() BetaContentBlockParamUnion {
	p := variant.ToParam()
	return BetaContentBlockParamUnion{OfCompaction: &p}
}

func (r BetaMessage) ToParam() BetaMessageParam {
	var p BetaMessageParam
	p.Role = BetaMessageParamRole(r.Role)
	p.Content = make([]BetaContentBlockParamUnion, len(r.Content))
	for i, c := range r.Content {
		contentParams := c.ToParam()
		p.Content[i] = contentParams
	}
	return p
}

func (r BetaRedactedThinkingBlock) ToParam() BetaRedactedThinkingBlockParam {
	var p BetaRedactedThinkingBlockParam
	p.Type = r.Type
	p.Data = r.Data
	return p
}

func (r BetaTextBlock) ToParam() BetaTextBlockParam {
	var p BetaTextBlockParam
	p.Type = r.Type
	p.Text = r.Text

	// Distinguish between a nil and zero length slice, since some compatible
	// APIs may not require citations.
	if r.Citations != nil {
		p.Citations = make([]BetaTextCitationParamUnion, 0, len(r.Citations))
	}
	for _, citation := range r.Citations {
		p.Citations = append(p.Citations, citation.AsAny().toParamUnion())
	}
	return p
}

func (r BetaCitationCharLocation) toParamUnion() BetaTextCitationParamUnion {
	var citationParam BetaCitationCharLocationParam
	citationParam.Type = r.Type
	citationParam.DocumentTitle = paramutil.ToOpt(r.DocumentTitle, r.JSON.DocumentTitle)
	citationParam.CitedText = r.CitedText
	citationParam.DocumentIndex = r.DocumentIndex
	citationParam.EndCharIndex = r.EndCharIndex
	citationParam.StartCharIndex = r.StartCharIndex
	return BetaTextCitationParamUnion{OfCharLocation: &citationParam}
}

func (citationVariant BetaCitationPageLocation) toParamUnion() BetaTextCitationParamUnion {
	var citationParam BetaCitationPageLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.DocumentTitle = paramutil.ToOpt(citationVariant.DocumentTitle, citationVariant.JSON.DocumentTitle)
	citationParam.DocumentIndex = citationVariant.DocumentIndex
	citationParam.EndPageNumber = citationVariant.EndPageNumber
	citationParam.StartPageNumber = citationVariant.StartPageNumber
	return BetaTextCitationParamUnion{OfPageLocation: &citationParam}
}

func (citationVariant BetaCitationContentBlockLocation) toParamUnion() BetaTextCitationParamUnion {
	var citationParam BetaCitationContentBlockLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.DocumentTitle = paramutil.ToOpt(citationVariant.DocumentTitle, citationVariant.JSON.DocumentTitle)
	citationParam.CitedText = citationVariant.CitedText
	citationParam.DocumentIndex = citationVariant.DocumentIndex
	citationParam.EndBlockIndex = citationVariant.EndBlockIndex
	citationParam.StartBlockIndex = citationVariant.StartBlockIndex
	return BetaTextCitationParamUnion{OfContentBlockLocation: &citationParam}
}

func (citationVariant BetaCitationsWebSearchResultLocation) toParamUnion() BetaTextCitationParamUnion {
	var citationParam BetaCitationWebSearchResultLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.CitedText = citationVariant.CitedText
	citationParam.Title = paramutil.ToOpt(citationVariant.Title, citationVariant.JSON.Title)
	return BetaTextCitationParamUnion{OfWebSearchResultLocation: &citationParam}
}

func (citationVariant BetaCitationSearchResultLocation) toParamUnion() BetaTextCitationParamUnion {
	var citationParam BetaCitationSearchResultLocationParam
	citationParam.Type = citationVariant.Type
	citationParam.CitedText = citationVariant.CitedText
	citationParam.Title = paramutil.ToOpt(citationVariant.Title, citationVariant.JSON.Title)
	citationParam.EndBlockIndex = citationVariant.EndBlockIndex
	citationParam.StartBlockIndex = citationVariant.StartBlockIndex
	citationParam.Source = citationVariant.Source
	return BetaTextCitationParamUnion{OfSearchResultLocation: &citationParam}
}

func (r BetaThinkingBlock) ToParam() BetaThinkingBlockParam {
	var p BetaThinkingBlockParam
	p.Type = r.Type
	p.Signature = r.Signature
	p.Thinking = r.Thinking
	return p
}

func (r BetaToolUseBlock) ToParam() BetaToolUseBlockParam {
	var p BetaToolUseBlockParam
	p.Type = r.Type
	p.ID = r.ID
	p.Input = r.Input
	p.Name = r.Name
	return p
}

func (r BetaWebSearchResultBlock) ToParam() BetaWebSearchResultBlockParam {
	var p BetaWebSearchResultBlockParam
	p.Type = r.Type
	p.EncryptedContent = r.EncryptedContent
	p.Title = r.Title
	p.URL = r.URL
	p.PageAge = paramutil.ToOpt(r.PageAge, r.JSON.PageAge)
	return p
}

func (r BetaWebSearchToolResultBlock) ToParam() BetaWebSearchToolResultBlockParam {
	var p BetaWebSearchToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID

	if len(r.Content.OfBetaWebSearchResultBlockArray) > 0 {
		for _, block := range r.Content.OfBetaWebSearchResultBlockArray {
			p.Content.OfResultBlock = append(p.Content.OfResultBlock, block.ToParam())
		}
	} else {
		p.Content.OfError = &BetaWebSearchToolRequestErrorParam{
			Type:      r.Content.Type,
			ErrorCode: r.Content.ErrorCode,
		}
	}
	return p
}

func (r BetaWebFetchToolResultBlock) ToParam() BetaWebFetchToolResultBlockParam {
	var p BetaWebFetchToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	return p
}

func (r BetaMCPToolUseBlock) ToParam() BetaMCPToolUseBlockParam {
	var p BetaMCPToolUseBlockParam
	p.Type = r.Type
	p.ID = r.ID
	p.Input = r.Input
	p.Name = r.Name
	p.ServerName = r.ServerName
	return p
}

func (r BetaContainerUploadBlock) ToParam() BetaContainerUploadBlockParam {
	var p BetaContainerUploadBlockParam
	p.Type = r.Type
	p.FileID = r.FileID
	return p
}

func (r BetaServerToolUseBlock) ToParam() BetaServerToolUseBlockParam {
	var p BetaServerToolUseBlockParam
	p.Type = r.Type
	p.ID = r.ID
	p.Input = r.Input
	p.Name = BetaServerToolUseBlockParamName(r.Name)
	return p
}

func (r BetaTextEditorCodeExecutionToolResultBlock) ToParam() BetaTextEditorCodeExecutionToolResultBlockParam {
	var p BetaTextEditorCodeExecutionToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	if r.Content.JSON.ErrorCode.Valid() {
		p.Content.OfRequestTextEditorCodeExecutionToolResultError = &BetaTextEditorCodeExecutionToolResultErrorParam{
			ErrorCode:    BetaTextEditorCodeExecutionToolResultErrorParamErrorCode(r.Content.ErrorCode),
			ErrorMessage: paramutil.ToOpt(r.Content.ErrorMessage, r.Content.JSON.ErrorMessage),
		}
	} else {
		p.Content = param.Override[BetaTextEditorCodeExecutionToolResultBlockParamContentUnion](r.Content.RawJSON())
	}
	return p
}

func (r BetaMCPToolResultBlock) ToParam() BetaRequestMCPToolResultBlockParam {
	var p BetaRequestMCPToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	if r.Content.JSON.OfString.Valid() {
		p.Content.OfString = paramutil.ToOpt(r.Content.OfString, r.Content.JSON.OfString)
	} else {
		for _, block := range r.Content.OfBetaMCPToolResultBlockContent {
			p.Content.OfBetaMCPToolResultBlockContent = append(p.Content.OfBetaMCPToolResultBlockContent, block.ToParam())
		}
	}
	return p
}

func (r BetaBashCodeExecutionToolResultBlock) ToParam() BetaBashCodeExecutionToolResultBlockParam {
	var p BetaBashCodeExecutionToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID

	if r.Content.JSON.ErrorCode.Valid() {
		p.Content.OfRequestBashCodeExecutionToolResultError = &BetaBashCodeExecutionToolResultErrorParam{
			ErrorCode: BetaBashCodeExecutionToolResultErrorParamErrorCode(r.Content.ErrorCode),
		}
	} else {
		requestBashContentResult := &BetaBashCodeExecutionResultBlockParam{
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

func (r BetaBashCodeExecutionOutputBlock) ToParam() BetaBashCodeExecutionOutputBlockParam {
	var p BetaBashCodeExecutionOutputBlockParam
	p.Type = r.Type
	p.FileID = r.FileID
	return p
}

func (r BetaCodeExecutionToolResultBlock) ToParam() BetaCodeExecutionToolResultBlockParam {
	var p BetaCodeExecutionToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	if r.Content.JSON.ErrorCode.Valid() {
		p.Content.OfError = &BetaCodeExecutionToolResultErrorParam{
			ErrorCode: r.Content.ErrorCode,
		}
	} else {
		p.Content.OfResultBlock = &BetaCodeExecutionResultBlockParam{
			ReturnCode: r.Content.ReturnCode,
			Stderr:     r.Content.Stderr,
			Stdout:     r.Content.Stdout,
		}
		for _, block := range r.Content.Content {
			p.Content.OfResultBlock.Content = append(p.Content.OfResultBlock.Content, block.ToParam())
		}
	}
	return p
}

func (r BetaCodeExecutionOutputBlock) ToParam() BetaCodeExecutionOutputBlockParam {
	var p BetaCodeExecutionOutputBlockParam
	p.Type = r.Type
	p.FileID = r.FileID
	return p
}

func (r BetaToolSearchToolResultBlock) ToParam() BetaToolSearchToolResultBlockParam {
	var p BetaToolSearchToolResultBlockParam
	p.Type = r.Type
	p.ToolUseID = r.ToolUseID
	if r.Content.JSON.ErrorCode.Valid() {
		p.Content.OfRequestToolSearchToolResultError = &BetaToolSearchToolResultErrorParam{
			ErrorCode: BetaToolSearchToolResultErrorParamErrorCode(r.Content.ErrorCode),
		}
	} else {
		p.Content.OfRequestToolSearchToolSearchResultBlock = &BetaToolSearchToolSearchResultBlockParam{}
		for _, block := range r.Content.ToolReferences {
			p.Content.OfRequestToolSearchToolSearchResultBlock.ToolReferences = append(
				p.Content.OfRequestToolSearchToolSearchResultBlock.ToolReferences,
				block.ToParam(),
			)
		}
	}
	return p
}

func (r BetaToolReferenceBlock) ToParam() BetaToolReferenceBlockParam {
	var p BetaToolReferenceBlockParam
	p.Type = r.Type
	p.ToolName = r.ToolName
	return p
}

func (r BetaCompactionBlock) ToParam() BetaCompactionBlockParam {
	var p BetaCompactionBlockParam
	p.Type = r.Type
	p.Content = param.NewOpt(r.Content)
	return p
}
