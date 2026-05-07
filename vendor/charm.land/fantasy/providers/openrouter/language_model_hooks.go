package openrouter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/google"
	"charm.land/fantasy/providers/openai"
	openaisdk "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/param"
)

const reasoningStartedCtx = "reasoning_started"

func languagePrepareModelCall(_ fantasy.LanguageModel, params *openaisdk.ChatCompletionNewParams, call fantasy.Call) ([]fantasy.CallWarning, error) {
	providerOptions := &ProviderOptions{}
	if v, ok := call.ProviderOptions[Name]; ok {
		providerOptions, ok = v.(*ProviderOptions)
		if !ok {
			return nil, &fantasy.Error{Title: "invalid argument", Message: "openrouter provider options should be *openrouter.ProviderOptions"}
		}
	}

	extraFields := make(map[string]any)

	if providerOptions.Provider != nil {
		data, err := structToMapJSON(providerOptions.Provider)
		if err != nil {
			return nil, err
		}
		extraFields["provider"] = data
	}

	if providerOptions.Reasoning != nil {
		data, err := structToMapJSON(providerOptions.Reasoning)
		if err != nil {
			return nil, err
		}
		extraFields["reasoning"] = data
	}

	if providerOptions.IncludeUsage != nil {
		extraFields["usage"] = map[string]any{
			"include": *providerOptions.IncludeUsage,
		}
	} else { // default include usage
		extraFields["usage"] = map[string]any{
			"include": true,
		}
	}
	if providerOptions.LogitBias != nil {
		params.LogitBias = providerOptions.LogitBias
	}
	if providerOptions.LogProbs != nil {
		params.Logprobs = param.NewOpt(*providerOptions.LogProbs)
	}
	if providerOptions.User != nil {
		params.User = param.NewOpt(*providerOptions.User)
	}
	if providerOptions.ParallelToolCalls != nil {
		params.ParallelToolCalls = param.NewOpt(*providerOptions.ParallelToolCalls)
	}

	maps.Copy(extraFields, providerOptions.ExtraBody)
	params.SetExtraFields(extraFields)
	return nil, nil
}

func languageModelExtraContent(choice openaisdk.ChatCompletionChoice) []fantasy.Content {
	content := make([]fantasy.Content, 0)
	reasoningData := ReasoningData{}
	err := json.Unmarshal([]byte(choice.Message.RawJSON()), &reasoningData)
	if err != nil {
		return content
	}
	type anthropicReasoningBlock struct {
		text     string
		metadata *anthropic.ReasoningOptionMetadata
	}
	type googleReasoningBlock struct {
		text     string
		metadata *google.ReasoningMetadata
	}

	responsesReasoningBlocks := make([]openai.ResponsesReasoningMetadata, 0)
	anthropicReasoningBlocks := make([]anthropicReasoningBlock, 0)
	googleReasoningBlocks := make([]googleReasoningBlock, 0)
	otherReasoning := make([]string, 0)
	for _, detail := range reasoningData.ReasoningDetails {
		if strings.HasPrefix(detail.Format, "openai-responses") || strings.HasPrefix(detail.Format, "xai-responses") {
			var thinkingBlock openai.ResponsesReasoningMetadata
			if len(responsesReasoningBlocks)-1 >= detail.Index {
				thinkingBlock = responsesReasoningBlocks[detail.Index]
			} else {
				thinkingBlock = openai.ResponsesReasoningMetadata{}
				responsesReasoningBlocks = append(responsesReasoningBlocks, thinkingBlock)
			}

			switch detail.Type {
			case "reasoning.summary":
				thinkingBlock.Summary = append(thinkingBlock.Summary, detail.Summary)
			case "reasoning.encrypted":
				thinkingBlock.EncryptedContent = &detail.Data
			}
			if detail.ID != "" {
				thinkingBlock.ItemID = detail.ID
			}

			responsesReasoningBlocks[detail.Index] = thinkingBlock
			continue
		}
		if strings.HasPrefix(detail.Format, "google-gemini") {
			var thinkingBlock googleReasoningBlock
			if len(googleReasoningBlocks)-1 >= detail.Index {
				thinkingBlock = googleReasoningBlocks[detail.Index]
			} else {
				thinkingBlock = googleReasoningBlock{metadata: &google.ReasoningMetadata{}}
				googleReasoningBlocks = append(googleReasoningBlocks, thinkingBlock)
			}

			switch detail.Type {
			case "reasoning.text":
				thinkingBlock.text = detail.Text
			case "reasoning.encrypted":
				thinkingBlock.metadata.Signature = detail.Data
				thinkingBlock.metadata.ToolID = detail.ID
			}

			googleReasoningBlocks[detail.Index] = thinkingBlock
			continue
		}

		if strings.HasPrefix(detail.Format, "anthropic-claude") {
			anthropicReasoningBlocks = append(anthropicReasoningBlocks, anthropicReasoningBlock{
				text: detail.Text,
				metadata: &anthropic.ReasoningOptionMetadata{
					Signature: detail.Signature,
				},
			})
			continue
		}

		otherReasoning = append(otherReasoning, detail.Text)
	}

	for _, block := range responsesReasoningBlocks {
		if len(block.Summary) == 0 {
			block.Summary = []string{""}
		}
		content = append(content, fantasy.ReasoningContent{
			Text: strings.Join(block.Summary, "\n"),
			ProviderMetadata: fantasy.ProviderMetadata{
				openai.Name: &block,
			},
		})
	}
	for _, block := range anthropicReasoningBlocks {
		content = append(content, fantasy.ReasoningContent{
			Text: block.text,
			ProviderMetadata: fantasy.ProviderMetadata{
				anthropic.Name: block.metadata,
			},
		})
	}
	for _, block := range googleReasoningBlocks {
		content = append(content, fantasy.ReasoningContent{
			Text: block.text,
			ProviderMetadata: fantasy.ProviderMetadata{
				google.Name: block.metadata,
			},
		})
	}

	for _, reasoning := range otherReasoning {
		content = append(content, fantasy.ReasoningContent{
			Text: reasoning,
		})
	}
	return content
}

type currentReasoningState struct {
	metadata       *openai.ResponsesReasoningMetadata
	googleMetadata *google.ReasoningMetadata
	googleText     string
}

func extractReasoningContext(ctx map[string]any) *currentReasoningState {
	reasoningStarted, ok := ctx[reasoningStartedCtx]
	if !ok {
		return nil
	}
	state, ok := reasoningStarted.(*currentReasoningState)
	if !ok {
		return nil
	}
	return state
}

func languageModelStreamExtra(chunk openaisdk.ChatCompletionChunk, yield func(fantasy.StreamPart) bool, ctx map[string]any) (map[string]any, bool) {
	if len(chunk.Choices) == 0 {
		return ctx, true
	}

	currentState := extractReasoningContext(ctx)

	inx := 0
	choice := chunk.Choices[inx]
	reasoningData := ReasoningData{}
	err := json.Unmarshal([]byte(choice.Delta.RawJSON()), &reasoningData)
	if err != nil {
		yield(fantasy.StreamPart{
			Type:  fantasy.StreamPartTypeError,
			Error: &fantasy.Error{Title: "stream error", Message: "error unmarshalling delta", Cause: err},
		})
		return ctx, false
	}

	// Reasoning Start
	if currentState == nil {
		if len(reasoningData.ReasoningDetails) == 0 {
			return ctx, true
		}

		var metadata fantasy.ProviderMetadata
		currentState = &currentReasoningState{}

		detail := reasoningData.ReasoningDetails[0]
		if strings.HasPrefix(detail.Format, "openai-responses") || strings.HasPrefix(detail.Format, "xai-responses") {
			currentState.metadata = &openai.ResponsesReasoningMetadata{
				Summary: []string{detail.Summary},
			}
			metadata = fantasy.ProviderMetadata{
				openai.Name: currentState.metadata,
			}
			// There was no summary just thinking we just send this as if it ended alredy
			if detail.Data != "" {
				shouldContinue := yield(fantasy.StreamPart{
					Type:             fantasy.StreamPartTypeReasoningStart,
					ID:               fmt.Sprintf("%d", inx),
					Delta:            detail.Summary,
					ProviderMetadata: metadata,
				})
				if !shouldContinue {
					return ctx, false
				}
				return ctx, yield(fantasy.StreamPart{
					Type: fantasy.StreamPartTypeReasoningEnd,
					ID:   fmt.Sprintf("%d", inx),
					ProviderMetadata: fantasy.ProviderMetadata{
						openai.Name: &openai.ResponsesReasoningMetadata{
							Summary:          []string{detail.Summary},
							EncryptedContent: &detail.Data,
							ItemID:           detail.ID,
						},
					},
				})
			}
		}

		if strings.HasPrefix(detail.Format, "google-gemini") {
			// this means there is only encrypted data available start and finish right away
			if detail.Type == "reasoning.encrypted" {
				ctx[reasoningStartedCtx] = nil
				if !yield(fantasy.StreamPart{
					Type: fantasy.StreamPartTypeReasoningStart,
					ID:   fmt.Sprintf("%d", inx),
				}) {
					return ctx, false
				}
				return ctx, yield(fantasy.StreamPart{
					Type: fantasy.StreamPartTypeReasoningEnd,
					ID:   fmt.Sprintf("%d", inx),
					ProviderMetadata: fantasy.ProviderMetadata{
						google.Name: &google.ReasoningMetadata{
							Signature: detail.Data,
							ToolID:    detail.ID,
						},
					},
				})
			}
			currentState.googleMetadata = &google.ReasoningMetadata{}
			currentState.googleText = detail.Text
			metadata = fantasy.ProviderMetadata{
				google.Name: currentState.googleMetadata,
			}
		}

		ctx[reasoningStartedCtx] = currentState
		delta := detail.Summary
		if strings.HasPrefix(detail.Format, "google-gemini") {
			delta = detail.Text
		}
		return ctx, yield(fantasy.StreamPart{
			Type:             fantasy.StreamPartTypeReasoningStart,
			ID:               fmt.Sprintf("%d", inx),
			Delta:            delta,
			ProviderMetadata: metadata,
		})
	}
	if len(reasoningData.ReasoningDetails) == 0 {
		// this means its a model different from openai/anthropic that ended reasoning
		if choice.Delta.Content != "" || len(choice.Delta.ToolCalls) > 0 {
			ctx[reasoningStartedCtx] = nil
			return ctx, yield(fantasy.StreamPart{
				Type: fantasy.StreamPartTypeReasoningEnd,
				ID:   fmt.Sprintf("%d", inx),
			})
		}
		return ctx, true
	}
	// Reasoning delta
	detail := reasoningData.ReasoningDetails[0]
	if strings.HasPrefix(detail.Format, "openai-responses") || strings.HasPrefix(detail.Format, "xai-responses") {
		// Reasoning has ended
		if detail.Data != "" {
			currentState.metadata.EncryptedContent = &detail.Data
			currentState.metadata.ItemID = detail.ID
			ctx[reasoningStartedCtx] = nil
			return ctx, yield(fantasy.StreamPart{
				Type: fantasy.StreamPartTypeReasoningEnd,
				ID:   fmt.Sprintf("%d", inx),
				ProviderMetadata: fantasy.ProviderMetadata{
					openai.Name: currentState.metadata,
				},
			})
		}
		var textDelta string
		// add to existing summary
		if len(currentState.metadata.Summary)-1 >= detail.Index {
			currentState.metadata.Summary[detail.Index] += detail.Summary
			textDelta = detail.Summary
		} else { // add new summary
			currentState.metadata.Summary = append(currentState.metadata.Summary, detail.Summary)
			textDelta = "\n" + detail.Summary
		}
		ctx[reasoningStartedCtx] = currentState
		return ctx, yield(fantasy.StreamPart{
			Type:  fantasy.StreamPartTypeReasoningDelta,
			ID:    fmt.Sprintf("%d", inx),
			Delta: textDelta,
			ProviderMetadata: fantasy.ProviderMetadata{
				openai.Name: currentState.metadata,
			},
		})
	}
	if strings.HasPrefix(detail.Format, "anthropic-claude") {
		// the reasoning has ended
		if detail.Signature != "" {
			metadata := fantasy.ProviderMetadata{
				anthropic.Name: &anthropic.ReasoningOptionMetadata{
					Signature: detail.Signature,
				},
			}
			// initial update
			shouldContinue := yield(fantasy.StreamPart{
				Type:             fantasy.StreamPartTypeReasoningDelta,
				ID:               fmt.Sprintf("%d", inx),
				Delta:            detail.Text,
				ProviderMetadata: metadata,
			})
			if !shouldContinue {
				return ctx, false
			}
			ctx[reasoningStartedCtx] = nil
			return ctx, yield(fantasy.StreamPart{
				Type: fantasy.StreamPartTypeReasoningEnd,
				ID:   fmt.Sprintf("%d", inx),
			})
		}

		return ctx, yield(fantasy.StreamPart{
			Type:  fantasy.StreamPartTypeReasoningDelta,
			ID:    fmt.Sprintf("%d", inx),
			Delta: detail.Text,
		})
	}

	if strings.HasPrefix(detail.Format, "google-gemini") {
		// reasoning.text type - accumulate text
		if detail.Type == "reasoning.text" {
			currentState.googleText += detail.Text
			ctx[reasoningStartedCtx] = currentState
			return ctx, yield(fantasy.StreamPart{
				Type:  fantasy.StreamPartTypeReasoningDelta,
				ID:    fmt.Sprintf("%d", inx),
				Delta: detail.Text,
			})
		}

		// reasoning.encrypted type - end reasoning with signature
		if detail.Type == "reasoning.encrypted" {
			currentState.googleMetadata.Signature = detail.Data
			currentState.googleMetadata.ToolID = detail.ID
			metadata := fantasy.ProviderMetadata{
				google.Name: currentState.googleMetadata,
			}
			ctx[reasoningStartedCtx] = nil
			return ctx, yield(fantasy.StreamPart{
				Type:             fantasy.StreamPartTypeReasoningEnd,
				ID:               fmt.Sprintf("%d", inx),
				ProviderMetadata: metadata,
			})
		}
	}

	return ctx, yield(fantasy.StreamPart{
		Type:  fantasy.StreamPartTypeReasoningDelta,
		ID:    fmt.Sprintf("%d", inx),
		Delta: detail.Text,
	})
}

func languageModelUsage(response openaisdk.ChatCompletion) (fantasy.Usage, fantasy.ProviderOptionsData) {
	if len(response.Choices) == 0 {
		return fantasy.Usage{}, nil
	}
	openrouterUsage := UsageAccounting{}
	usage := response.Usage

	_ = json.Unmarshal([]byte(usage.RawJSON()), &openrouterUsage)

	completionTokenDetails := usage.CompletionTokensDetails
	promptTokenDetails := usage.PromptTokensDetails

	var provider string
	if p, ok := response.JSON.ExtraFields["provider"]; ok {
		provider = p.Raw()
	}

	// Build provider metadata
	providerMetadata := &ProviderMetadata{
		Provider: provider,
		Usage:    openrouterUsage,
	}

	return fantasy.Usage{
		InputTokens:     usage.PromptTokens,
		OutputTokens:    usage.CompletionTokens,
		TotalTokens:     usage.TotalTokens,
		ReasoningTokens: completionTokenDetails.ReasoningTokens,
		CacheReadTokens: promptTokenDetails.CachedTokens,
	}, providerMetadata
}

func languageModelStreamUsage(chunk openaisdk.ChatCompletionChunk, _ map[string]any, metadata fantasy.ProviderMetadata) (fantasy.Usage, fantasy.ProviderMetadata) {
	usage := chunk.Usage
	if usage.TotalTokens == 0 {
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
	openrouterUsage := UsageAccounting{}
	_ = json.Unmarshal([]byte(usage.RawJSON()), &openrouterUsage)
	streamProviderMetadata.Usage = openrouterUsage

	if p, ok := chunk.JSON.ExtraFields["provider"]; ok {
		streamProviderMetadata.Provider = p.Raw()
	}

	// we do this here because the acc does not add prompt details
	completionTokenDetails := usage.CompletionTokensDetails
	promptTokenDetails := usage.PromptTokensDetails
	aiUsage := fantasy.Usage{
		InputTokens:     usage.PromptTokens,
		OutputTokens:    usage.CompletionTokens,
		TotalTokens:     usage.TotalTokens,
		ReasoningTokens: completionTokenDetails.ReasoningTokens,
		CacheReadTokens: promptTokenDetails.CachedTokens,
	}

	return aiUsage, fantasy.ProviderMetadata{
		Name: streamProviderMetadata,
	}
}

func languageModelToPrompt(prompt fantasy.Prompt, _, model string) ([]openaisdk.ChatCompletionMessageParamUnion, []fantasy.CallWarning) {
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
			systemMsg := openaisdk.SystemMessage(strings.Join(systemPromptParts, "\n"))
			anthropicCache := anthropic.GetCacheControl(msg.ProviderOptions)
			if anthropicCache != nil {
				systemMsg.OfSystem.SetExtraFields(map[string]any{
					"cache_control": map[string]string{
						"type": anthropicCache.Type,
					},
				})
			}
			messages = append(messages, systemMsg)
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
				userMsg := openaisdk.UserMessage(textPart.Text)

				anthropicCache := anthropic.GetCacheControl(msg.ProviderOptions)
				if anthropicCache != nil {
					userMsg.OfUser.SetExtraFields(map[string]any{
						"cache_control": map[string]string{
							"type": anthropicCache.Type,
						},
					})
				}
				messages = append(messages, userMsg)
				continue
			}
			// text content and attachments
			// for now we only support image content later we need to check
			// TODO: add the supported media types to the language model so we
			//  can use that to validate the data here.
			var content []openaisdk.ChatCompletionContentPartUnionParam
			for i, c := range msg.Content {
				isLastPart := i == len(msg.Content)-1
				cacheControl := anthropic.GetCacheControl(c.Options())
				if cacheControl == nil && isLastPart {
					cacheControl = anthropic.GetCacheControl(msg.ProviderOptions)
				}
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
					part := openaisdk.ChatCompletionContentPartUnionParam{
						OfText: &openaisdk.ChatCompletionContentPartTextParam{
							Text: textPart.Text,
						},
					}
					if cacheControl != nil {
						part.OfText.SetExtraFields(map[string]any{
							"cache_control": map[string]string{
								"type": cacheControl.Type,
							},
						})
					}
					content = append(content, part)
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
						if providerOptions, ok := filePart.ProviderOptions[Name]; ok {
							if detail, ok := providerOptions.(*openai.ProviderFileOptions); ok {
								imageURL.Detail = detail.ImageDetail
							}
						}

						imageBlock := openaisdk.ChatCompletionContentPartImageParam{ImageURL: imageURL}
						if cacheControl != nil {
							imageBlock.SetExtraFields(map[string]any{
								"cache_control": map[string]string{
									"type": cacheControl.Type,
								},
							})
						}
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
						if cacheControl != nil {
							audioBlock.SetExtraFields(map[string]any{
								"cache_control": map[string]string{
									"type": cacheControl.Type,
								},
							})
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
						if cacheControl != nil {
							audioBlock.SetExtraFields(map[string]any{
								"cache_control": map[string]string{
									"type": cacheControl.Type,
								},
							})
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

							if cacheControl != nil {
								fileBlock.SetExtraFields(map[string]any{
									"cache_control": map[string]string{
										"type": cacheControl.Type,
									},
								})
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
							if cacheControl != nil {
								fileBlock.SetExtraFields(map[string]any{
									"cache_control": map[string]string{
										"type": cacheControl.Type,
									},
								})
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

				assistantMsg := openaisdk.AssistantMessage(textPart.Text)
				anthropicCache := anthropic.GetCacheControl(msg.ProviderOptions)
				if anthropicCache != nil {
					assistantMsg.OfAssistant.SetExtraFields(map[string]any{
						"cache_control": map[string]string{
							"type": anthropicCache.Type,
						},
					})
				}
				messages = append(messages, assistantMsg)
				continue
			}
			assistantMsg := openaisdk.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}
			for i, c := range msg.Content {
				isLastPart := i == len(msg.Content)-1
				cacheControl := anthropic.GetCacheControl(c.Options())
				if cacheControl == nil && isLastPart {
					cacheControl = anthropic.GetCacheControl(msg.ProviderOptions)
				}
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
					// there is some text already there
					if assistantMsg.Content.OfString.Valid() {
						textPart.Text = assistantMsg.Content.OfString.Value + "\n" + textPart.Text
					}
					assistantMsg.Content = openaisdk.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(textPart.Text),
					}
					if cacheControl != nil {
						assistantMsg.Content.SetExtraFields(map[string]any{
							"cache_control": map[string]string{
								"type": cacheControl.Type,
							},
						})
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
					var reasoningDetails []ReasoningDetail
					switch {
					case strings.HasPrefix(model, "anthropic/") && reasoningPart.Text != "":
						metadata := anthropic.GetReasoningMetadata(reasoningPart.Options())
						if metadata == nil {
							text := fmt.Sprintf("<thoughts>%s</thoughts>", reasoningPart.Text)
							if assistantMsg.Content.OfString.Valid() {
								text = assistantMsg.Content.OfString.Value + "\n" + text
							}
							// this reasoning did not come from anthropic just add a text content
							assistantMsg.Content = openaisdk.ChatCompletionAssistantMessageParamContentUnion{
								OfString: param.NewOpt(text),
							}
							if cacheControl != nil {
								assistantMsg.Content.SetExtraFields(map[string]any{
									"cache_control": map[string]string{
										"type": cacheControl.Type,
									},
								})
							}
							continue
						}
						reasoningDetails = append(reasoningDetails, ReasoningDetail{
							Format:    "anthropic-claude-v1",
							Type:      "reasoning.text",
							Text:      reasoningPart.Text,
							Signature: metadata.Signature,
						})
						data, _ := json.Marshal(reasoningDetails)
						reasoningDetailsMap := []map[string]any{}
						_ = json.Unmarshal(data, &reasoningDetailsMap)
						assistantMsg.SetExtraFields(map[string]any{
							"reasoning_details": reasoningDetailsMap,
							"reasoning":         reasoningPart.Text,
						})
					case strings.HasPrefix(model, "openai/"):
						metadata := openai.GetReasoningMetadata(reasoningPart.Options())
						if metadata == nil {
							text := fmt.Sprintf("<thoughts>%s</thoughts>", reasoningPart.Text)
							if assistantMsg.Content.OfString.Valid() {
								text = assistantMsg.Content.OfString.Value + "\n" + text
							}
							// this reasoning did not come from anthropic just add a text content
							assistantMsg.Content = openaisdk.ChatCompletionAssistantMessageParamContentUnion{
								OfString: param.NewOpt(text),
							}
							continue
						}
						for inx, summary := range metadata.Summary {
							if summary == "" {
								continue
							}
							reasoningDetails = append(reasoningDetails, ReasoningDetail{
								Type:    "reasoning.summary",
								Format:  "openai-responses-v1",
								Summary: summary,
								Index:   inx,
							})
						}
						reasoningDetails = append(reasoningDetails, ReasoningDetail{
							Type:   "reasoning.encrypted",
							Format: "openai-responses-v1",
							Data:   *metadata.EncryptedContent,
							ID:     metadata.ItemID,
						})
						data, _ := json.Marshal(reasoningDetails)
						reasoningDetailsMap := []map[string]any{}
						_ = json.Unmarshal(data, &reasoningDetailsMap)
						assistantMsg.SetExtraFields(map[string]any{
							"reasoning_details": reasoningDetailsMap,
						})
					case strings.HasPrefix(model, "xai/"):
						metadata := openai.GetReasoningMetadata(reasoningPart.Options())
						if metadata == nil {
							text := fmt.Sprintf("<thoughts>%s</thoughts>", reasoningPart.Text)
							if assistantMsg.Content.OfString.Valid() {
								text = assistantMsg.Content.OfString.Value + "\n" + text
							}
							// this reasoning did not come from anthropic just add a text content
							assistantMsg.Content = openaisdk.ChatCompletionAssistantMessageParamContentUnion{
								OfString: param.NewOpt(text),
							}
							continue
						}
						for inx, summary := range metadata.Summary {
							if summary == "" {
								continue
							}
							reasoningDetails = append(reasoningDetails, ReasoningDetail{
								Type:    "reasoning.summary",
								Format:  "xai-responses-v1",
								Summary: summary,
								Index:   inx,
							})
						}
						reasoningDetails = append(reasoningDetails, ReasoningDetail{
							Type:   "reasoning.encrypted",
							Format: "xai-responses-v1",
							Data:   *metadata.EncryptedContent,
							ID:     metadata.ItemID,
						})
						data, _ := json.Marshal(reasoningDetails)
						reasoningDetailsMap := []map[string]any{}
						_ = json.Unmarshal(data, &reasoningDetailsMap)
						assistantMsg.SetExtraFields(map[string]any{
							"reasoning_details": reasoningDetailsMap,
						})
					case strings.HasPrefix(model, "google/"):
						metadata := google.GetReasoningMetadata(reasoningPart.Options())
						if metadata == nil {
							text := fmt.Sprintf("<thoughts>%s</thoughts>", reasoningPart.Text)
							if assistantMsg.Content.OfString.Valid() {
								text = assistantMsg.Content.OfString.Value + "\n" + text
							}
							// this reasoning did not come from anthropic just add a text content
							assistantMsg.Content = openaisdk.ChatCompletionAssistantMessageParamContentUnion{
								OfString: param.NewOpt(text),
							}
							continue
						}
						if reasoningPart.Text != "" {
							reasoningDetails = append(reasoningDetails, ReasoningDetail{
								Type:   "reasoning.text",
								Format: "google-gemini-v1",
								Text:   reasoningPart.Text,
							})
						}
						reasoningDetails = append(reasoningDetails, ReasoningDetail{
							Type:   "reasoning.encrypted",
							Format: "google-gemini-v1",
							Data:   metadata.Signature,
							ID:     metadata.ToolID,
						})
						data, _ := json.Marshal(reasoningDetails)
						reasoningDetailsMap := []map[string]any{}
						_ = json.Unmarshal(data, &reasoningDetailsMap)
						assistantMsg.SetExtraFields(map[string]any{
							"reasoning_details": reasoningDetailsMap,
						})
					default:
						reasoningDetails = append(reasoningDetails, ReasoningDetail{
							Type:   "reasoning.text",
							Text:   reasoningPart.Text,
							Format: "unknown",
						})
						data, _ := json.Marshal(reasoningDetails)
						reasoningDetailsMap := []map[string]any{}
						_ = json.Unmarshal(data, &reasoningDetailsMap)
						assistantMsg.SetExtraFields(map[string]any{
							"reasoning_details": reasoningDetailsMap,
						})
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
					tc := openaisdk.ChatCompletionMessageToolCallUnionParam{
						OfFunction: &openaisdk.ChatCompletionMessageFunctionToolCallParam{
							ID:   toolCallPart.ToolCallID,
							Type: "function",
							Function: openaisdk.ChatCompletionMessageFunctionToolCallFunctionParam{
								Name:      toolCallPart.ToolName,
								Arguments: toolCallPart.Input,
							},
						},
					}
					if cacheControl != nil {
						tc.OfFunction.SetExtraFields(map[string]any{
							"cache_control": map[string]string{
								"type": cacheControl.Type,
							},
						})
					}
					assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, tc)
				}
			}
			messages = append(messages, openaisdk.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})
		case fantasy.MessageRoleTool:
			for i, c := range msg.Content {
				isLastPart := i == len(msg.Content)-1
				cacheControl := anthropic.GetCacheControl(c.Options())
				if cacheControl == nil && isLastPart {
					cacheControl = anthropic.GetCacheControl(msg.ProviderOptions)
				}
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
					tr := openaisdk.ToolMessage(output.Text, toolResultPart.ToolCallID)
					if cacheControl != nil {
						tr.SetExtraFields(map[string]any{
							"cache_control": map[string]string{
								"type": cacheControl.Type,
							},
						})
					}
					messages = append(messages, tr)
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
					tr := openaisdk.ToolMessage(output.Error.Error(), toolResultPart.ToolCallID)
					if cacheControl != nil {
						tr.SetExtraFields(map[string]any{
							"cache_control": map[string]string{
								"type": cacheControl.Type,
							},
						})
					}
					messages = append(messages, tr)
				}
			}
		}
	}
	return messages, warnings
}
