// Package openaicompat provides an implementation of the fantasy AI SDK for OpenAI-compatible APIs.
package openaicompat

import (
	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"github.com/openai/openai-go/v2/option"
)

type options struct {
	openaiOptions        []openai.Option
	languageModelOptions []openai.LanguageModelOption
	sdkOptions           []option.RequestOption
	objectMode           fantasy.ObjectMode
}

const (
	// Name is the name of the OpenAI-compatible provider.
	Name = "openai-compat"
)

// Option defines a function that configures OpenAI-compatible provider options.
type Option = func(*options)

// New creates a new OpenAI-compatible provider with the given options.
func New(opts ...Option) (fantasy.Provider, error) {
	providerOptions := options{
		openaiOptions: []openai.Option{
			openai.WithName(Name),
		},
		languageModelOptions: []openai.LanguageModelOption{
			openai.WithLanguageModelPrepareCallFunc(PrepareCallFunc),
			openai.WithLanguageModelStreamExtraFunc(StreamExtraFunc),
			openai.WithLanguageModelExtraContentFunc(ExtraContentFunc),
			openai.WithLanguageModelToPromptFunc(ToPromptFunc),
		},
		objectMode: fantasy.ObjectModeTool, // Default to tool mode for openai-compat
	}
	for _, o := range opts {
		o(&providerOptions)
	}

	// Handle object mode: convert unsupported modes to tool
	// OpenAI-compat endpoints don't support native JSON mode, so we use tool or text
	objectMode := providerOptions.objectMode
	if objectMode == fantasy.ObjectModeAuto || objectMode == fantasy.ObjectModeJSON {
		objectMode = fantasy.ObjectModeTool
	}

	providerOptions.openaiOptions = append(
		providerOptions.openaiOptions,
		openai.WithSDKOptions(providerOptions.sdkOptions...),
		openai.WithLanguageModelOptions(providerOptions.languageModelOptions...),
		openai.WithObjectMode(objectMode),
	)
	return openai.New(providerOptions.openaiOptions...)
}

// WithBaseURL sets the base URL for the OpenAI-compatible provider.
func WithBaseURL(url string) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithBaseURL(url))
	}
}

// WithAPIKey sets the API key for the OpenAI-compatible provider.
func WithAPIKey(apiKey string) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithAPIKey(apiKey))
	}
}

// WithName sets the name for the OpenAI-compatible provider.
func WithName(name string) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithName(name))
	}
}

// WithHeaders sets the headers for the OpenAI-compatible provider.
func WithHeaders(headers map[string]string) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithHeaders(headers))
	}
}

// WithHTTPClient sets the HTTP client for the OpenAI-compatible provider.
func WithHTTPClient(client option.HTTPClient) Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithHTTPClient(client))
	}
}

// WithSDKOptions sets the SDK options for the OpenAI-compatible provider.
func WithSDKOptions(opts ...option.RequestOption) Option {
	return func(o *options) {
		o.sdkOptions = append(o.sdkOptions, opts...)
	}
}

// WithObjectMode sets the object generation mode for the OpenAI-compatible provider.
// Supported modes: ObjectModeTool, ObjectModeText.
// ObjectModeAuto and ObjectModeJSON are automatically converted to ObjectModeTool
// since OpenAI-compatible endpoints typically don't support native JSON mode.
func WithObjectMode(om fantasy.ObjectMode) Option {
	return func(o *options) {
		o.objectMode = om
	}
}

// WithUseResponsesAPI configures the provider to use the responses API for models that support it.
func WithUseResponsesAPI() Option {
	return func(o *options) {
		o.openaiOptions = append(o.openaiOptions, openai.WithUseResponsesAPI())
	}
}
