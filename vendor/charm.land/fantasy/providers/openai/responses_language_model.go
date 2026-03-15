package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/object"
	"charm.land/fantasy/schema"
	"github.com/google/uuid"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared"
)

const topLogprobsMax = 20

type responsesLanguageModel struct {
	provider   string
	modelID    string
	client     openai.Client
	objectMode fantasy.ObjectMode
}

// newResponsesLanguageModel implements a responses api model
// INFO: (kujtim) currently we do not support stored parameter we default it to false.
func newResponsesLanguageModel(modelID string, provider string, client openai.Client, objectMode fantasy.ObjectMode) responsesLanguageModel {
	return responsesLanguageModel{
		modelID:    modelID,
		provider:   provider,
		client:     client,
		objectMode: objectMode,
	}
}

func (o responsesLanguageModel) Model() string {
	return o.modelID
}

func (o responsesLanguageModel) Provider() string {
	return o.provider
}

type responsesModelConfig struct {
	isReasoningModel           bool
	systemMessageMode          string
	requiredAutoTruncation     bool
	supportsFlexProcessing     bool
	supportsPriorityProcessing bool
}

func getResponsesModelConfig(modelID string) responsesModelConfig {
	supportsFlexProcessing := strings.HasPrefix(modelID, "o3") ||
		strings.Contains(modelID, "-o3") || strings.Contains(modelID, "o4-mini") ||
		(strings.Contains(modelID, "gpt-5") && !strings.Contains(modelID, "gpt-5-chat"))

	supportsPriorityProcessing := strings.Contains(modelID, "gpt-4") ||
		strings.Contains(modelID, "gpt-5-mini") ||
		(strings.Contains(modelID, "gpt-5") &&
			!strings.Contains(modelID, "gpt-5-nano") &&
			!strings.Contains(modelID, "gpt-5-chat")) ||
		strings.HasPrefix(modelID, "o3") ||
		strings.Contains(modelID, "-o3") ||
		strings.Contains(modelID, "o4-mini")

	defaults := responsesModelConfig{
		requiredAutoTruncation:     false,
		systemMessageMode:          "system",
		supportsFlexProcessing:     supportsFlexProcessing,
		supportsPriorityProcessing: supportsPriorityProcessing,
	}

	if strings.Contains(modelID, "gpt-5-chat") {
		return responsesModelConfig{
			isReasoningModel:           false,
			systemMessageMode:          defaults.systemMessageMode,
			requiredAutoTruncation:     defaults.requiredAutoTruncation,
			supportsFlexProcessing:     defaults.supportsFlexProcessing,
			supportsPriorityProcessing: defaults.supportsPriorityProcessing,
		}
	}

	if strings.HasPrefix(modelID, "o1") || strings.Contains(modelID, "-o1") ||
		strings.HasPrefix(modelID, "o3") || strings.Contains(modelID, "-o3") ||
		strings.HasPrefix(modelID, "o4") || strings.Contains(modelID, "-o4") ||
		strings.HasPrefix(modelID, "oss") || strings.Contains(modelID, "-oss") ||
		strings.Contains(modelID, "gpt-5") || strings.Contains(modelID, "codex-") ||
		strings.Contains(modelID, "computer-use") {
		if strings.Contains(modelID, "o1-mini") || strings.Contains(modelID, "o1-preview") {
			return responsesModelConfig{
				isReasoningModel:           true,
				systemMessageMode:          "remove",
				requiredAutoTruncation:     defaults.requiredAutoTruncation,
				supportsFlexProcessing:     defaults.supportsFlexProcessing,
				supportsPriorityProcessing: defaults.supportsPriorityProcessing,
			}
		}

		return responsesModelConfig{
			isReasoningModel:           true,
			systemMessageMode:          "developer",
			requiredAutoTruncation:     defaults.requiredAutoTruncation,
			supportsFlexProcessing:     defaults.supportsFlexProcessing,
			supportsPriorityProcessing: defaults.supportsPriorityProcessing,
		}
	}

	return responsesModelConfig{
		isReasoningModel:           false,
		systemMessageMode:          defaults.systemMessageMode,
		requiredAutoTruncation:     defaults.requiredAutoTruncation,
		supportsFlexProcessing:     defaults.supportsFlexProcessing,
		supportsPriorityProcessing: defaults.supportsPriorityProcessing,
	}
}

func (o responsesLanguageModel) prepareParams(call fantasy.Call) (*responses.ResponseNewParams, []fantasy.CallWarning) {
	var warnings []fantasy.CallWarning
	params := &responses.ResponseNewParams{
		Store: param.NewOpt(false),
	}

	modelConfig := getResponsesModelConfig(o.modelID)

	if call.TopK != nil {
		warnings = append(warnings, fantasy.CallWarning{
			Type:    fantasy.CallWarningTypeUnsupportedSetting,
			Setting: "topK",
		})
	}

	if call.PresencePenalty != nil {
		warnings = append(warnings, fantasy.CallWarning{
			Type:    fantasy.CallWarningTypeUnsupportedSetting,
			Setting: "presencePenalty",
		})
	}

	if call.FrequencyPenalty != nil {
		warnings = append(warnings, fantasy.CallWarning{
			Type:    fantasy.CallWarningTypeUnsupportedSetting,
			Setting: "frequencyPenalty",
		})
	}

	var openaiOptions *ResponsesProviderOptions
	if opts, ok := call.ProviderOptions[Name]; ok {
		if typedOpts, ok := opts.(*ResponsesProviderOptions); ok {
			openaiOptions = typedOpts
		}
	}

	input, inputWarnings := toResponsesPrompt(call.Prompt, modelConfig.systemMessageMode)
	warnings = append(warnings, inputWarnings...)

	var include []IncludeType

	addInclude := func(key IncludeType) {
		include = append(include, key)
	}

	topLogprobs := 0
	if openaiOptions != nil && openaiOptions.Logprobs != nil {
		switch v := openaiOptions.Logprobs.(type) {
		case bool:
			if v {
				topLogprobs = topLogprobsMax
			}
		case float64:
			topLogprobs = int(v)
		case int:
			topLogprobs = v
		}
	}

	if topLogprobs > 0 {
		addInclude(IncludeMessageOutputTextLogprobs)
	}

	params.Model = o.modelID
	params.Input = responses.ResponseNewParamsInputUnion{
		OfInputItemList: input,
	}

	if call.Temperature != nil {
		params.Temperature = param.NewOpt(*call.Temperature)
	}
	if call.TopP != nil {
		params.TopP = param.NewOpt(*call.TopP)
	}
	if call.MaxOutputTokens != nil {
		params.MaxOutputTokens = param.NewOpt(*call.MaxOutputTokens)
	}

	if openaiOptions != nil {
		if openaiOptions.MaxToolCalls != nil {
			params.MaxToolCalls = param.NewOpt(*openaiOptions.MaxToolCalls)
		}
		if openaiOptions.Metadata != nil {
			metadata := make(shared.Metadata)
			for k, v := range openaiOptions.Metadata {
				if str, ok := v.(string); ok {
					metadata[k] = str
				}
			}
			params.Metadata = metadata
		}
		if openaiOptions.ParallelToolCalls != nil {
			params.ParallelToolCalls = param.NewOpt(*openaiOptions.ParallelToolCalls)
		}
		if openaiOptions.User != nil {
			params.User = param.NewOpt(*openaiOptions.User)
		}
		if openaiOptions.Instructions != nil {
			params.Instructions = param.NewOpt(*openaiOptions.Instructions)
		}
		if openaiOptions.ServiceTier != nil {
			params.ServiceTier = responses.ResponseNewParamsServiceTier(*openaiOptions.ServiceTier)
		}
		if openaiOptions.PromptCacheKey != nil {
			params.PromptCacheKey = param.NewOpt(*openaiOptions.PromptCacheKey)
		}
		if openaiOptions.SafetyIdentifier != nil {
			params.SafetyIdentifier = param.NewOpt(*openaiOptions.SafetyIdentifier)
		}
		if topLogprobs > 0 {
			params.TopLogprobs = param.NewOpt(int64(topLogprobs))
		}

		if len(openaiOptions.Include) > 0 {
			include = append(include, openaiOptions.Include...)
		}

		if modelConfig.isReasoningModel && (openaiOptions.ReasoningEffort != nil || openaiOptions.ReasoningSummary != nil) {
			reasoning := shared.ReasoningParam{}
			if openaiOptions.ReasoningEffort != nil {
				reasoning.Effort = shared.ReasoningEffort(*openaiOptions.ReasoningEffort)
			}
			if openaiOptions.ReasoningSummary != nil {
				reasoning.Summary = shared.ReasoningSummary(*openaiOptions.ReasoningSummary)
			}
			params.Reasoning = reasoning
		}
	}

	if modelConfig.requiredAutoTruncation {
		params.Truncation = responses.ResponseNewParamsTruncationAuto
	}

	if len(include) > 0 {
		includeParams := make([]responses.ResponseIncludable, len(include))
		for i, inc := range include {
			includeParams[i] = responses.ResponseIncludable(string(inc))
		}
		params.Include = includeParams
	}

	if modelConfig.isReasoningModel {
		if call.Temperature != nil {
			params.Temperature = param.Opt[float64]{}
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "temperature",
				Details: "temperature is not supported for reasoning models",
			})
		}

		if call.TopP != nil {
			params.TopP = param.Opt[float64]{}
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "topP",
				Details: "topP is not supported for reasoning models",
			})
		}
	} else {
		if openaiOptions != nil {
			if openaiOptions.ReasoningEffort != nil {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeUnsupportedSetting,
					Setting: "reasoningEffort",
					Details: "reasoningEffort is not supported for non-reasoning models",
				})
			}

			if openaiOptions.ReasoningSummary != nil {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeUnsupportedSetting,
					Setting: "reasoningSummary",
					Details: "reasoningSummary is not supported for non-reasoning models",
				})
			}
		}
	}

	if openaiOptions != nil && openaiOptions.ServiceTier != nil {
		if *openaiOptions.ServiceTier == ServiceTierFlex && !modelConfig.supportsFlexProcessing {
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "serviceTier",
				Details: "flex processing is only available for o3, o4-mini, and gpt-5 models",
			})
			params.ServiceTier = ""
		}

		if *openaiOptions.ServiceTier == ServiceTierPriority && !modelConfig.supportsPriorityProcessing {
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "serviceTier",
				Details: "priority processing is only available for supported models (gpt-4, gpt-5, gpt-5-mini, o3, o4-mini) and requires Enterprise access. gpt-5-nano is not supported",
			})
			params.ServiceTier = ""
		}
	}

	tools, toolChoice, toolWarnings := toResponsesTools(call.Tools, call.ToolChoice, openaiOptions)
	warnings = append(warnings, toolWarnings...)

	if len(tools) > 0 {
		params.Tools = tools
		params.ToolChoice = toolChoice
	}

	return params, warnings
}

func toResponsesPrompt(prompt fantasy.Prompt, systemMessageMode string) (responses.ResponseInputParam, []fantasy.CallWarning) {
	var input responses.ResponseInputParam
	var warnings []fantasy.CallWarning

	for _, msg := range prompt {
		switch msg.Role {
		case fantasy.MessageRoleSystem:
			var systemText string
			for _, c := range msg.Content {
				if c.GetType() != fantasy.ContentTypeText {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "system prompt can only have text content",
					})
					continue
				}
				textPart, ok := fantasy.AsContentType[fantasy.TextPart](c)
				if !ok {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "system prompt text part does not have the right type",
					})
					continue
				}
				if strings.TrimSpace(textPart.Text) != "" {
					systemText += textPart.Text
				}
			}

			if systemText == "" {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "system prompt has no text parts",
				})
				continue
			}

			switch systemMessageMode {
			case "system":
				input = append(input, responses.ResponseInputItemParamOfMessage(systemText, responses.EasyInputMessageRoleSystem))
			case "developer":
				input = append(input, responses.ResponseInputItemParamOfMessage(systemText, responses.EasyInputMessageRoleDeveloper))
			case "remove":
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "system messages are removed for this model",
				})
			}

		case fantasy.MessageRoleUser:
			var contentParts responses.ResponseInputMessageContentListParam
			for i, c := range msg.Content {
				switch c.GetType() {
				case fantasy.ContentTypeText:
					textPart, ok := fantasy.AsContentType[fantasy.TextPart](c)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "user message text part does not have the right type",
						})
						continue
					}
					contentParts = append(contentParts, responses.ResponseInputContentUnionParam{
						OfInputText: &responses.ResponseInputTextParam{
							Type: "input_text",
							Text: textPart.Text,
						},
					})

				case fantasy.ContentTypeFile:
					filePart, ok := fantasy.AsContentType[fantasy.FilePart](c)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "user message file part does not have the right type",
						})
						continue
					}

					if strings.HasPrefix(filePart.MediaType, "image/") {
						base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
						imageURL := fmt.Sprintf("data:%s;base64,%s", filePart.MediaType, base64Encoded)
						contentParts = append(contentParts, responses.ResponseInputContentUnionParam{
							OfInputImage: &responses.ResponseInputImageParam{
								Type:     "input_image",
								ImageURL: param.NewOpt(imageURL),
							},
						})
					} else if filePart.MediaType == "application/pdf" {
						base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
						fileData := fmt.Sprintf("data:application/pdf;base64,%s", base64Encoded)
						filename := filePart.Filename
						if filename == "" {
							filename = fmt.Sprintf("part-%d.pdf", i)
						}
						contentParts = append(contentParts, responses.ResponseInputContentUnionParam{
							OfInputFile: &responses.ResponseInputFileParam{
								Type:     "input_file",
								Filename: param.NewOpt(filename),
								FileData: param.NewOpt(fileData),
							},
						})
					} else {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: fmt.Sprintf("file part media type %s not supported", filePart.MediaType),
						})
					}
				}
			}

			if !hasVisibleResponsesUserContent(contentParts) {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "dropping empty user message (contains neither user-facing content nor tool results)",
				})
				continue
			}

			input = append(input, responses.ResponseInputItemParamOfMessage(contentParts, responses.EasyInputMessageRoleUser))

		case fantasy.MessageRoleAssistant:
			startIdx := len(input)
			for _, c := range msg.Content {
				switch c.GetType() {
				case fantasy.ContentTypeText:
					textPart, ok := fantasy.AsContentType[fantasy.TextPart](c)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "assistant message text part does not have the right type",
						})
						continue
					}
					input = append(input, responses.ResponseInputItemParamOfMessage(textPart.Text, responses.EasyInputMessageRoleAssistant))

				case fantasy.ContentTypeToolCall:
					toolCallPart, ok := fantasy.AsContentType[fantasy.ToolCallPart](c)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "assistant message tool call part does not have the right type",
						})
						continue
					}

					if toolCallPart.ProviderExecuted {
						continue
					}

					inputJSON, err := json.Marshal(toolCallPart.Input)
					if err != nil {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: fmt.Sprintf("failed to marshal tool call input: %v", err),
						})
						continue
					}

					input = append(input, responses.ResponseInputItemParamOfFunctionCall(string(inputJSON), toolCallPart.ToolCallID, toolCallPart.ToolName))
				case fantasy.ContentTypeReasoning:
					reasoningMetadata := GetReasoningMetadata(c.Options())
					if reasoningMetadata == nil || reasoningMetadata.ItemID == "" {
						continue
					}
					if len(reasoningMetadata.Summary) == 0 && reasoningMetadata.EncryptedContent == nil {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "assistant message reasoning part does is empty",
						})
						continue
					}
					// we want to always send an empty array
					summary := []responses.ResponseReasoningItemSummaryParam{}
					for _, s := range reasoningMetadata.Summary {
						summary = append(summary, responses.ResponseReasoningItemSummaryParam{
							Type: "summary_text",
							Text: s,
						})
					}
					reasoning := &responses.ResponseReasoningItemParam{
						ID:      reasoningMetadata.ItemID,
						Summary: summary,
					}
					if reasoningMetadata.EncryptedContent != nil {
						reasoning.EncryptedContent = param.NewOpt(*reasoningMetadata.EncryptedContent)
					}
					input = append(input, responses.ResponseInputItemUnionParam{
						OfReasoning: reasoning,
					})
				}
			}

			if !hasVisibleResponsesAssistantContent(input, startIdx) {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "dropping empty assistant message (contains neither user-facing content nor tool calls)",
				})
				// Remove any items that were added during this iteration
				input = input[:startIdx]
				continue
			}

		case fantasy.MessageRoleTool:
			for _, c := range msg.Content {
				if c.GetType() != fantasy.ContentTypeToolResult {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "tool message can only have tool result content",
					})
					continue
				}

				toolResultPart, ok := fantasy.AsContentType[fantasy.ToolResultPart](c)
				if !ok {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "tool message result part does not have the right type",
					})
					continue
				}

				var outputStr string
				switch toolResultPart.Output.GetType() {
				case fantasy.ToolResultContentTypeText:
					output, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentText](toolResultPart.Output)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "tool result output does not have the right type",
						})
						continue
					}
					outputStr = output.Text
				case fantasy.ToolResultContentTypeError:
					output, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](toolResultPart.Output)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "tool result output does not have the right type",
						})
						continue
					}
					outputStr = output.Error.Error()
				}

				input = append(input, responses.ResponseInputItemParamOfFunctionCallOutput(toolResultPart.ToolCallID, outputStr))
			}
		}
	}

	return input, warnings
}

func hasVisibleResponsesUserContent(content responses.ResponseInputMessageContentListParam) bool {
	return len(content) > 0
}

func hasVisibleResponsesAssistantContent(items []responses.ResponseInputItemUnionParam, startIdx int) bool {
	// Check if we added any assistant content parts from this message
	for i := startIdx; i < len(items); i++ {
		if items[i].OfMessage != nil || items[i].OfFunctionCall != nil {
			return true
		}
	}
	return false
}

func toResponsesTools(tools []fantasy.Tool, toolChoice *fantasy.ToolChoice, options *ResponsesProviderOptions) ([]responses.ToolUnionParam, responses.ResponseNewParamsToolChoiceUnion, []fantasy.CallWarning) {
	warnings := make([]fantasy.CallWarning, 0)
	var openaiTools []responses.ToolUnionParam

	if len(tools) == 0 {
		return nil, responses.ResponseNewParamsToolChoiceUnion{}, nil
	}

	strictJSONSchema := false
	if options != nil && options.StrictJSONSchema != nil {
		strictJSONSchema = *options.StrictJSONSchema
	}

	for _, tool := range tools {
		if tool.GetType() == fantasy.ToolTypeFunction {
			ft, ok := tool.(fantasy.FunctionTool)
			if !ok {
				continue
			}
			openaiTools = append(openaiTools, responses.ToolUnionParam{
				OfFunction: &responses.FunctionToolParam{
					Name:        ft.Name,
					Description: param.NewOpt(ft.Description),
					Parameters:  ft.InputSchema,
					Strict:      param.NewOpt(strictJSONSchema),
					Type:        "function",
				},
			})
			continue
		}

		warnings = append(warnings, fantasy.CallWarning{
			Type:    fantasy.CallWarningTypeUnsupportedTool,
			Tool:    tool,
			Message: "tool is not supported",
		})
	}

	if toolChoice == nil {
		return openaiTools, responses.ResponseNewParamsToolChoiceUnion{}, warnings
	}

	var openaiToolChoice responses.ResponseNewParamsToolChoiceUnion

	switch *toolChoice {
	case fantasy.ToolChoiceAuto:
		openaiToolChoice = responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: param.NewOpt(responses.ToolChoiceOptionsAuto),
		}
	case fantasy.ToolChoiceNone:
		openaiToolChoice = responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: param.NewOpt(responses.ToolChoiceOptionsNone),
		}
	case fantasy.ToolChoiceRequired:
		openaiToolChoice = responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: param.NewOpt(responses.ToolChoiceOptionsRequired),
		}
	default:
		openaiToolChoice = responses.ResponseNewParamsToolChoiceUnion{
			OfFunctionTool: &responses.ToolChoiceFunctionParam{
				Type: "function",
				Name: string(*toolChoice),
			},
		}
	}

	return openaiTools, openaiToolChoice, warnings
}

func (o responsesLanguageModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	params, warnings := o.prepareParams(call)
	response, err := o.client.Responses.New(ctx, *params)
	if err != nil {
		return nil, toProviderErr(err)
	}

	if response.Error.Message != "" {
		return nil, &fantasy.Error{
			Title:   "provider error",
			Message: fmt.Sprintf("%s (code: %s)", response.Error.Message, response.Error.Code),
		}
	}

	var content []fantasy.Content
	hasFunctionCall := false

	for _, outputItem := range response.Output {
		switch outputItem.Type {
		case "message":
			for _, contentPart := range outputItem.Content {
				if contentPart.Type == "output_text" {
					content = append(content, fantasy.TextContent{
						Text: contentPart.Text,
					})

					for _, annotation := range contentPart.Annotations {
						switch annotation.Type {
						case "url_citation":
							content = append(content, fantasy.SourceContent{
								SourceType: fantasy.SourceTypeURL,
								ID:         uuid.NewString(),
								URL:        annotation.URL,
								Title:      annotation.Title,
							})
						case "file_citation":
							title := "Document"
							if annotation.Filename != "" {
								title = annotation.Filename
							}
							filename := annotation.Filename
							if filename == "" {
								filename = annotation.FileID
							}
							content = append(content, fantasy.SourceContent{
								SourceType: fantasy.SourceTypeDocument,
								ID:         uuid.NewString(),
								MediaType:  "text/plain",
								Title:      title,
								Filename:   filename,
							})
						}
					}
				}
			}

		case "function_call":
			hasFunctionCall = true
			content = append(content, fantasy.ToolCallContent{
				ProviderExecuted: false,
				ToolCallID:       outputItem.CallID,
				ToolName:         outputItem.Name,
				Input:            outputItem.Arguments,
			})

		case "reasoning":
			metadata := &ResponsesReasoningMetadata{
				ItemID: outputItem.ID,
			}
			if outputItem.EncryptedContent != "" {
				metadata.EncryptedContent = &outputItem.EncryptedContent
			}

			if len(outputItem.Summary) == 0 && metadata.EncryptedContent == nil {
				continue
			}

			// When there are no summary parts, add an empty reasoning part
			summaries := outputItem.Summary
			if len(summaries) == 0 {
				summaries = []responses.ResponseReasoningItemSummary{{Type: "summary_text", Text: ""}}
			}

			for _, s := range summaries {
				metadata.Summary = append(metadata.Summary, s.Text)
			}

			content = append(content, fantasy.ReasoningContent{
				Text: strings.Join(metadata.Summary, "\n"),
				ProviderMetadata: fantasy.ProviderMetadata{
					Name: metadata,
				},
			})
		}
	}

	usage := fantasy.Usage{
		InputTokens:  response.Usage.InputTokens,
		OutputTokens: response.Usage.OutputTokens,
		TotalTokens:  response.Usage.InputTokens + response.Usage.OutputTokens,
	}

	if response.Usage.OutputTokensDetails.ReasoningTokens != 0 {
		usage.ReasoningTokens = response.Usage.OutputTokensDetails.ReasoningTokens
	}
	if response.Usage.InputTokensDetails.CachedTokens != 0 {
		usage.CacheReadTokens = response.Usage.InputTokensDetails.CachedTokens
	}

	finishReason := mapResponsesFinishReason(response.IncompleteDetails.Reason, hasFunctionCall)

	return &fantasy.Response{
		Content:          content,
		Usage:            usage,
		FinishReason:     finishReason,
		ProviderMetadata: fantasy.ProviderMetadata{},
		Warnings:         warnings,
	}, nil
}

func mapResponsesFinishReason(reason string, hasFunctionCall bool) fantasy.FinishReason {
	if hasFunctionCall {
		return fantasy.FinishReasonToolCalls
	}

	switch reason {
	case "":
		return fantasy.FinishReasonStop
	case "max_tokens", "max_output_tokens":
		return fantasy.FinishReasonLength
	case "content_filter":
		return fantasy.FinishReasonContentFilter
	default:
		return fantasy.FinishReasonOther
	}
}

func (o responsesLanguageModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	params, warnings := o.prepareParams(call)

	stream := o.client.Responses.NewStreaming(ctx, *params)

	finishReason := fantasy.FinishReasonUnknown
	var usage fantasy.Usage
	ongoingToolCalls := make(map[int64]*ongoingToolCall)
	hasFunctionCall := false
	activeReasoning := make(map[string]*reasoningState)

	return func(yield func(fantasy.StreamPart) bool) {
		if len(warnings) > 0 {
			if !yield(fantasy.StreamPart{
				Type:     fantasy.StreamPartTypeWarnings,
				Warnings: warnings,
			}) {
				return
			}
		}

		for stream.Next() {
			event := stream.Current()

			switch event.Type {
			case "response.created":
				_ = event.AsResponseCreated()

			case "response.output_item.added":
				added := event.AsResponseOutputItemAdded()
				switch added.Item.Type {
				case "function_call":
					ongoingToolCalls[added.OutputIndex] = &ongoingToolCall{
						toolName:   added.Item.Name,
						toolCallID: added.Item.CallID,
					}
					if !yield(fantasy.StreamPart{
						Type:         fantasy.StreamPartTypeToolInputStart,
						ID:           added.Item.CallID,
						ToolCallName: added.Item.Name,
					}) {
						return
					}

				case "message":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeTextStart,
						ID:   added.Item.ID,
					}) {
						return
					}

				case "reasoning":
					metadata := &ResponsesReasoningMetadata{
						ItemID:  added.Item.ID,
						Summary: []string{},
					}
					if added.Item.EncryptedContent != "" {
						metadata.EncryptedContent = &added.Item.EncryptedContent
					}

					activeReasoning[added.Item.ID] = &reasoningState{
						metadata: metadata,
					}
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeReasoningStart,
						ID:   added.Item.ID,
						ProviderMetadata: fantasy.ProviderMetadata{
							Name: metadata,
						},
					}) {
						return
					}
				}

			case "response.output_item.done":
				done := event.AsResponseOutputItemDone()
				switch done.Item.Type {
				case "function_call":
					tc := ongoingToolCalls[done.OutputIndex]
					if tc != nil {
						delete(ongoingToolCalls, done.OutputIndex)
						hasFunctionCall = true

						if !yield(fantasy.StreamPart{
							Type: fantasy.StreamPartTypeToolInputEnd,
							ID:   done.Item.CallID,
						}) {
							return
						}
						if !yield(fantasy.StreamPart{
							Type:          fantasy.StreamPartTypeToolCall,
							ID:            done.Item.CallID,
							ToolCallName:  done.Item.Name,
							ToolCallInput: done.Item.Arguments,
						}) {
							return
						}
					}

				case "message":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeTextEnd,
						ID:   done.Item.ID,
					}) {
						return
					}

				case "reasoning":
					state := activeReasoning[done.Item.ID]
					if state != nil {
						if !yield(fantasy.StreamPart{
							Type: fantasy.StreamPartTypeReasoningEnd,
							ID:   done.Item.ID,
							ProviderMetadata: fantasy.ProviderMetadata{
								Name: state.metadata,
							},
						}) {
							return
						}
						delete(activeReasoning, done.Item.ID)
					}
				}

			case "response.function_call_arguments.delta":
				delta := event.AsResponseFunctionCallArgumentsDelta()
				tc := ongoingToolCalls[delta.OutputIndex]
				if tc != nil {
					if !yield(fantasy.StreamPart{
						Type:  fantasy.StreamPartTypeToolInputDelta,
						ID:    tc.toolCallID,
						Delta: delta.Delta,
					}) {
						return
					}
				}

			case "response.output_text.delta":
				textDelta := event.AsResponseOutputTextDelta()
				if !yield(fantasy.StreamPart{
					Type:  fantasy.StreamPartTypeTextDelta,
					ID:    textDelta.ItemID,
					Delta: textDelta.Delta,
				}) {
					return
				}

			case "response.reasoning_summary_part.added":
				added := event.AsResponseReasoningSummaryPartAdded()
				state := activeReasoning[added.ItemID]
				if state != nil {
					state.metadata.Summary = append(state.metadata.Summary, "")
					activeReasoning[added.ItemID] = state
					if !yield(fantasy.StreamPart{
						Type:  fantasy.StreamPartTypeReasoningDelta,
						ID:    added.ItemID,
						Delta: "\n",
						ProviderMetadata: fantasy.ProviderMetadata{
							Name: state.metadata,
						},
					}) {
						return
					}
				}

			case "response.reasoning_summary_text.delta":
				textDelta := event.AsResponseReasoningSummaryTextDelta()
				state := activeReasoning[textDelta.ItemID]
				if state != nil {
					if len(state.metadata.Summary)-1 >= int(textDelta.SummaryIndex) {
						state.metadata.Summary[textDelta.SummaryIndex] += textDelta.Delta
					}
					activeReasoning[textDelta.ItemID] = state
					if !yield(fantasy.StreamPart{
						Type:  fantasy.StreamPartTypeReasoningDelta,
						ID:    textDelta.ItemID,
						Delta: textDelta.Delta,
						ProviderMetadata: fantasy.ProviderMetadata{
							Name: state.metadata,
						},
					}) {
						return
					}
				}

			case "response.completed", "response.incomplete":
				completed := event.AsResponseCompleted()
				finishReason = mapResponsesFinishReason(completed.Response.IncompleteDetails.Reason, hasFunctionCall)
				usage = fantasy.Usage{
					InputTokens:  completed.Response.Usage.InputTokens,
					OutputTokens: completed.Response.Usage.OutputTokens,
					TotalTokens:  completed.Response.Usage.InputTokens + completed.Response.Usage.OutputTokens,
				}
				if completed.Response.Usage.OutputTokensDetails.ReasoningTokens != 0 {
					usage.ReasoningTokens = completed.Response.Usage.OutputTokensDetails.ReasoningTokens
				}
				if completed.Response.Usage.InputTokensDetails.CachedTokens != 0 {
					usage.CacheReadTokens = completed.Response.Usage.InputTokensDetails.CachedTokens
				}

			case "error":
				errorEvent := event.AsError()
				if !yield(fantasy.StreamPart{
					Type:  fantasy.StreamPartTypeError,
					Error: fmt.Errorf("response error: %s (code: %s)", errorEvent.Message, errorEvent.Code),
				}) {
					return
				}
				return
			}
		}

		err := stream.Err()
		if err != nil {
			yield(fantasy.StreamPart{
				Type:  fantasy.StreamPartTypeError,
				Error: toProviderErr(err),
			})
			return
		}

		yield(fantasy.StreamPart{
			Type:         fantasy.StreamPartTypeFinish,
			Usage:        usage,
			FinishReason: finishReason,
		})
	}, nil
}

// GetReasoningMetadata extracts reasoning metadata from provider options for responses models.
func GetReasoningMetadata(providerOptions fantasy.ProviderOptions) *ResponsesReasoningMetadata {
	if openaiResponsesOptions, ok := providerOptions[Name]; ok {
		if reasoning, ok := openaiResponsesOptions.(*ResponsesReasoningMetadata); ok {
			return reasoning
		}
	}
	return nil
}

type ongoingToolCall struct {
	toolName   string
	toolCallID string
}

type reasoningState struct {
	metadata *ResponsesReasoningMetadata
}

// GenerateObject implements fantasy.LanguageModel.
func (o responsesLanguageModel) GenerateObject(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	switch o.objectMode {
	case fantasy.ObjectModeText:
		return object.GenerateWithText(ctx, o, call)
	case fantasy.ObjectModeTool:
		return object.GenerateWithTool(ctx, o, call)
	default:
		return o.generateObjectWithJSONMode(ctx, call)
	}
}

// StreamObject implements fantasy.LanguageModel.
func (o responsesLanguageModel) StreamObject(ctx context.Context, call fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	switch o.objectMode {
	case fantasy.ObjectModeTool:
		return object.StreamWithTool(ctx, o, call)
	case fantasy.ObjectModeText:
		return object.StreamWithText(ctx, o, call)
	default:
		return o.streamObjectWithJSONMode(ctx, call)
	}
}

func (o responsesLanguageModel) generateObjectWithJSONMode(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	// Convert our Schema to OpenAI's JSON Schema format
	jsonSchemaMap := schema.ToMap(call.Schema)

	// Add additionalProperties: false recursively for strict mode (OpenAI requirement)
	addAdditionalPropertiesFalse(jsonSchemaMap)

	schemaName := call.SchemaName
	if schemaName == "" {
		schemaName = "response"
	}

	// Build request using prepareParams
	fantasyCall := fantasy.Call{
		Prompt:           call.Prompt,
		MaxOutputTokens:  call.MaxOutputTokens,
		Temperature:      call.Temperature,
		TopP:             call.TopP,
		PresencePenalty:  call.PresencePenalty,
		FrequencyPenalty: call.FrequencyPenalty,
		ProviderOptions:  call.ProviderOptions,
	}

	params, warnings := o.prepareParams(fantasyCall)

	// Add structured output via Text.Format field
	params.Text = responses.ResponseTextConfigParam{
		Format: responses.ResponseFormatTextConfigParamOfJSONSchema(schemaName, jsonSchemaMap),
	}

	// Make request
	response, err := o.client.Responses.New(ctx, *params)
	if err != nil {
		return nil, toProviderErr(err)
	}

	if response.Error.Message != "" {
		return nil, &fantasy.Error{
			Title:   "provider error",
			Message: fmt.Sprintf("%s (code: %s)", response.Error.Message, response.Error.Code),
		}
	}

	// Extract JSON text from response
	var jsonText string
	for _, outputItem := range response.Output {
		if outputItem.Type == "message" {
			for _, contentPart := range outputItem.Content {
				if contentPart.Type == "output_text" {
					jsonText = contentPart.Text
					break
				}
			}
		}
	}

	if jsonText == "" {
		usage := fantasy.Usage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
			TotalTokens:  response.Usage.InputTokens + response.Usage.OutputTokens,
		}
		finishReason := mapResponsesFinishReason(response.IncompleteDetails.Reason, false)
		return nil, &fantasy.NoObjectGeneratedError{
			RawText:      "",
			ParseError:   fmt.Errorf("no text content in response"),
			Usage:        usage,
			FinishReason: finishReason,
		}
	}

	// Parse and validate
	var obj any
	if call.RepairText != nil {
		obj, err = schema.ParseAndValidateWithRepair(ctx, jsonText, call.Schema, call.RepairText)
	} else {
		obj, err = schema.ParseAndValidate(jsonText, call.Schema)
	}

	usage := fantasy.Usage{
		InputTokens:  response.Usage.InputTokens,
		OutputTokens: response.Usage.OutputTokens,
		TotalTokens:  response.Usage.InputTokens + response.Usage.OutputTokens,
	}
	if response.Usage.OutputTokensDetails.ReasoningTokens != 0 {
		usage.ReasoningTokens = response.Usage.OutputTokensDetails.ReasoningTokens
	}
	if response.Usage.InputTokensDetails.CachedTokens != 0 {
		usage.CacheReadTokens = response.Usage.InputTokensDetails.CachedTokens
	}

	finishReason := mapResponsesFinishReason(response.IncompleteDetails.Reason, false)

	if err != nil {
		// Add usage info to error
		if nogErr, ok := err.(*fantasy.NoObjectGeneratedError); ok {
			nogErr.Usage = usage
			nogErr.FinishReason = finishReason
		}
		return nil, err
	}

	return &fantasy.ObjectResponse{
		Object:       obj,
		RawText:      jsonText,
		Usage:        usage,
		FinishReason: finishReason,
		Warnings:     warnings,
	}, nil
}

func (o responsesLanguageModel) streamObjectWithJSONMode(ctx context.Context, call fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	// Convert our Schema to OpenAI's JSON Schema format
	jsonSchemaMap := schema.ToMap(call.Schema)

	// Add additionalProperties: false recursively for strict mode (OpenAI requirement)
	addAdditionalPropertiesFalse(jsonSchemaMap)

	schemaName := call.SchemaName
	if schemaName == "" {
		schemaName = "response"
	}

	// Build request using prepareParams
	fantasyCall := fantasy.Call{
		Prompt:           call.Prompt,
		MaxOutputTokens:  call.MaxOutputTokens,
		Temperature:      call.Temperature,
		TopP:             call.TopP,
		PresencePenalty:  call.PresencePenalty,
		FrequencyPenalty: call.FrequencyPenalty,
		ProviderOptions:  call.ProviderOptions,
	}

	params, warnings := o.prepareParams(fantasyCall)

	// Add structured output via Text.Format field
	params.Text = responses.ResponseTextConfigParam{
		Format: responses.ResponseFormatTextConfigParamOfJSONSchema(schemaName, jsonSchemaMap),
	}

	stream := o.client.Responses.NewStreaming(ctx, *params)

	return func(yield func(fantasy.ObjectStreamPart) bool) {
		if len(warnings) > 0 {
			if !yield(fantasy.ObjectStreamPart{
				Type:     fantasy.ObjectStreamPartTypeObject,
				Warnings: warnings,
			}) {
				return
			}
		}

		var accumulated string
		var lastParsedObject any
		var usage fantasy.Usage
		var finishReason fantasy.FinishReason
		var streamErr error
		hasFunctionCall := false

		for stream.Next() {
			event := stream.Current()

			switch event.Type {
			case "response.output_text.delta":
				textDelta := event.AsResponseOutputTextDelta()
				accumulated += textDelta.Delta

				// Try to parse the accumulated text
				obj, state, parseErr := schema.ParsePartialJSON(accumulated)

				// If we successfully parsed, validate and emit
				if state == schema.ParseStateSuccessful || state == schema.ParseStateRepaired {
					if err := schema.ValidateAgainstSchema(obj, call.Schema); err == nil {
						// Only emit if object is different from last
						if !reflect.DeepEqual(obj, lastParsedObject) {
							if !yield(fantasy.ObjectStreamPart{
								Type:   fantasy.ObjectStreamPartTypeObject,
								Object: obj,
							}) {
								return
							}
							lastParsedObject = obj
						}
					}
				}

				// If parsing failed and we have a repair function, try it
				if state == schema.ParseStateFailed && call.RepairText != nil {
					repairedText, repairErr := call.RepairText(ctx, accumulated, parseErr)
					if repairErr == nil {
						obj2, state2, _ := schema.ParsePartialJSON(repairedText)
						if (state2 == schema.ParseStateSuccessful || state2 == schema.ParseStateRepaired) &&
							schema.ValidateAgainstSchema(obj2, call.Schema) == nil {
							if !reflect.DeepEqual(obj2, lastParsedObject) {
								if !yield(fantasy.ObjectStreamPart{
									Type:   fantasy.ObjectStreamPartTypeObject,
									Object: obj2,
								}) {
									return
								}
								lastParsedObject = obj2
							}
						}
					}
				}

			case "response.completed", "response.incomplete":
				completed := event.AsResponseCompleted()
				finishReason = mapResponsesFinishReason(completed.Response.IncompleteDetails.Reason, hasFunctionCall)
				usage = fantasy.Usage{
					InputTokens:  completed.Response.Usage.InputTokens,
					OutputTokens: completed.Response.Usage.OutputTokens,
					TotalTokens:  completed.Response.Usage.InputTokens + completed.Response.Usage.OutputTokens,
				}
				if completed.Response.Usage.OutputTokensDetails.ReasoningTokens != 0 {
					usage.ReasoningTokens = completed.Response.Usage.OutputTokensDetails.ReasoningTokens
				}
				if completed.Response.Usage.InputTokensDetails.CachedTokens != 0 {
					usage.CacheReadTokens = completed.Response.Usage.InputTokensDetails.CachedTokens
				}

			case "error":
				errorEvent := event.AsError()
				streamErr = fmt.Errorf("response error: %s (code: %s)", errorEvent.Message, errorEvent.Code)
				if !yield(fantasy.ObjectStreamPart{
					Type:  fantasy.ObjectStreamPartTypeError,
					Error: streamErr,
				}) {
					return
				}
				return
			}
		}

		err := stream.Err()
		if err != nil {
			yield(fantasy.ObjectStreamPart{
				Type:  fantasy.ObjectStreamPartTypeError,
				Error: toProviderErr(err),
			})
			return
		}

		// Final validation and emit
		if streamErr == nil && lastParsedObject != nil {
			yield(fantasy.ObjectStreamPart{
				Type:         fantasy.ObjectStreamPartTypeFinish,
				Usage:        usage,
				FinishReason: finishReason,
			})
		} else if streamErr == nil && lastParsedObject == nil {
			// No object was generated
			yield(fantasy.ObjectStreamPart{
				Type: fantasy.ObjectStreamPartTypeError,
				Error: &fantasy.NoObjectGeneratedError{
					RawText:      accumulated,
					ParseError:   fmt.Errorf("no valid object generated in stream"),
					Usage:        usage,
					FinishReason: finishReason,
				},
			})
		}
	}, nil
}
