// Package object provides utilities for generating structured objects with automatic schema generation.
// It simplifies working with typed structured outputs by handling schema reflection and unmarshaling.
package object

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"charm.land/fantasy"
	"charm.land/fantasy/schema"
)

// Generate generates a structured object that matches the given type T.
// The schema is automatically generated from T using reflection.
//
// Example:
//
//	type Recipe struct {
//	    Name        string   `json:"name"`
//	    Ingredients []string `json:"ingredients"`
//	}
//
//	result, err := object.Generate[Recipe](ctx, model, fantasy.ObjectCall{
//	    Prompt: fantasy.Prompt{fantasy.NewUserMessage("Generate a lasagna recipe")},
//	})
func Generate[T any](
	ctx context.Context,
	model fantasy.LanguageModel,
	opts fantasy.ObjectCall,
) (*fantasy.ObjectResult[T], error) {
	var zero T
	s := schema.Generate(reflect.TypeOf(zero))
	opts.Schema = s

	resp, err := model.GenerateObject(ctx, opts)
	if err != nil {
		return nil, err
	}

	var result T
	if err := unmarshal(resp.Object, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to %T: %w", result, err)
	}

	return &fantasy.ObjectResult[T]{
		Object:           result,
		RawText:          resp.RawText,
		Usage:            resp.Usage,
		FinishReason:     resp.FinishReason,
		Warnings:         resp.Warnings,
		ProviderMetadata: resp.ProviderMetadata,
	}, nil
}

// Stream streams a structured object that matches the given type T.
// Returns a StreamObjectResult[T] with progressive updates and deduplication.
//
// Example:
//
//	stream, err := object.Stream[Recipe](ctx, model, fantasy.ObjectCall{
//	    Prompt: fantasy.Prompt{fantasy.NewUserMessage("Generate a lasagna recipe")},
//	})
//
//	for partial := range stream.PartialObjectStream() {
//	    fmt.Printf("Progress: %s\n", partial.Name)
//	}
//
//	result, err := stream.Object()  // Wait for final result
func Stream[T any](
	ctx context.Context,
	model fantasy.LanguageModel,
	opts fantasy.ObjectCall,
) (*fantasy.StreamObjectResult[T], error) {
	var zero T
	s := schema.Generate(reflect.TypeOf(zero))
	opts.Schema = s

	stream, err := model.StreamObject(ctx, opts)
	if err != nil {
		return nil, err
	}

	return fantasy.NewStreamObjectResult[T](ctx, stream), nil
}

// GenerateWithTool is a helper for providers without native JSON mode.
// It converts the schema to a tool definition, forces the model to call it,
// and extracts the tool's input as the structured output.
func GenerateWithTool(
	ctx context.Context,
	model fantasy.LanguageModel,
	call fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	toolName := call.SchemaName
	if toolName == "" {
		toolName = "generate_object"
	}

	toolDescription := call.SchemaDescription
	if toolDescription == "" {
		toolDescription = "Generate a structured object matching the schema"
	}

	tool := fantasy.FunctionTool{
		Name:        toolName,
		Description: toolDescription,
		InputSchema: schema.ToMap(call.Schema),
	}

	toolChoice := fantasy.SpecificToolChoice(tool.Name)
	resp, err := model.Generate(ctx, fantasy.Call{
		Prompt:           call.Prompt,
		Tools:            []fantasy.Tool{tool},
		ToolChoice:       &toolChoice,
		MaxOutputTokens:  call.MaxOutputTokens,
		Temperature:      call.Temperature,
		TopP:             call.TopP,
		TopK:             call.TopK,
		PresencePenalty:  call.PresencePenalty,
		FrequencyPenalty: call.FrequencyPenalty,
		ProviderOptions:  call.ProviderOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("tool-based generation failed: %w", err)
	}

	toolCalls := resp.Content.ToolCalls()
	if len(toolCalls) == 0 {
		return nil, &fantasy.NoObjectGeneratedError{
			RawText:      resp.Content.Text(),
			ParseError:   fmt.Errorf("no tool call generated"),
			Usage:        resp.Usage,
			FinishReason: resp.FinishReason,
		}
	}

	toolCall := toolCalls[0]

	var obj any
	if call.RepairText != nil {
		obj, err = schema.ParseAndValidateWithRepair(ctx, toolCall.Input, call.Schema, call.RepairText)
	} else {
		obj, err = schema.ParseAndValidate(toolCall.Input, call.Schema)
	}

	if err != nil {
		if nogErr, ok := err.(*fantasy.NoObjectGeneratedError); ok {
			nogErr.Usage = resp.Usage
			nogErr.FinishReason = resp.FinishReason
		}
		return nil, err
	}

	return &fantasy.ObjectResponse{
		Object:           obj,
		RawText:          toolCall.Input,
		Usage:            resp.Usage,
		FinishReason:     resp.FinishReason,
		Warnings:         resp.Warnings,
		ProviderMetadata: resp.ProviderMetadata,
	}, nil
}

// GenerateWithText is a helper for providers without tool or JSON mode support.
// It adds the schema to the system prompt and parses the text response as JSON.
// This is a fallback for older models or simple providers.
func GenerateWithText(
	ctx context.Context,
	model fantasy.LanguageModel,
	call fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	jsonSchemaBytes, err := json.Marshal(call.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	schemaInstruction := fmt.Sprintf(
		"You must respond with valid JSON that matches this schema: %s\n"+
			"Respond ONLY with the JSON object, no additional text or explanation.",
		string(jsonSchemaBytes),
	)

	enhancedPrompt := make(fantasy.Prompt, 0, len(call.Prompt)+1)

	hasSystem := false
	for _, msg := range call.Prompt {
		if msg.Role == fantasy.MessageRoleSystem {
			hasSystem = true
			existingText := ""
			if len(msg.Content) > 0 {
				if textPart, ok := msg.Content[0].(fantasy.TextPart); ok {
					existingText = textPart.Text
				}
			}
			enhancedPrompt = append(enhancedPrompt, fantasy.NewSystemMessage(existingText+"\n\n"+schemaInstruction))
		} else {
			enhancedPrompt = append(enhancedPrompt, msg)
		}
	}

	if !hasSystem {
		enhancedPrompt = append(fantasy.Prompt{fantasy.NewSystemMessage(schemaInstruction)}, call.Prompt...)
	}

	resp, err := model.Generate(ctx, fantasy.Call{
		Prompt:           enhancedPrompt,
		MaxOutputTokens:  call.MaxOutputTokens,
		Temperature:      call.Temperature,
		TopP:             call.TopP,
		TopK:             call.TopK,
		PresencePenalty:  call.PresencePenalty,
		FrequencyPenalty: call.FrequencyPenalty,
		ProviderOptions:  call.ProviderOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("text-based generation failed: %w", err)
	}

	textContent := resp.Content.Text()
	if textContent == "" {
		return nil, &fantasy.NoObjectGeneratedError{
			RawText:      "",
			ParseError:   fmt.Errorf("no text content in response"),
			Usage:        resp.Usage,
			FinishReason: resp.FinishReason,
		}
	}

	var obj any
	if call.RepairText != nil {
		obj, err = schema.ParseAndValidateWithRepair(ctx, textContent, call.Schema, call.RepairText)
	} else {
		obj, err = schema.ParseAndValidate(textContent, call.Schema)
	}

	if err != nil {
		if nogErr, ok := err.(*schema.ParseError); ok {
			return nil, &fantasy.NoObjectGeneratedError{
				RawText:         nogErr.RawText,
				ParseError:      nogErr.ParseError,
				ValidationError: nogErr.ValidationError,
				Usage:           resp.Usage,
				FinishReason:    resp.FinishReason,
			}
		}
		return nil, err
	}

	return &fantasy.ObjectResponse{
		Object:           obj,
		RawText:          textContent,
		Usage:            resp.Usage,
		FinishReason:     resp.FinishReason,
		Warnings:         resp.Warnings,
		ProviderMetadata: resp.ProviderMetadata,
	}, nil
}

// StreamWithTool is a helper for providers without native JSON streaming.
// It uses streaming tool calls to extract and parse the structured output progressively.
func StreamWithTool(
	ctx context.Context,
	model fantasy.LanguageModel,
	call fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	// Create a tool from the schema
	toolName := call.SchemaName
	if toolName == "" {
		toolName = "generate_object"
	}

	toolDescription := call.SchemaDescription
	if toolDescription == "" {
		toolDescription = "Generate a structured object matching the schema"
	}

	tool := fantasy.FunctionTool{
		Name:        toolName,
		Description: toolDescription,
		InputSchema: schema.ToMap(call.Schema),
	}

	// Make a streaming Generate call with forced tool choice
	toolChoice := fantasy.SpecificToolChoice(tool.Name)
	stream, err := model.Stream(ctx, fantasy.Call{
		Prompt:           call.Prompt,
		Tools:            []fantasy.Tool{tool},
		ToolChoice:       &toolChoice,
		MaxOutputTokens:  call.MaxOutputTokens,
		Temperature:      call.Temperature,
		TopP:             call.TopP,
		TopK:             call.TopK,
		PresencePenalty:  call.PresencePenalty,
		FrequencyPenalty: call.FrequencyPenalty,
		ProviderOptions:  call.ProviderOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("tool-based streaming failed: %w", err)
	}

	// Convert the text stream to object stream parts
	return func(yield func(fantasy.ObjectStreamPart) bool) {
		var accumulated string
		var lastParsedObject any
		var usage fantasy.Usage
		var finishReason fantasy.FinishReason
		var warnings []fantasy.CallWarning
		var providerMetadata fantasy.ProviderMetadata
		var streamErr error

		for part := range stream {
			switch part.Type {
			case fantasy.StreamPartTypeTextDelta:
				accumulated += part.Delta

				obj, state, parseErr := schema.ParsePartialJSON(accumulated)

				if state == schema.ParseStateSuccessful || state == schema.ParseStateRepaired {
					if err := schema.ValidateAgainstSchema(obj, call.Schema); err == nil {
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

			case fantasy.StreamPartTypeToolInputDelta:
				accumulated += part.Delta

				obj, state, parseErr := schema.ParsePartialJSON(accumulated)
				if state == schema.ParseStateSuccessful || state == schema.ParseStateRepaired {
					if err := schema.ValidateAgainstSchema(obj, call.Schema); err == nil {
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

			case fantasy.StreamPartTypeToolCall:
				toolInput := part.ToolCallInput

				var obj any
				var err error
				if call.RepairText != nil {
					obj, err = schema.ParseAndValidateWithRepair(ctx, toolInput, call.Schema, call.RepairText)
				} else {
					obj, err = schema.ParseAndValidate(toolInput, call.Schema)
				}

				if err == nil {
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

			case fantasy.StreamPartTypeError:
				streamErr = part.Error
				if !yield(fantasy.ObjectStreamPart{
					Type:  fantasy.ObjectStreamPartTypeError,
					Error: part.Error,
				}) {
					return
				}

			case fantasy.StreamPartTypeFinish:
				usage = part.Usage
				finishReason = part.FinishReason

			case fantasy.StreamPartTypeWarnings:
				warnings = part.Warnings
			}

			if len(part.ProviderMetadata) > 0 {
				providerMetadata = part.ProviderMetadata
			}
		}

		if streamErr == nil && lastParsedObject != nil {
			yield(fantasy.ObjectStreamPart{
				Type:             fantasy.ObjectStreamPartTypeFinish,
				Usage:            usage,
				FinishReason:     finishReason,
				Warnings:         warnings,
				ProviderMetadata: providerMetadata,
			})
		} else if streamErr == nil && lastParsedObject == nil {
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

// StreamWithText is a helper for providers without tool or JSON streaming support.
// It adds the schema to the system prompt and parses the streamed text as JSON progressively.
func StreamWithText(
	ctx context.Context,
	model fantasy.LanguageModel,
	call fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	jsonSchemaMap := schema.ToMap(call.Schema)
	jsonSchemaBytes, err := json.Marshal(jsonSchemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	schemaInstruction := fmt.Sprintf(
		"You must respond with valid JSON that matches this schema: %s\n"+
			"Respond ONLY with the JSON object, no additional text or explanation.",
		string(jsonSchemaBytes),
	)

	enhancedPrompt := make(fantasy.Prompt, 0, len(call.Prompt)+1)

	hasSystem := false
	for _, msg := range call.Prompt {
		if msg.Role == fantasy.MessageRoleSystem {
			hasSystem = true
			existingText := ""
			if len(msg.Content) > 0 {
				if textPart, ok := msg.Content[0].(fantasy.TextPart); ok {
					existingText = textPart.Text
				}
			}
			enhancedPrompt = append(enhancedPrompt, fantasy.NewSystemMessage(existingText+"\n\n"+schemaInstruction))
		} else {
			enhancedPrompt = append(enhancedPrompt, msg)
		}
	}

	if !hasSystem {
		enhancedPrompt = append(fantasy.Prompt{fantasy.NewSystemMessage(schemaInstruction)}, call.Prompt...)
	}

	stream, err := model.Stream(ctx, fantasy.Call{
		Prompt:           enhancedPrompt,
		MaxOutputTokens:  call.MaxOutputTokens,
		Temperature:      call.Temperature,
		TopP:             call.TopP,
		TopK:             call.TopK,
		PresencePenalty:  call.PresencePenalty,
		FrequencyPenalty: call.FrequencyPenalty,
		ProviderOptions:  call.ProviderOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("text-based streaming failed: %w", err)
	}

	return func(yield func(fantasy.ObjectStreamPart) bool) {
		var accumulated string
		var lastParsedObject any
		var usage fantasy.Usage
		var finishReason fantasy.FinishReason
		var warnings []fantasy.CallWarning
		var providerMetadata fantasy.ProviderMetadata
		var streamErr error

		for part := range stream {
			switch part.Type {
			case fantasy.StreamPartTypeTextDelta:
				accumulated += part.Delta

				obj, state, parseErr := schema.ParsePartialJSON(accumulated)

				if state == schema.ParseStateSuccessful || state == schema.ParseStateRepaired {
					if err := schema.ValidateAgainstSchema(obj, call.Schema); err == nil {
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

			case fantasy.StreamPartTypeError:
				streamErr = part.Error
				if !yield(fantasy.ObjectStreamPart{
					Type:  fantasy.ObjectStreamPartTypeError,
					Error: part.Error,
				}) {
					return
				}

			case fantasy.StreamPartTypeFinish:
				usage = part.Usage
				finishReason = part.FinishReason

			case fantasy.StreamPartTypeWarnings:
				warnings = part.Warnings
			}

			if len(part.ProviderMetadata) > 0 {
				providerMetadata = part.ProviderMetadata
			}
		}

		if streamErr == nil && lastParsedObject != nil {
			yield(fantasy.ObjectStreamPart{
				Type:             fantasy.ObjectStreamPartTypeFinish,
				Usage:            usage,
				FinishReason:     finishReason,
				Warnings:         warnings,
				ProviderMetadata: providerMetadata,
			})
		} else if streamErr == nil && lastParsedObject == nil {
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

func unmarshal(obj any, target any) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal into target type: %w", err)
	}

	return nil
}
