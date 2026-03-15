package openaicompat

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	openaisdk "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/shared"
)

const reasoningStartedCtx = "reasoning_started"

// PrepareCallFunc prepares the call for the language model.
func PrepareCallFunc(_ fantasy.LanguageModel, params *openaisdk.ChatCompletionNewParams, call fantasy.Call) ([]fantasy.CallWarning, error) {
	providerOptions := &ProviderOptions{}
	if v, ok := call.ProviderOptions[Name]; ok {
		providerOptions, ok = v.(*ProviderOptions)
		if !ok {
			return nil, &fantasy.Error{Title: "invalid argument", Message: "openai-compat provider options should be *openaicompat.ProviderOptions"}
		}
	}

	if providerOptions.ReasoningEffort != nil {
		switch *providerOptions.ReasoningEffort {
		case openai.ReasoningEffortMinimal:
			params.ReasoningEffort = shared.ReasoningEffortMinimal
		case openai.ReasoningEffortLow:
			params.ReasoningEffort = shared.ReasoningEffortLow
		case openai.ReasoningEffortMedium:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case openai.ReasoningEffortHigh:
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			return nil, fmt.Errorf("reasoning model `%s` not supported", *providerOptions.ReasoningEffort)
		}
	}

	if providerOptions.User != nil {
		params.User = param.NewOpt(*providerOptions.User)
	}
	return nil, nil
}

// ExtraContentFunc adds extra content to the response.
func ExtraContentFunc(choice openaisdk.ChatCompletionChoice) []fantasy.Content {
	var content []fantasy.Content
	reasoningData := ReasoningData{}
	err := json.Unmarshal([]byte(choice.Message.RawJSON()), &reasoningData)
	if err != nil {
		return content
	}
	if reasoningData.ReasoningContent != "" {
		content = append(content, fantasy.ReasoningContent{
			Text: reasoningData.ReasoningContent,
		})
	}
	return content
}

func extractReasoningContext(ctx map[string]any) bool {
	reasoningStarted, ok := ctx[reasoningStartedCtx]
	if !ok {
		return false
	}
	b, ok := reasoningStarted.(bool)
	if !ok {
		return false
	}
	return b
}

// StreamExtraFunc handles extra functionality for streaming responses.
func StreamExtraFunc(chunk openaisdk.ChatCompletionChunk, yield func(fantasy.StreamPart) bool, ctx map[string]any) (map[string]any, bool) {
	if len(chunk.Choices) == 0 {
		return ctx, true
	}

	reasoningStarted := extractReasoningContext(ctx)

	for inx, choice := range chunk.Choices {
		reasoningData := ReasoningData{}
		err := json.Unmarshal([]byte(choice.Delta.RawJSON()), &reasoningData)
		if err != nil {
			yield(fantasy.StreamPart{
				Type:  fantasy.StreamPartTypeError,
				Error: &fantasy.Error{Title: "stream error", Message: "error unmarshalling delta", Cause: err},
			})
			return ctx, false
		}

		emitEvent := func(reasoningContent string) bool {
			if !reasoningStarted {
				shouldContinue := yield(fantasy.StreamPart{
					Type: fantasy.StreamPartTypeReasoningStart,
					ID:   fmt.Sprintf("%d", inx),
				})
				if !shouldContinue {
					return false
				}
			}

			return yield(fantasy.StreamPart{
				Type:  fantasy.StreamPartTypeReasoningDelta,
				ID:    fmt.Sprintf("%d", inx),
				Delta: reasoningContent,
			})
		}
		if reasoningData.ReasoningContent != "" {
			if !reasoningStarted {
				ctx[reasoningStartedCtx] = true
			}
			return ctx, emitEvent(reasoningData.ReasoningContent)
		}
		if reasoningStarted && (choice.Delta.Content != "" || len(choice.Delta.ToolCalls) > 0) {
			ctx[reasoningStartedCtx] = false
			return ctx, yield(fantasy.StreamPart{
				Type: fantasy.StreamPartTypeReasoningEnd,
				ID:   fmt.Sprintf("%d", inx),
			})
		}
	}
	return ctx, true
}

// ToPromptFunc converts a fantasy prompt to OpenAI format with reasoning support.
// It handles fantasy.ContentTypeReasoning in assistant messages by adding the
// reasoning_content field to the message JSON.
func ToPromptFunc(prompt fantasy.Prompt, _, _ string) ([]openaisdk.ChatCompletionMessageParamUnion, []fantasy.CallWarning) {
	var messages []openaisdk.ChatCompletionMessageParamUnion
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
			messages = append(messages, openaisdk.SystemMessage(strings.Join(systemPromptParts, "\n")))
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
				messages = append(messages, openaisdk.UserMessage(textPart.Text))
				continue
			}
			// text content and attachments
			var content []openaisdk.ChatCompletionContentPartUnionParam
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
					content = append(content, openaisdk.ChatCompletionContentPartUnionParam{
						OfText: &openaisdk.ChatCompletionContentPartTextParam{
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
						imageURL := openaisdk.ChatCompletionContentPartImageImageURLParam{URL: data}

						// Check for provider-specific options like image detail
						if providerOptions, ok := filePart.ProviderOptions[openai.Name]; ok {
							if detail, ok := providerOptions.(*openai.ProviderFileOptions); ok {
								imageURL.Detail = detail.ImageDetail
							}
						}

						imageBlock := openaisdk.ChatCompletionContentPartImageParam{ImageURL: imageURL}
						content = append(content, openaisdk.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})

					case filePart.MediaType == "audio/wav":
						// Handle WAV audio files
						base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
						audioBlock := openaisdk.ChatCompletionContentPartInputAudioParam{
							InputAudio: openaisdk.ChatCompletionContentPartInputAudioInputAudioParam{
								Data:   base64Encoded,
								Format: "wav",
							},
						}
						content = append(content, openaisdk.ChatCompletionContentPartUnionParam{OfInputAudio: &audioBlock})

					case filePart.MediaType == "audio/mpeg" || filePart.MediaType == "audio/mp3":
						// Handle MP3 audio files
						base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
						audioBlock := openaisdk.ChatCompletionContentPartInputAudioParam{
							InputAudio: openaisdk.ChatCompletionContentPartInputAudioInputAudioParam{
								Data:   base64Encoded,
								Format: "mp3",
							},
						}
						content = append(content, openaisdk.ChatCompletionContentPartUnionParam{OfInputAudio: &audioBlock})

					case filePart.MediaType == "application/pdf":
						// Handle PDF files
						dataStr := string(filePart.Data)

						// Check if data looks like a file ID (starts with "file-")
						if strings.HasPrefix(dataStr, "file-") {
							fileBlock := openaisdk.ChatCompletionContentPartFileParam{
								File: openaisdk.ChatCompletionContentPartFileFileParam{
									FileID: param.NewOpt(dataStr),
								},
							}
							content = append(content, openaisdk.ChatCompletionContentPartUnionParam{OfFile: &fileBlock})
						} else {
							// Handle as base64 data
							base64Encoded := base64.StdEncoding.EncodeToString(filePart.Data)
							data := "data:application/pdf;base64," + base64Encoded

							filename := filePart.Filename
							if filename == "" {
								// Generate default filename based on content index
								filename = fmt.Sprintf("part-%d.pdf", len(content))
							}

							fileBlock := openaisdk.ChatCompletionContentPartFileParam{
								File: openaisdk.ChatCompletionContentPartFileFileParam{
									Filename: param.NewOpt(filename),
									FileData: param.NewOpt(data),
								},
							}
							content = append(content, openaisdk.ChatCompletionContentPartUnionParam{OfFile: &fileBlock})
						}

					default:
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: fmt.Sprintf("file part media type %s not supported", filePart.MediaType),
						})
					}
				}
			}
			if !hasVisibleCompatUserContent(content) {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "dropping empty user message (contains neither user-facing content nor tool results)",
				})
				continue
			}
			messages = append(messages, openaisdk.UserMessage(content))
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
				messages = append(messages, openaisdk.AssistantMessage(textPart.Text))
				continue
			}
			assistantMsg := openaisdk.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}
			var reasoningText string
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
					assistantMsg.Content = openaisdk.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(textPart.Text),
					}
				case fantasy.ContentTypeReasoning:
					reasoningPart, ok := fantasy.AsContentType[fantasy.ReasoningPart](c)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "assistant message reasoning part does not have the right type",
						})
						continue
					}
					reasoningText = reasoningPart.Text
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
						openaisdk.ChatCompletionMessageToolCallUnionParam{
							OfFunction: &openaisdk.ChatCompletionMessageFunctionToolCallParam{
								ID:   toolCallPart.ToolCallID,
								Type: "function",
								Function: openaisdk.ChatCompletionMessageFunctionToolCallFunctionParam{
									Name:      toolCallPart.ToolName,
									Arguments: toolCallPart.Input,
								},
							},
						})
				}
			}
			// Add reasoning_content field if present
			if reasoningText != "" {
				assistantMsg.SetExtraFields(map[string]any{
					"reasoning_content": reasoningText,
				})
			}
			if !hasVisibleCompatAssistantContent(&assistantMsg) {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "dropping empty assistant message (contains neither user-facing content nor tool calls)",
				})
				continue
			}
			messages = append(messages, openaisdk.ChatCompletionMessageParamUnion{
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
					messages = append(messages, openaisdk.ToolMessage(output.Text, toolResultPart.ToolCallID))
				case fantasy.ToolResultContentTypeError:
					output, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](toolResultPart.Output)
					if !ok {
						warnings = append(warnings, fantasy.CallWarning{
							Type:    fantasy.CallWarningTypeOther,
							Message: "tool result output does not have the right type",
						})
						continue
					}
					messages = append(messages, openaisdk.ToolMessage(output.Error.Error(), toolResultPart.ToolCallID))
				}
			}
		}
	}
	return messages, warnings
}

func hasVisibleCompatUserContent(content []openaisdk.ChatCompletionContentPartUnionParam) bool {
	for _, part := range content {
		if part.OfText != nil || part.OfImageURL != nil || part.OfInputAudio != nil || part.OfFile != nil {
			return true
		}
	}
	return false
}

func hasVisibleCompatAssistantContent(msg *openaisdk.ChatCompletionAssistantMessageParam) bool {
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
