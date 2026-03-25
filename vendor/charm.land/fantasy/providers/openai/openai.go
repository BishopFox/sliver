// Package openai provides an implementation of the fantasy AI SDK for OpenAI's language models.
package openai

import (
	"cmp"
	"context"
	"maps"

	"charm.land/fantasy"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

const (
	// Name is the name of the OpenAI provider.
	Name = "openai"
	// DefaultURL is the default URL for the OpenAI API.
	DefaultURL = "https://api.openai.com/v1"
)

type provider struct {
	options options
}

type options struct {
	baseURL              string
	apiKey               string
	organization         string
	project              string
	name                 string
	useResponsesAPI      bool
	headers              map[string]string
	client               option.HTTPClient
	sdkOptions           []option.RequestOption
	objectMode           fantasy.ObjectMode
	languageModelOptions []LanguageModelOption
}

// Option defines a function that configures OpenAI provider options.
type Option = func(*options)

// New creates a new OpenAI provider with the given options.
func New(opts ...Option) (fantasy.Provider, error) {
	providerOptions := options{
		headers:              map[string]string{},
		languageModelOptions: make([]LanguageModelOption, 0),
	}
	for _, o := range opts {
		o(&providerOptions)
	}

	providerOptions.baseURL = cmp.Or(providerOptions.baseURL, DefaultURL)
	providerOptions.name = cmp.Or(providerOptions.name, Name)

	if providerOptions.organization != "" {
		providerOptions.headers["OpenAi-Organization"] = providerOptions.organization
	}
	if providerOptions.project != "" {
		providerOptions.headers["OpenAi-Project"] = providerOptions.project
	}

	return &provider{options: providerOptions}, nil
}

// WithBaseURL sets the base URL for the OpenAI provider.
func WithBaseURL(baseURL string) Option {
	return func(o *options) {
		o.baseURL = baseURL
	}
}

// WithAPIKey sets the API key for the OpenAI provider.
func WithAPIKey(apiKey string) Option {
	return func(o *options) {
		o.apiKey = apiKey
	}
}

// WithOrganization sets the organization for the OpenAI provider.
func WithOrganization(organization string) Option {
	return func(o *options) {
		o.organization = organization
	}
}

// WithProject sets the project for the OpenAI provider.
func WithProject(project string) Option {
	return func(o *options) {
		o.project = project
	}
}

// WithName sets the name for the OpenAI provider.
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithHeaders sets the headers for the OpenAI provider.
func WithHeaders(headers map[string]string) Option {
	return func(o *options) {
		maps.Copy(o.headers, headers)
	}
}

// WithHTTPClient sets the HTTP client for the OpenAI provider.
func WithHTTPClient(client option.HTTPClient) Option {
	return func(o *options) {
		o.client = client
	}
}

// WithSDKOptions sets the SDK options for the OpenAI provider.
func WithSDKOptions(opts ...option.RequestOption) Option {
	return func(o *options) {
		o.sdkOptions = append(o.sdkOptions, opts...)
	}
}

// WithLanguageModelOptions sets the language model options for the OpenAI provider.
func WithLanguageModelOptions(opts ...LanguageModelOption) Option {
	return func(o *options) {
		o.languageModelOptions = append(o.languageModelOptions, opts...)
	}
}

// WithUseResponsesAPI configures the provider to use the responses API for models that support it.
func WithUseResponsesAPI() Option {
	return func(o *options) {
		o.useResponsesAPI = true
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

// LanguageModel implements fantasy.Provider.
func (o *provider) LanguageModel(_ context.Context, modelID string) (fantasy.LanguageModel, error) {
	openaiClientOptions := make([]option.RequestOption, 0, 5+len(o.options.headers)+len(o.options.sdkOptions))
	openaiClientOptions = append(openaiClientOptions, option.WithMaxRetries(0))

	if o.options.apiKey != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithAPIKey(o.options.apiKey))
	}
	if o.options.baseURL != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithBaseURL(o.options.baseURL))
	}

	for key, value := range o.options.headers {
		openaiClientOptions = append(openaiClientOptions, option.WithHeader(key, value))
	}

	if o.options.client != nil {
		openaiClientOptions = append(openaiClientOptions, option.WithHTTPClient(o.options.client))
	}

	openaiClientOptions = append(openaiClientOptions, o.options.sdkOptions...)

	client := openai.NewClient(openaiClientOptions...)

	if o.options.useResponsesAPI && IsResponsesModel(modelID) {
		// Not supported for responses API
		objectMode := o.options.objectMode
		if objectMode == fantasy.ObjectModeJSON {
			objectMode = fantasy.ObjectModeAuto
		}
		return newResponsesLanguageModel(modelID, o.options.name, client, objectMode), nil
	}

	o.options.languageModelOptions = append(o.options.languageModelOptions, WithLanguageModelObjectMode(o.options.objectMode))

	return newLanguageModel(
		modelID,
		o.options.name,
		client,
		o.options.languageModelOptions...,
	), nil
}

func (o *provider) Name() string {
	return Name
}
