// Package anthropic provides an implementation of the fantasy AI SDK for Anthropic's language models.
package anthropic

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/object"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/charmbracelet/anthropic-sdk-go"
	"github.com/charmbracelet/anthropic-sdk-go/bedrock"
	"github.com/charmbracelet/anthropic-sdk-go/option"
	"github.com/charmbracelet/anthropic-sdk-go/packages/param"
	"github.com/charmbracelet/anthropic-sdk-go/vertex"
	"golang.org/x/oauth2/google"
)

const (
	// Name is the name of the Anthropic provider.
	Name = "anthropic"
	// DefaultURL is the default URL for the Anthropic API.
	DefaultURL = "https://api.anthropic.com"
)

type options struct {
	baseURL string
	apiKey  string
	name    string
	headers map[string]string
	client  option.HTTPClient

	vertexProject  string
	vertexLocation string
	skipAuth       bool

	useBedrock bool

	objectMode fantasy.ObjectMode
}

type provider struct {
	options options
}

// Option defines a function that configures Anthropic provider options.
type Option = func(*options)

// New creates a new Anthropic provider with the given options.
func New(opts ...Option) (fantasy.Provider, error) {
	providerOptions := options{
		headers:    map[string]string{},
		objectMode: fantasy.ObjectModeAuto,
	}
	for _, o := range opts {
		o(&providerOptions)
	}

	providerOptions.baseURL = cmp.Or(providerOptions.baseURL, DefaultURL)
	providerOptions.name = cmp.Or(providerOptions.name, Name)
	return &provider{options: providerOptions}, nil
}

// WithBaseURL sets the base URL for the Anthropic provider.
func WithBaseURL(baseURL string) Option {
	return func(o *options) {
		o.baseURL = baseURL
	}
}

// WithAPIKey sets the API key for the Anthropic provider.
func WithAPIKey(apiKey string) Option {
	return func(o *options) {
		o.apiKey = apiKey
	}
}

// WithVertex configures the Anthropic provider to use Vertex AI.
func WithVertex(project, location string) Option {
	return func(o *options) {
		o.vertexProject = project
		o.vertexLocation = location
	}
}

// WithSkipAuth configures whether to skip authentication for the Anthropic provider.
func WithSkipAuth(skip bool) Option {
	return func(o *options) {
		o.skipAuth = skip
	}
}

// WithBedrock configures the Anthropic provider to use AWS Bedrock.
func WithBedrock() Option {
	return func(o *options) {
		o.useBedrock = true
	}
}

// WithName sets the name for the Anthropic provider.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithHeaders sets the headers for the Anthropic provider.
func WithHeaders(headers map[string]string) Option {
	return func(o *options) {
		maps.Copy(o.headers, headers)
	}
}

// WithHTTPClient sets the HTTP client for the Anthropic provider.
func WithHTTPClient(client option.HTTPClient) Option {
	return func(o *options) {
		o.client = client
	}
}

// WithObjectMode sets the object generation mode.
func WithObjectMode(om fantasy.ObjectMode) Option {
	return func(o *options) {
		// not supported
		if om == fantasy.ObjectModeJSON {
			om = fantasy.ObjectModeAuto
		}
		o.objectMode = om
	}
}

func (a *provider) LanguageModel(ctx context.Context, modelID string) (fantasy.LanguageModel, error) {
	clientOptions := make([]option.RequestOption, 0, 5+len(a.options.headers))
	clientOptions = append(clientOptions, option.WithMaxRetries(0))

	if a.options.apiKey != "" && !a.options.useBedrock {
		clientOptions = append(clientOptions, option.WithAPIKey(a.options.apiKey))
	}
	if a.options.baseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(a.options.baseURL))
	}
	for key, value := range a.options.headers {
		clientOptions = append(clientOptions, option.WithHeader(key, value))
	}
	if a.options.client != nil {
		clientOptions = append(clientOptions, option.WithHTTPClient(a.options.client))
	}
	if a.options.vertexProject != "" && a.options.vertexLocation != "" {
		var credentials *google.Credentials
		if a.options.skipAuth {
			credentials = &google.Credentials{TokenSource: &googleDummyTokenSource{}}
		} else {
			var err error
			credentials, err = google.FindDefaultCredentials(ctx)
			if err != nil {
				return nil, err
			}
		}

		clientOptions = append(
			clientOptions,
			vertex.WithCredentials(
				ctx,
				a.options.vertexLocation,
				a.options.vertexProject,
				credentials,
			),
		)
	}
	if a.options.useBedrock {
		modelID = bedrockPrefixModelWithRegion(modelID)

		if a.options.skipAuth || a.options.apiKey != "" {
			clientOptions = append(
				clientOptions,
				bedrock.WithConfig(bedrockBasicAuthConfig(a.options.apiKey)),
			)
		} else {
			if cfg, err := config.LoadDefaultConfig(ctx); err == nil {
				clientOptions = append(
					clientOptions,
					bedrock.WithConfig(cfg),
				)
			}
		}
	}
	return languageModel{
		modelID:  modelID,
		provider: a.options.name,
		options:  a.options,
		client:   anthropic.NewClient(clientOptions...),
	}, nil
}

type languageModel struct {
	provider string
	modelID  string
	client   anthropic.Client
	options  options
}

// Model implements fantasy.LanguageModel.
func (a languageModel) Model() string {
	return a.modelID
}

// Provider implements fantasy.LanguageModel.
func (a languageModel) Provider() string {
	return a.provider
}

func (a languageModel) prepareParams(call fantasy.Call) (*anthropic.MessageNewParams, []fantasy.CallWarning, error) {
	params := &anthropic.MessageNewParams{}
	providerOptions := &ProviderOptions{}
	if v, ok := call.ProviderOptions[Name]; ok {
		providerOptions, ok = v.(*ProviderOptions)
		if !ok {
			return nil, nil, &fantasy.Error{Title: "invalid argument", Message: "anthropic provider options should be *anthropic.ProviderOptions"}
		}
	}
	sendReasoning := true
	if providerOptions.SendReasoning != nil {
		sendReasoning = *providerOptions.SendReasoning
	}
	systemBlocks, messages, warnings := toPrompt(call.Prompt, sendReasoning)

	if call.FrequencyPenalty != nil {
		warnings = append(warnings, fantasy.CallWarning{
			Type:    fantasy.CallWarningTypeUnsupportedSetting,
			Setting: "FrequencyPenalty",
		})
	}
	if call.PresencePenalty != nil {
		warnings = append(warnings, fantasy.CallWarning{
			Type:    fantasy.CallWarningTypeUnsupportedSetting,
			Setting: "PresencePenalty",
		})
	}

	params.System = systemBlocks
	params.Messages = messages
	params.Model = anthropic.Model(a.modelID)
	params.MaxTokens = 4096

	if call.MaxOutputTokens != nil {
		params.MaxTokens = *call.MaxOutputTokens
	}

	if call.Temperature != nil {
		params.Temperature = param.NewOpt(*call.Temperature)
	}
	if call.TopK != nil {
		params.TopK = param.NewOpt(*call.TopK)
	}
	if call.TopP != nil {
		params.TopP = param.NewOpt(*call.TopP)
	}

	isThinking := false
	var thinkingBudget int64
	if providerOptions.Thinking != nil {
		isThinking = true
		thinkingBudget = providerOptions.Thinking.BudgetTokens
	}
	if isThinking {
		if thinkingBudget == 0 {
			return nil, nil, &fantasy.Error{Title: "no budget", Message: "thinking requires budget"}
		}
		params.Thinking = anthropic.ThinkingConfigParamOfEnabled(thinkingBudget)
		if call.Temperature != nil {
			params.Temperature = param.Opt[float64]{}
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "temperature",
				Details: "temperature is not supported when thinking is enabled",
			})
		}
		if call.TopP != nil {
			params.TopP = param.Opt[float64]{}
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "TopP",
				Details: "TopP is not supported when thinking is enabled",
			})
		}
		if call.TopK != nil {
			params.TopK = param.Opt[int64]{}
			warnings = append(warnings, fantasy.CallWarning{
				Type:    fantasy.CallWarningTypeUnsupportedSetting,
				Setting: "TopK",
				Details: "TopK is not supported when thinking is enabled",
			})
		}
	}

	if len(call.Tools) > 0 {
		disableParallelToolUse := false
		if providerOptions.DisableParallelToolUse != nil {
			disableParallelToolUse = *providerOptions.DisableParallelToolUse
		}
		tools, toolChoice, toolWarnings := a.toTools(call.Tools, call.ToolChoice, disableParallelToolUse)
		params.Tools = tools
		if toolChoice != nil {
			params.ToolChoice = *toolChoice
		}
		warnings = append(warnings, toolWarnings...)
	}

	return params, warnings, nil
}

func (a *provider) Name() string {
	return Name
}

// GetCacheControl extracts cache control settings from provider options.
func GetCacheControl(providerOptions fantasy.ProviderOptions) *CacheControl {
	if anthropicOptions, ok := providerOptions[Name]; ok {
		if options, ok := anthropicOptions.(*ProviderCacheControlOptions); ok {
			return &options.CacheControl
		}
	}
	return nil
}

// GetReasoningMetadata extracts reasoning metadata from provider options.
func GetReasoningMetadata(providerOptions fantasy.ProviderOptions) *ReasoningOptionMetadata {
	if anthropicOptions, ok := providerOptions[Name]; ok {
		if reasoning, ok := anthropicOptions.(*ReasoningOptionMetadata); ok {
			return reasoning
		}
	}
	return nil
}

type messageBlock struct {
	Role     fantasy.MessageRole
	Messages []fantasy.Message
}

func groupIntoBlocks(prompt fantasy.Prompt) []*messageBlock {
	var blocks []*messageBlock

	var currentBlock *messageBlock

	for _, msg := range prompt {
		switch msg.Role {
		case fantasy.MessageRoleSystem:
			if currentBlock == nil || currentBlock.Role != fantasy.MessageRoleSystem {
				currentBlock = &messageBlock{
					Role:     fantasy.MessageRoleSystem,
					Messages: []fantasy.Message{},
				}
				blocks = append(blocks, currentBlock)
			}
			currentBlock.Messages = append(currentBlock.Messages, msg)
		case fantasy.MessageRoleUser:
			if currentBlock == nil || currentBlock.Role != fantasy.MessageRoleUser {
				currentBlock = &messageBlock{
					Role:     fantasy.MessageRoleUser,
					Messages: []fantasy.Message{},
				}
				blocks = append(blocks, currentBlock)
			}
			currentBlock.Messages = append(currentBlock.Messages, msg)
		case fantasy.MessageRoleAssistant:
			if currentBlock == nil || currentBlock.Role != fantasy.MessageRoleAssistant {
				currentBlock = &messageBlock{
					Role:     fantasy.MessageRoleAssistant,
					Messages: []fantasy.Message{},
				}
				blocks = append(blocks, currentBlock)
			}
			currentBlock.Messages = append(currentBlock.Messages, msg)
		case fantasy.MessageRoleTool:
			if currentBlock == nil || currentBlock.Role != fantasy.MessageRoleUser {
				currentBlock = &messageBlock{
					Role:     fantasy.MessageRoleUser,
					Messages: []fantasy.Message{},
				}
				blocks = append(blocks, currentBlock)
			}
			currentBlock.Messages = append(currentBlock.Messages, msg)
		}
	}
	return blocks
}

func (a languageModel) toTools(tools []fantasy.Tool, toolChoice *fantasy.ToolChoice, disableParallelToolCalls bool) (anthropicTools []anthropic.ToolUnionParam, anthropicToolChoice *anthropic.ToolChoiceUnionParam, warnings []fantasy.CallWarning) {
	for _, tool := range tools {
		if tool.GetType() == fantasy.ToolTypeFunction {
			ft, ok := tool.(fantasy.FunctionTool)
			if !ok {
				continue
			}
			required := []string{}
			var properties any
			if props, ok := ft.InputSchema["properties"]; ok {
				properties = props
			}
			if req, ok := ft.InputSchema["required"]; ok {
				if reqArr, ok := req.([]string); ok {
					required = reqArr
				}
			}
			cacheControl := GetCacheControl(ft.ProviderOptions)

			anthropicTool := anthropic.ToolParam{
				Name:        ft.Name,
				Description: anthropic.String(ft.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: properties,
					Required:   required,
				},
			}
			if cacheControl != nil {
				anthropicTool.CacheControl = anthropic.NewCacheControlEphemeralParam()
			}
			anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{OfTool: &anthropicTool})
			continue
		}
		// TODO: handle provider tool calls
		warnings = append(warnings, fantasy.CallWarning{
			Type:    fantasy.CallWarningTypeUnsupportedTool,
			Tool:    tool,
			Message: "tool is not supported",
		})
	}

	// NOTE: Bedrock does not support this attribute.
	var disableParallelToolUse param.Opt[bool]
	if !a.options.useBedrock {
		disableParallelToolUse = param.NewOpt(disableParallelToolCalls)
	}

	if toolChoice == nil {
		if disableParallelToolCalls {
			anthropicToolChoice = &anthropic.ToolChoiceUnionParam{
				OfAuto: &anthropic.ToolChoiceAutoParam{
					Type:                   "auto",
					DisableParallelToolUse: disableParallelToolUse,
				},
			}
		}
		return anthropicTools, anthropicToolChoice, warnings
	}

	switch *toolChoice {
	case fantasy.ToolChoiceAuto:
		anthropicToolChoice = &anthropic.ToolChoiceUnionParam{
			OfAuto: &anthropic.ToolChoiceAutoParam{
				Type:                   "auto",
				DisableParallelToolUse: disableParallelToolUse,
			},
		}
	case fantasy.ToolChoiceRequired:
		anthropicToolChoice = &anthropic.ToolChoiceUnionParam{
			OfAny: &anthropic.ToolChoiceAnyParam{
				Type:                   "any",
				DisableParallelToolUse: disableParallelToolUse,
			},
		}
	case fantasy.ToolChoiceNone:
		return anthropicTools, anthropicToolChoice, warnings
	default:
		anthropicToolChoice = &anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{
				Type:                   "tool",
				Name:                   string(*toolChoice),
				DisableParallelToolUse: disableParallelToolUse,
			},
		}
	}
	return anthropicTools, anthropicToolChoice, warnings
}

func toPrompt(prompt fantasy.Prompt, sendReasoningData bool) ([]anthropic.TextBlockParam, []anthropic.MessageParam, []fantasy.CallWarning) {
	var systemBlocks []anthropic.TextBlockParam
	var messages []anthropic.MessageParam
	var warnings []fantasy.CallWarning

	blocks := groupIntoBlocks(prompt)
	finishedSystemBlock := false
	for _, block := range blocks {
		switch block.Role {
		case fantasy.MessageRoleSystem:
			if finishedSystemBlock {
				// skip multiple system messages that are separated by user/assistant messages
				// TODO: see if we need to send error here?
				continue
			}
			finishedSystemBlock = true
			for _, msg := range block.Messages {
				for i, part := range msg.Content {
					isLastPart := i == len(msg.Content)-1
					cacheControl := GetCacheControl(part.Options())
					if cacheControl == nil && isLastPart {
						cacheControl = GetCacheControl(msg.ProviderOptions)
					}
					text, ok := fantasy.AsMessagePart[fantasy.TextPart](part)
					if !ok {
						continue
					}
					textBlock := anthropic.TextBlockParam{
						Text: text.Text,
					}
					if cacheControl != nil {
						textBlock.CacheControl = anthropic.NewCacheControlEphemeralParam()
					}
					systemBlocks = append(systemBlocks, textBlock)
				}
			}

		case fantasy.MessageRoleUser:
			var anthropicContent []anthropic.ContentBlockParamUnion
			for _, msg := range block.Messages {
				if msg.Role == fantasy.MessageRoleUser {
					for i, part := range msg.Content {
						isLastPart := i == len(msg.Content)-1
						cacheControl := GetCacheControl(part.Options())
						if cacheControl == nil && isLastPart {
							cacheControl = GetCacheControl(msg.ProviderOptions)
						}
						switch part.GetType() {
						case fantasy.ContentTypeText:
							text, ok := fantasy.AsMessagePart[fantasy.TextPart](part)
							if !ok {
								continue
							}
							textBlock := &anthropic.TextBlockParam{
								Text: text.Text,
							}
							if cacheControl != nil {
								textBlock.CacheControl = anthropic.NewCacheControlEphemeralParam()
							}
							anthropicContent = append(anthropicContent, anthropic.ContentBlockParamUnion{
								OfText: textBlock,
							})
						case fantasy.ContentTypeFile:
							file, ok := fantasy.AsMessagePart[fantasy.FilePart](part)
							if !ok {
								continue
							}
							// TODO: handle other file types
							if !strings.HasPrefix(file.MediaType, "image/") {
								continue
							}

							base64Encoded := base64.StdEncoding.EncodeToString(file.Data)
							imageBlock := anthropic.NewImageBlockBase64(file.MediaType, base64Encoded)
							if cacheControl != nil {
								imageBlock.OfImage.CacheControl = anthropic.NewCacheControlEphemeralParam()
							}
							anthropicContent = append(anthropicContent, imageBlock)
						}
					}
				} else if msg.Role == fantasy.MessageRoleTool {
					for i, part := range msg.Content {
						isLastPart := i == len(msg.Content)-1
						cacheControl := GetCacheControl(part.Options())
						if cacheControl == nil && isLastPart {
							cacheControl = GetCacheControl(msg.ProviderOptions)
						}
						result, ok := fantasy.AsMessagePart[fantasy.ToolResultPart](part)
						if !ok {
							continue
						}
						toolResultBlock := anthropic.ToolResultBlockParam{
							ToolUseID: result.ToolCallID,
						}
						switch result.Output.GetType() {
						case fantasy.ToolResultContentTypeText:
							content, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentText](result.Output)
							if !ok {
								continue
							}
							toolResultBlock.Content = []anthropic.ToolResultBlockParamContentUnion{
								{
									OfText: &anthropic.TextBlockParam{
										Text: content.Text,
									},
								},
							}
						case fantasy.ToolResultContentTypeMedia:
							content, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentMedia](result.Output)
							if !ok {
								continue
							}
							contentBlocks := []anthropic.ToolResultBlockParamContentUnion{
								{
									OfImage: anthropic.NewImageBlockBase64(content.MediaType, content.Data).OfImage,
								},
							}
							if content.Text != "" {
								contentBlocks = append(contentBlocks, anthropic.ToolResultBlockParamContentUnion{
									OfText: &anthropic.TextBlockParam{
										Text: content.Text,
									},
								})
							}
							toolResultBlock.Content = contentBlocks
						case fantasy.ToolResultContentTypeError:
							content, ok := fantasy.AsToolResultOutputType[fantasy.ToolResultOutputContentError](result.Output)
							if !ok {
								continue
							}
							toolResultBlock.Content = []anthropic.ToolResultBlockParamContentUnion{
								{
									OfText: &anthropic.TextBlockParam{
										Text: content.Error.Error(),
									},
								},
							}
							toolResultBlock.IsError = param.NewOpt(true)
						}
						if cacheControl != nil {
							toolResultBlock.CacheControl = anthropic.NewCacheControlEphemeralParam()
						}
						anthropicContent = append(anthropicContent, anthropic.ContentBlockParamUnion{
							OfToolResult: &toolResultBlock,
						})
					}
				}
			}
			if !hasVisibleUserContent(anthropicContent) {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "dropping empty user message (contains neither user-facing content nor tool results)",
				})
				continue
			}
			messages = append(messages, anthropic.NewUserMessage(anthropicContent...))
		case fantasy.MessageRoleAssistant:
			var anthropicContent []anthropic.ContentBlockParamUnion
			for _, msg := range block.Messages {
				for i, part := range msg.Content {
					isLastPart := i == len(msg.Content)-1
					cacheControl := GetCacheControl(part.Options())
					if cacheControl == nil && isLastPart {
						cacheControl = GetCacheControl(msg.ProviderOptions)
					}
					switch part.GetType() {
					case fantasy.ContentTypeText:
						text, ok := fantasy.AsMessagePart[fantasy.TextPart](part)
						if !ok {
							continue
						}
						textBlock := &anthropic.TextBlockParam{
							Text: text.Text,
						}
						if cacheControl != nil {
							textBlock.CacheControl = anthropic.NewCacheControlEphemeralParam()
						}
						anthropicContent = append(anthropicContent, anthropic.ContentBlockParamUnion{
							OfText: textBlock,
						})
					case fantasy.ContentTypeReasoning:
						reasoning, ok := fantasy.AsMessagePart[fantasy.ReasoningPart](part)
						if !ok {
							continue
						}
						if !sendReasoningData {
							warnings = append(warnings, fantasy.CallWarning{
								Type:    fantasy.CallWarningTypeOther,
								Message: "sending reasoning content is disabled for this model",
							})
							continue
						}
						reasoningMetadata := GetReasoningMetadata(part.Options())
						if reasoningMetadata == nil {
							warnings = append(warnings, fantasy.CallWarning{
								Type:    fantasy.CallWarningTypeOther,
								Message: "unsupported reasoning metadata",
							})
							continue
						}

						if reasoningMetadata.Signature != "" {
							anthropicContent = append(anthropicContent, anthropic.NewThinkingBlock(reasoningMetadata.Signature, reasoning.Text))
						} else if reasoningMetadata.RedactedData != "" {
							anthropicContent = append(anthropicContent, anthropic.NewRedactedThinkingBlock(reasoningMetadata.RedactedData))
						} else {
							warnings = append(warnings, fantasy.CallWarning{
								Type:    fantasy.CallWarningTypeOther,
								Message: "unsupported reasoning metadata",
							})
							continue
						}
					case fantasy.ContentTypeToolCall:
						toolCall, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](part)
						if !ok {
							continue
						}
						if toolCall.ProviderExecuted {
							// TODO: implement provider executed call
							continue
						}

						var inputMap map[string]any
						err := json.Unmarshal([]byte(toolCall.Input), &inputMap)
						if err != nil {
							continue
						}
						toolUseBlock := anthropic.NewToolUseBlock(toolCall.ToolCallID, inputMap, toolCall.ToolName)
						if cacheControl != nil {
							toolUseBlock.OfToolUse.CacheControl = anthropic.NewCacheControlEphemeralParam()
						}
						anthropicContent = append(anthropicContent, toolUseBlock)
					case fantasy.ContentTypeToolResult:
						// TODO: implement provider executed tool result
					}
				}
			}

			if !hasVisibleAssistantContent(anthropicContent) {
				warnings = append(warnings, fantasy.CallWarning{
					Type:    fantasy.CallWarningTypeOther,
					Message: "dropping empty assistant message (contains neither user-facing content nor tool calls)",
				})
				continue
			}
			messages = append(messages, anthropic.NewAssistantMessage(anthropicContent...))
		}
	}
	return systemBlocks, messages, warnings
}

func hasVisibleUserContent(content []anthropic.ContentBlockParamUnion) bool {
	for _, block := range content {
		if block.OfText != nil || block.OfImage != nil || block.OfToolResult != nil {
			return true
		}
	}
	return false
}

func hasVisibleAssistantContent(content []anthropic.ContentBlockParamUnion) bool {
	for _, block := range content {
		if block.OfText != nil || block.OfToolUse != nil {
			return true
		}
	}
	return false
}

func mapFinishReason(finishReason string) fantasy.FinishReason {
	switch finishReason {
	case "end_turn", "pause_turn", "stop_sequence":
		return fantasy.FinishReasonStop
	case "max_tokens":
		return fantasy.FinishReasonLength
	case "tool_use":
		return fantasy.FinishReasonToolCalls
	default:
		return fantasy.FinishReasonUnknown
	}
}

// Generate implements fantasy.LanguageModel.
func (a languageModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	params, warnings, err := a.prepareParams(call)
	if err != nil {
		return nil, err
	}
	response, err := a.client.Messages.New(ctx, *params)
	if err != nil {
		return nil, toProviderErr(err)
	}

	var content []fantasy.Content
	for _, block := range response.Content {
		switch block.Type {
		case "text":
			text, ok := block.AsAny().(anthropic.TextBlock)
			if !ok {
				continue
			}
			content = append(content, fantasy.TextContent{
				Text: text.Text,
			})
		case "thinking":
			reasoning, ok := block.AsAny().(anthropic.ThinkingBlock)
			if !ok {
				continue
			}
			content = append(content, fantasy.ReasoningContent{
				Text: reasoning.Thinking,
				ProviderMetadata: fantasy.ProviderMetadata{
					Name: &ReasoningOptionMetadata{
						Signature: reasoning.Signature,
					},
				},
			})
		case "redacted_thinking":
			reasoning, ok := block.AsAny().(anthropic.RedactedThinkingBlock)
			if !ok {
				continue
			}
			content = append(content, fantasy.ReasoningContent{
				Text: "",
				ProviderMetadata: fantasy.ProviderMetadata{
					Name: &ReasoningOptionMetadata{
						RedactedData: reasoning.Data,
					},
				},
			})
		case "tool_use":
			toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}
			content = append(content, fantasy.ToolCallContent{
				ToolCallID:       toolUse.ID,
				ToolName:         toolUse.Name,
				Input:            string(toolUse.Input),
				ProviderExecuted: false,
			})
		}
	}

	return &fantasy.Response{
		Content: content,
		Usage: fantasy.Usage{
			InputTokens:         response.Usage.InputTokens,
			OutputTokens:        response.Usage.OutputTokens,
			TotalTokens:         response.Usage.InputTokens + response.Usage.OutputTokens,
			CacheCreationTokens: response.Usage.CacheCreationInputTokens,
			CacheReadTokens:     response.Usage.CacheReadInputTokens,
		},
		FinishReason:     mapFinishReason(string(response.StopReason)),
		ProviderMetadata: fantasy.ProviderMetadata{},
		Warnings:         warnings,
	}, nil
}

// Stream implements fantasy.LanguageModel.
func (a languageModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	params, warnings, err := a.prepareParams(call)
	if err != nil {
		return nil, err
	}

	stream := a.client.Messages.NewStreaming(ctx, *params)
	acc := anthropic.Message{}
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
			chunk := stream.Current()
			_ = acc.Accumulate(chunk)
			switch chunk.Type {
			case "content_block_start":
				contentBlockType := chunk.ContentBlock.Type
				switch contentBlockType {
				case "text":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeTextStart,
						ID:   fmt.Sprintf("%d", chunk.Index),
					}) {
						return
					}
				case "thinking":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeReasoningStart,
						ID:   fmt.Sprintf("%d", chunk.Index),
					}) {
						return
					}
				case "redacted_thinking":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeReasoningStart,
						ID:   fmt.Sprintf("%d", chunk.Index),
						ProviderMetadata: fantasy.ProviderMetadata{
							Name: &ReasoningOptionMetadata{
								RedactedData: chunk.ContentBlock.Data,
							},
						},
					}) {
						return
					}
				case "tool_use":
					if !yield(fantasy.StreamPart{
						Type:          fantasy.StreamPartTypeToolInputStart,
						ID:            chunk.ContentBlock.ID,
						ToolCallName:  chunk.ContentBlock.Name,
						ToolCallInput: "",
					}) {
						return
					}
				}
			case "content_block_stop":
				if len(acc.Content)-1 < int(chunk.Index) {
					continue
				}
				contentBlock := acc.Content[int(chunk.Index)]
				switch contentBlock.Type {
				case "text":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeTextEnd,
						ID:   fmt.Sprintf("%d", chunk.Index),
					}) {
						return
					}
				case "thinking":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeReasoningEnd,
						ID:   fmt.Sprintf("%d", chunk.Index),
					}) {
						return
					}
				case "tool_use":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeToolInputEnd,
						ID:   contentBlock.ID,
					}) {
						return
					}
					if !yield(fantasy.StreamPart{
						Type:          fantasy.StreamPartTypeToolCall,
						ID:            contentBlock.ID,
						ToolCallName:  contentBlock.Name,
						ToolCallInput: string(contentBlock.Input),
					}) {
						return
					}
				}
			case "content_block_delta":
				switch chunk.Delta.Type {
				case "text_delta":
					if !yield(fantasy.StreamPart{
						Type:  fantasy.StreamPartTypeTextDelta,
						ID:    fmt.Sprintf("%d", chunk.Index),
						Delta: chunk.Delta.Text,
					}) {
						return
					}
				case "thinking_delta":
					if !yield(fantasy.StreamPart{
						Type:  fantasy.StreamPartTypeReasoningDelta,
						ID:    fmt.Sprintf("%d", chunk.Index),
						Delta: chunk.Delta.Thinking,
					}) {
						return
					}
				case "signature_delta":
					if !yield(fantasy.StreamPart{
						Type: fantasy.StreamPartTypeReasoningDelta,
						ID:   fmt.Sprintf("%d", chunk.Index),
						ProviderMetadata: fantasy.ProviderMetadata{
							Name: &ReasoningOptionMetadata{
								Signature: chunk.Delta.Signature,
							},
						},
					}) {
						return
					}
				case "input_json_delta":
					if len(acc.Content)-1 < int(chunk.Index) {
						continue
					}
					contentBlock := acc.Content[int(chunk.Index)]
					if !yield(fantasy.StreamPart{
						Type:          fantasy.StreamPartTypeToolInputDelta,
						ID:            contentBlock.ID,
						ToolCallInput: chunk.Delta.PartialJSON,
					}) {
						return
					}
				}
			case "message_stop":
			}
		}

		err := stream.Err()
		if err == nil || errors.Is(err, io.EOF) {
			yield(fantasy.StreamPart{
				Type:         fantasy.StreamPartTypeFinish,
				ID:           acc.ID,
				FinishReason: mapFinishReason(string(acc.StopReason)),
				Usage: fantasy.Usage{
					InputTokens:         acc.Usage.InputTokens,
					OutputTokens:        acc.Usage.OutputTokens,
					TotalTokens:         acc.Usage.InputTokens + acc.Usage.OutputTokens,
					CacheCreationTokens: acc.Usage.CacheCreationInputTokens,
					CacheReadTokens:     acc.Usage.CacheReadInputTokens,
				},
				ProviderMetadata: fantasy.ProviderMetadata{},
			})
			return
		} else { //nolint: revive
			yield(fantasy.StreamPart{
				Type:  fantasy.StreamPartTypeError,
				Error: toProviderErr(err),
			})
			return
		}
	}, nil
}

// GenerateObject implements fantasy.LanguageModel.
func (a languageModel) GenerateObject(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	switch a.options.objectMode {
	case fantasy.ObjectModeText:
		return object.GenerateWithText(ctx, a, call)
	default:
		return object.GenerateWithTool(ctx, a, call)
	}
}

// StreamObject implements fantasy.LanguageModel.
func (a languageModel) StreamObject(ctx context.Context, call fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	switch a.options.objectMode {
	case fantasy.ObjectModeText:
		return object.StreamWithText(ctx, a, call)
	default:
		return object.StreamWithTool(ctx, a, call)
	}
}
