package openai

import (
	"encoding/base64"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/shared"
)

// LanguageModelPrepareCallFunc is a function that prepares the call for the language model.
type LanguageModelPrepareCallFunc = func(model fantasy.LanguageModel, params *openai.ChatCompletionNewParams, call fantasy.Call) ([]fantasy.CallWarning, error)

// LanguageModelMapFinishReasonFunc is a function that maps the finish reason for the language model.
type LanguageModelMapFinishReasonFunc = func(finishReason string) fantasy.FinishReason

// LanguageModelUsageFunc is a function that calculates usage for the language model.
type LanguageModelUsageFunc = func(choice openai.ChatCompletion) (fantasy.Usage, fantasy.ProviderOptionsData)

// LanguageModelExtraContentFunc is a function that adds extra content for the language model.
type LanguageModelExtraContentFunc = func(choice openai.ChatCompletionChoice) []fantasy.Content

// LanguageModelStreamExtraFunc is a function that handles stream extra functionality for the language model.
type LanguageModelStreamExtraFunc = func(chunk openai.ChatCompletionChunk, yield func(fantasy.StreamPart) bool, ctx map[string]any) (map[string]any, bool)

// LanguageModelStreamUsageFunc is a function that calculates stream usage for the language model.
type LanguageModelStreamUsageFunc = func(chunk openai.ChatCompletionChunk, ctx map[string]any, metadata fantasy.ProviderMetadata) (fantasy.Usage, fantasy.ProviderMetadata)

// LanguageModelStreamProviderMetadataFunc is a function that handles stream provider metadata for the language model.
type LanguageModelStreamProviderMetadataFunc = func(choice openai.ChatCompletionChoice, metadata fantasy.ProviderMetadata) fantasy.ProviderMetadata

// LanguageModelToPromptFunc is a function that handles converting fantasy prompts to openai sdk messages.
type LanguageModelToPromptFunc = func(prompt fantasy.Prompt, provider, model string) ([]openai.ChatCompletionMessageParamUnion, []fantasy.CallWarning)

// DefaultPrepareCallFunc is the default implementation for preparing a call to the language model.
func DefaultPrepareCallFunc(model fantasy.LanguageModel, params *openai.ChatCompletionNewParams, call fantasy.Call) ([]fantasy.CallWarning, error) {
	if call.ProviderOptions == nil {
		return nil, nil
	}
	var warnings []fantasy.CallWarning
	providerOptions := &ProviderOptions{}
	if v, ok := call.ProviderOptions[Name]; ok {
		providerOptions, ok = v.(*ProviderOptions)
		if !ok {
			return nil, &fantasy.Error{Title: "invalid argument", Message: "openai provider options should be *openai.ProviderOptions"}
		}
	}

	if providerOptions.LogitBias != nil {
		params.LogitBias = providerOptions.LogitBias
	}
	if providerOptions.LogProbs != nil && providerOptions.TopLogProbs != nil {
		providerOptions.LogProbs = nil
	}
	if providerOptions.LogProbs != nil {
		params.Logprobs = param.NewOpt(*providerOptions.LogProbs)
	}
	if providerOptions.TopLogProbs != nil {
		params.TopLogprobs = param.NewOpt(*providerOptions.TopLogProbs)
	}
	if providerOptions.User != nil {
		params.User = param.NewOpt(*providerOptions.User)
	}
	if providerOptions.ParallelToolCalls != nil {
		params.ParallelToolCalls = param.NewOpt(*providerOptions.ParallelToolCalls)
	}
	if providerOptions.MaxCompletionTokens != nil {
		params.MaxCompletionTokens = param.NewOpt(*providerOptions.MaxCompletionTokens)
	}

	if providerOptions.TextVerbosity != nil {
		params.Verbosity = openai.ChatCompletionNewParamsVerbosity(*providerOptions.TextVerbosity)
	}
	if providerOptions.Prediction != nil {
		// Convert map[string]any to ChatCompletionPredictionContentParam
		if content, ok := providerOptions.Prediction["content"]; ok {
			if contentStr, ok := content.(string); ok {
				params.Prediction = openai.ChatCompletionPredictionContentParam{
					Content: openai.ChatCompletionPredictionContentContentUnionParam{
						OfString: param.NewOpt(contentStr),
					},
				}
			}
		}
	}
	if providerOptions.Store != nil {
		params.Store = param.NewOpt(*providerOptions.Store)
	}
	if providerOptions.Metadata != nil {
		// Convert map[string]any to map[string]string
		metadata := make(map[string]string)
		for k, v := range providerOptions.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			}
		}
		params.Metadata = metadata
	}
	if providerOptions.PromptCacheKey != nil {
		params.PromptCacheKey = param.NewOpt(*providerOptions.PromptCacheKey)
	}
	if providerOptions.SafetyIdentifier != nil {
		params.SafetyIdentifier = param.NewOpt(*providerOptions.SafetyIdentifier)
	}
	if providerOptions.ServiceTier != nil {
		params.ServiceTier = openai.ChatCompletionNewParamsServiceTier(*providerOptions.ServiceTier)
	}

	if providerOptions.ReasoningEffort != nil {
		switch *providerOptions.ReasoningEffort {
		case ReasoningEffortMinimal:
			params.ReasoningEffort = shared.ReasoningEffortMinimal
		case ReasoningEffortLow:
			params.ReasoningEffort = shared.ReasoningEffortLow
		case ReasoningEffortMedium:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case ReasoningEffortHigh:
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			return nil, fmt.Errorf("reasoning model `%s` not supported", *providerOptions.ReasoningEffort)
		}
	}

	if isReasoningModel(model.Model()) {
		if providerOptions.LogitBias != nil {
			params.LogitBias = nil
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "LogitBias",
				Message: "LogitBias is not supported for reasoning models",
			})
		}
		if providerOptions.LogProbs != nil {
			params.Logprobs = param.Opt[bool]{}
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "Logprobs",
				Message: "Logprobs is not supported for reasoning models",
			})
		}
		if providerOptions.TopLogProbs != nil {
			params.TopLogprobs = param.Opt[int64]{}
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "TopLogprobs",
				Message: "TopLogprobs is not supported for reasoning models",
			})
		}
	}

	// Handle service tier validation
	if providerOptions.ServiceTier != nil {
		serviceTier := *providerOptions.ServiceTier
		if serviceTier == "flex" && !supportsFlexProcessing(model.Model()) {
			params.ServiceTier = ""
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "ServiceTier",
				Details: "flex processing is only available for o3, o4-mini, and gpt-5 models",
			})
		} else if serviceTier == "priority" && !supportsPriorityProcessing(model.Model()) {
			params.ServiceTier = ""
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "ServiceTier",
				Details: "priority processing is only available for supported models (gpt-4, gpt-5, gpt-5-mini, o3, o4-mini) and requires Enterprise access. gpt-5-nano is not supported",
			})
		}
	}
	return warnings, nil
}

// DefaultMapFinishReasonFunc is the default implementation for mapping finish reasons.
func DefaultMapFinishReasonFunc(finishReason string) fantasy.FinishReason {
	switch finishReason {
	case "stop":
		return fantasy.FinishReasonStop
	case "length":
		return fantasy.FinishReasonLength
	case "content_filter":
		return fantasy.FinishReasonContentFilter
	case "function_call", "tool_calls":
		return fantasy.FinishReasonToolCalls
	default:
		return fantasy.FinishReasonUnknown
	}
}

// DefaultUsageFunc is the default implementation for calculating usage.
func DefaultUsageFunc(response openai.ChatCompletion) (fantasy.Usage, fantasy.ProviderOptionsData) {
	completionTokenDetails := response.Usage.CompletionTokensDetails
	promptTokenDetails := response.Usage.PromptTokensDetails

	// Build provider metadata
	providerMetadata := &ProviderMetadata{}

	// Add logprobs if available
	if len(response.Choices) > 0 && len(response.Choices[0].Logprobs.Content) > 0 {
		providerMetadata.Logprobs = response.Choices[0].Logprobs.Content
	}

	// Add prediction tokens if available
	if completionTokenDetails.AcceptedPredictionTokens > 0 || completionTokenDetails.RejectedPredictionTokens > 0 {
		if completionTokenDetails.AcceptedPredictionTokens > 0 {
			providerMetadata.AcceptedPredictionTokens = completionTokenDetails.AcceptedPredictionTokens
		}
		if completionTokenDetails.RejectedPredictionTokens > 0 {
			providerMetadata.RejectedPredictionTokens = completionTokenDetails.RejectedPredictionTokens
		}
	}
	return fantasy.Usage{
		InputTokens:     response.Usage.PromptTokens,
		OutputTokens:    response.Usage.CompletionTokens,
		TotalTokens:     response.Usage.TotalTokens,
		ReasoningTokens: completionTokenDetails.ReasoningTokens,
		CacheReadTokens: promptTokenDetails.CachedTokens,
	}, providerMetadata
}

// DefaultStreamUsageFunc is the default implementation for calculating stream usage.
func DefaultStreamUsageFunc(chunk openai.ChatCompletionChunk, _ map[string]any, metadata fantasy.ProviderMetadata) (fantasy.Usage, fantasy.ProviderMetadata) {
	if chunk.Usage.TotalTokens == 0 {
		return fantasy.Usage{}, nil
	}
	streamProviderMetadata := &ProviderMetadata{}
	if metadata != nil {
		if providerMetadata, ok := metadata[Name]; ok {
			converted, ok := providerMetadata.(*ProviderMetadata)
			if ok {
				streamProviderMetadata = converted
			}
		}
	}
	// we do this here because the acc does not add prompt details
	completionTokenDetails := chunk.Usage.CompletionTokensDetails
	promptTokenDetails := chunk.Usage.PromptTokensDetails
	usage := fantasy.Usage{
		InputTokens:     chunk.Usage.PromptTokens,
		OutputTokens:    chunk.Usage.CompletionTokens,
		TotalTokens:     chunk.Usage.TotalTokens,
		ReasoningTokens: completionTokenDetails.ReasoningTokens,
		CacheReadTokens: promptTokenDetails.CachedTokens,
	}

	// Add prediction tokens if available
	if completionTokenDetails.AcceptedPredictionTokens > 0 || completionTokenDetails.RejectedPredictionTokens > 0 {
		if completionTokenDetails.AcceptedPredictionTokens > 0 {
			streamProviderMetadata.AcceptedPredictionTokens = completionTokenDetails.AcceptedPredictionTokens
		}
		if completionTokenDetails.RejectedPredictionTokens > 0 {
			streamProviderMetadata.RejectedPredictionTokens = completionTokenDetails.RejectedPredictionTokens
		}
	}

	return usage, fantasy.ProviderMetadata{
		Name: streamProviderMetadata,
	}
}

// DefaultStreamProviderMetadataFunc is the default implementation for handling stream provider metadata.
func DefaultStreamProviderMetadataFunc(choice openai.ChatCompletionChoice, metadata fantasy.ProviderMetadata) fantasy.ProviderMetadata {
	if metadata == nil {
		metadata = fantasy.ProviderMetadata{}
	}
	streamProviderMetadata, ok := metadata[Name]
	if !ok {
		streamProviderMetadata = &ProviderMetadata{}
	}
	if converted, ok := streamProviderMetadata.(*ProviderMetadata); ok {
		converted.Logprobs = choice.Logprobs.Content
		metadata[Name] = converted
	}
	return metadata
}

// DefaultToPrompt converts a fantasy prompt to OpenAI format with default handling.
func DefaultToPrompt(prompt fantasy.Prompt, _, _ string) ([]openai.ChatCompletionMessageParamUnion, []fantasy.CallWarning) {
	var messages []openai.ChatCompletionMessageParamUnion
	var warnings []fantasy.CallWarning
	for _, msg := range prompt {
		switch msg.Role {
		case fantasy.MessageRoleSystem:
			var systemPromptParts []string
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
				text := textPart.Text
				if strings.TrimSpace(text) != "" {
					systemPromptParts = append(systemPromptParts, textPart.Text)
				}
			}
			if len(systemPromptParts) == 0 {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "system prompt has no text parts",
				})
				continue
			}
			messages = append(messages, openai.SystemMessage(strings.Join(systemPromptParts, "\n")))
		case fantasy.MessageRoleUser:
			// simple user message just text content
			if len(msg.Content) == 1 && msg.Content[0].GetType() == fantasy.ContentTypeText {
				textPart, ok := fantasy.AsContentType[fantasy.TextPart](msg.Content[0])
				if !ok {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "user message text part does not have the right type",
					})
					continue
				}
				messages = append(messages, openai.UserMessage(textPart.Text))
				continue
			}
			// text content and attachments
			// for now we only support image content later we need to check
			// TODO: add the supported media types to the language model so we
			//  can use that to validate the data here.
			var content []openai.ChatCompletionContentPartUnionParam
			for _, c := range msg.Content {
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
					content = append(content, openai.ChatCompletionContentPartUnionParam{
						OfText: &openai.ChatCompletionContentPartTextParam{
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

					switch {
					case strings.HasPrefix(filePart.MediaType, "image/"):
						// Handle image files
						base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
						data := "data:" + filePart.MediaType + ";base64," + base64Encoded
						imageURL := openai.ChatCompletionContentPartImageImageURLParam{URL: data}

						// Check for provider-specific options like image detail
						if providerOptions, ok := filePart.ProviderOptions[Name]; ok {
							if detail, ok := providerOptions.(*ProviderFileOptions); ok {
								imageURL.Detail = detail.ImageDetail
							}
						}

						imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}
						content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})

					case filePart.MediaType == "audio/wav":
						// Handle WAV audio files
						base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
						audioBlock := openai.ChatCompletionContentPartInputAudioParam{
							InputAudio: openai.ChatCompletionContentPartInputAudioInputAudioParam{
								Data:   base64Encoded,
								Format: "wav",
							},
						}
						content = append(content, openai.ChatCompletionContentPartUnionParam{OfInputAudio: &audioBlock})

					case filePart.MediaType == "audio/mpeg" || filePart.MediaType == "audio/mp3":
						// Handle MP3 audio files
						base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
						audioBlock := openai.ChatCompletionContentPartInputAudioParam{
							InputAudio: openai.ChatCompletionContentPartInputAudioInputAudioParam{
								Data:   base64Encoded,
								Format: "mp3",
							},
						}
						content = append(content, openai.ChatCompletionContentPartUnionParam{OfInputAudio: &audioBlock})

					case filePart.MediaType == "application/pdf":
						// Handle PDF files
						dataStr := string(filePart.Data)

						// Check if data looks like a file ID (starts with "file-")
						if strings.HasPrefix(dataStr, "file-") {
							fileBlock := openai.ChatCompletionContentPartFileParam{
								File: openai.ChatCompletionContentPartFileFileParam{
									FileID: param.NewOpt(dataStr),
								},
							}
							content = append(content, openai.ChatCompletionContentPartUnionParam{OfFile: &fileBlock})
						} else {
							// Handle as base64 data
							base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
							data := "data:application/pdf;base64," + base64Encoded

							filename := filePart.Filename
							if filename == "" {
								// Generate default filename based on content index
								filename = fmt.Sprintf("part-%d.pdf", len(content))
							}

							fileBlock := openai.ChatCompletionContentPartFileParam{
								File: openai.ChatCompletionContentPartFileFileParam{
									Filename: param.NewOpt(filename),
									FileData: param.NewOpt(data),
								},
							}
							content = append(content, openai.ChatCompletionContentPartUnionParam{OfFile: &fileBlock})
						}

					default:
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: fmt.Sprintf("file part media type %s not supported", filePart.MediaType),
						})
					}
				}
			}
			if !hasVisibleUserContent(content) {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "dropping empty user message (contains neither user-facing content nor tool results)",
				})
				continue
			}
			messages = append(messages, openai.UserMessage(content))
		case fantasy.MessageRoleAssistant:
			// simple assistant message just text content
			if len(msg.Content) == 1 && msg.Content[0].GetType() == fantasy.ContentTypeText {
				textPart, ok := fantasy.AsContentType[fantasy.TextPart](msg.Content[0])
				if !ok {
					warnings = append(warnings, fantasy.CallWarning{
						Type:    fantasy.CallWarningTypeOther,
						Message: "assistant message text part does not have the right type",
					})
					continue
				}
				messages = append(messages, openai.AssistantMessage(textPart.Text))
				continue
			}
			assistantMsg := openai.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}
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
					assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(textPart.Text),
					}
				case fantasy.ContentTypeToolCall:
					toolCallPart, ok := fantasy.AsContentType[fantasy.ToolCallPart](c)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "assistant message tool part does not have the right type",
						})
						continue
					}
					assistantMsg.ToolCalls = append(assistantMsg.ToolCalls,
						openai.ChatCompletionMessageToolCallUnionParam{
							OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
								ID:   toolCallPart.ToolCallID,
								Type: "function",
								Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
									Name:      toolCallPart.ToolName,
									Arguments: toolCallPart.Input,
								},
							},
						})
				}
			}
			if !hasVisibleAssistantContent(&assistantMsg) {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "dropping empty assistant message (contains neither user-facing content nor tool calls)",
				})
				continue
			}
			messages = append(messages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})
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
					messages = append(messages, openai.ToolMessage(output.Text, toolResultPart.ToolCallID))
				case fantasy.ToolResultContentTypeError:
					// TODO: check if better handling is needed
					output, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](toolResultPart.Output)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "tool result output does not have the right type",
						})
						continue
					}
					messages = append(messages, openai.ToolMessage(output.Error.Error(), toolResultPart.ToolCallID))
				}
			}
		}
	}
	return messages, warnings
}

func hasVisibleUserContent(content []openai.ChatCompletionContentPartUnionParam) bool {
	for _, part := range content {
		if part.OfText != nil || part.OfImageURL != nil || part.OfInputAudio != nil || part.OfFile != nil {
			return true
		}
	}
	return false
}

func hasVisibleAssistantContent(msg *openai.ChatCompletionAssistantMessageParam) bool {
	// Check if there's text content
	if !param.IsOmitted(msg.Content.OfString) || len(msg.Content.OfArrayOfContentParts) > 0 {
		return true
	}
	// Check if there are tool calls
	if len(msg.ToolCalls) > 0 {
		return true
	}
	return false
}
