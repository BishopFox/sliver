package forms

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"github.com/bishopfox/sliver/server/ai"
)

// AIConfigFormResult captures the server-side AI configuration collected from the form.
type AIConfigFormResult struct {
	Provider        string
	Model           string
	ThinkingLevel   string
	SystemPrompt    string
	APIKey          string
	BaseURL         string
	UserAgent       string
	Organization    string
	Project         string
	Location        string
	UseResponsesAPI bool
	SkipAuth        bool
	UseBedrock      bool
}

// AIConfig prompts for the server-side AI configuration stored in ai.yaml.
func AIConfig(result *AIConfigFormResult) error {
	if result == nil {
		return errors.New("AI config result is required")
	}

	if err := SelectAIProvider(result); err != nil {
		return err
	}
	return EditAIConfig(result)
}

// SelectAIProvider prompts for the AI provider to edit.
func SelectAIProvider(result *AIConfigFormResult) error {
	if result == nil {
		return errors.New("AI config result is required")
	}

	normalizeAIConfigResult(result)

	providerOptions := []huh.Option[string]{
		huh.NewOption("Anthropic", ai.ProviderAnthropic),
		huh.NewOption("Google (Gemini / Vertex)", ai.ProviderGoogle),
		huh.NewOption("OpenAI", ai.ProviderOpenAI),
		huh.NewOption("OpenAI-Compatible", ai.ProviderOpenAICompat),
		huh.NewOption("OpenRouter", ai.ProviderOpenRouter),
	}

	providerForm := newConsoleForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AI provider").
				Description("Choose the default AI provider stored in ai.yaml.").
				Options(providerOptions...).
				Height(listHeight(len(providerOptions))).
				Value(&result.Provider).
				Validate(func(val string) error {
					if !ai.IsSupportedProvider(val) {
						return errors.New("select a supported provider")
					}
					return nil
				}),
		),
	)
	if err := runConsoleForm(providerForm); err != nil {
		return err
	}

	normalizeAIConfigResult(result)
	return nil
}

// EditAIConfig prompts for the provider-independent and provider-specific AI settings.
func EditAIConfig(result *AIConfigFormResult) error {
	if result == nil {
		return errors.New("AI config result is required")
	}

	normalizeAIConfigResult(result)

	thinkingOptions := []huh.Option[string]{
		huh.NewOption("Provider default", ""),
		huh.NewOption("Disabled", "disabled"),
		huh.NewOption("Low", "low"),
		huh.NewOption("Medium", "medium"),
		huh.NewOption("High", "high"),
	}

	detailsForm := newConsoleForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Server AI configuration").
				DescriptionFunc(func() string {
					return fmt.Sprintf(
						"Updating the %q provider. Re-run this command to edit credentials for a different provider.",
						providerDisplayName(result.Provider),
					)
				}, &result.Provider),
			huh.NewInput().
				Title("Model").
				Description("Optional default model identifier. Leave blank to let the provider choose.").
				Placeholder("provider default").
				Value(&result.Model),
			huh.NewSelect[string]().
				Title("Thinking level").
				Description("Optional reasoning/thinking level to store in ai.yaml.").
				Options(thinkingOptions...).
				Height(listHeight(len(thinkingOptions))).
				Value(&result.ThinkingLevel),
			huh.NewInput().
				Title("System prompt").
				Description("Optional default system prompt for new AI conversations.").
				Placeholder("leave blank for no default system prompt").
				Value(&result.SystemPrompt),
			huh.NewInput().
				TitleFunc(func() string {
					return fmt.Sprintf("%s API key", providerDisplayName(result.Provider))
				}, &result.Provider).
				DescriptionFunc(func() string {
					return providerAPIKeyDescription(result.Provider)
				}, &result.Provider).
				EchoMode(huh.EchoModePassword).
				Value(&result.APIKey),
			huh.NewInput().
				TitleFunc(func() string {
					return fmt.Sprintf("%s base URL", providerDisplayName(result.Provider))
				}, &result.Provider).
				DescriptionFunc(func() string {
					return providerBaseURLDescription(result.Provider)
				}, &result.Provider).
				Placeholder("provider default").
				Value(&result.BaseURL),
			huh.NewInput().
				TitleFunc(func() string {
					return fmt.Sprintf("%s user agent", providerDisplayName(result.Provider))
				}, &result.Provider).
				Description("Optional custom User-Agent header override.").
				Placeholder("library default").
				Value(&result.UserAgent),
		),
	)
	if err := runConsoleForm(detailsForm); err != nil {
		return err
	}

	if err := providerSpecificAIConfig(result); err != nil {
		return err
	}

	normalizeAIConfigResult(result)
	return nil
}

func providerSpecificAIConfig(result *AIConfigFormResult) error {
	switch ai.NormalizeProviderName(result.Provider) {
	case ai.ProviderAnthropic:
		return anthropicAIConfig(result)
	case ai.ProviderGoogle:
		return googleAIConfig(result)
	case ai.ProviderOpenAI:
		return openAIAIConfig(result)
	default:
		return nil
	}
}

func openAIAIConfig(result *AIConfigFormResult) error {
	form := newConsoleForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Organization").
				Description("Optional OpenAI organization header.").
				Placeholder("org_...").
				Value(&result.Organization),
			huh.NewInput().
				Title("Project").
				Description("Optional OpenAI project header.").
				Placeholder("proj_...").
				Value(&result.Project),
			huh.NewConfirm().
				Title("Use Responses API").
				Description("Enable the OpenAI Responses API for supported models.").
				Value(&result.UseResponsesAPI),
		),
	)
	return runConsoleForm(form)
}

func anthropicAIConfig(result *AIConfigFormResult) error {
	form := newConsoleForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Vertex project").
				Description("Optional Google Cloud project when routing Anthropic through Vertex AI.").
				Placeholder("my-project").
				Value(&result.Project),
			huh.NewInput().
				Title("Vertex location").
				Description("Optional Google Cloud region when routing Anthropic through Vertex AI.").
				Placeholder("us-central1").
				Value(&result.Location),
			huh.NewConfirm().
				Title("Skip cloud auth lookup").
				Description("Use provider-specific test or proxy auth instead of default cloud credentials lookup.").
				Value(&result.SkipAuth),
			huh.NewConfirm().
				Title("Use Bedrock").
				Description("Route Anthropic requests through AWS Bedrock instead of the direct Anthropic API.").
				Value(&result.UseBedrock),
		),
	)
	return runConsoleForm(form)
}

func googleAIConfig(result *AIConfigFormResult) error {
	form := newConsoleForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Vertex project").
				Description("Optional Google Cloud project when using Vertex AI instead of a Gemini API key.").
				Placeholder("my-project").
				Value(&result.Project),
			huh.NewInput().
				Title("Vertex location").
				Description("Optional Google Cloud region when using Vertex AI instead of a Gemini API key.").
				Placeholder("us-central1").
				Value(&result.Location),
			huh.NewConfirm().
				Title("Skip cloud auth lookup").
				Description("Use provider-specific test or proxy auth instead of default cloud credentials lookup.").
				Value(&result.SkipAuth),
		),
	)
	return runConsoleForm(form)
}

func normalizeAIConfigResult(result *AIConfigFormResult) {
	if result == nil {
		return
	}
	result.Provider = ai.NormalizeProviderName(result.Provider)
	if !ai.IsSupportedProvider(result.Provider) {
		result.Provider = ai.ProviderOpenAI
	}
	result.Model = strings.TrimSpace(result.Model)
	result.ThinkingLevel = strings.ToLower(strings.TrimSpace(result.ThinkingLevel))
	result.SystemPrompt = strings.TrimSpace(result.SystemPrompt)
	switch result.ThinkingLevel {
	case "", "disabled", "low", "medium", "high":
	default:
		result.ThinkingLevel = ""
	}
	result.APIKey = strings.TrimSpace(result.APIKey)
	result.BaseURL = strings.TrimSpace(result.BaseURL)
	result.UserAgent = strings.TrimSpace(result.UserAgent)
	result.Organization = strings.TrimSpace(result.Organization)
	result.Project = strings.TrimSpace(result.Project)
	result.Location = strings.TrimSpace(result.Location)

	switch result.Provider {
	case ai.ProviderOpenAI:
		result.SkipAuth = false
		result.UseBedrock = false
	case ai.ProviderAnthropic:
		result.Organization = ""
		result.UseResponsesAPI = false
	case ai.ProviderGoogle:
		result.Organization = ""
		result.UseResponsesAPI = false
		result.UseBedrock = false
	default:
		result.Organization = ""
		result.Project = ""
		result.Location = ""
		result.UseResponsesAPI = false
		result.SkipAuth = false
		result.UseBedrock = false
	}
}

func providerDisplayName(provider string) string {
	switch ai.NormalizeProviderName(provider) {
	case ai.ProviderAnthropic:
		return "Anthropic"
	case ai.ProviderGoogle:
		return "Google"
	case ai.ProviderOpenAI:
		return "OpenAI"
	case ai.ProviderOpenAICompat:
		return "OpenAI-Compatible"
	case ai.ProviderOpenRouter:
		return "OpenRouter"
	default:
		return "AI"
	}
}

func providerAPIKeyDescription(provider string) string {
	switch ai.NormalizeProviderName(provider) {
	case ai.ProviderAnthropic:
		return "Stored at ai.anthropic.api_key. Leave blank to clear it. Not required when using Bedrock or Vertex."
	case ai.ProviderGoogle:
		return "Stored at ai.google.api_key. Leave blank to clear it. Not required when using Vertex."
	case ai.ProviderOpenAICompat:
		return "Stored at ai.openai_compat.api_key. Leave blank to clear it for unauthenticated local endpoints."
	default:
		return fmt.Sprintf("Stored at ai.%s.api_key. Leave blank to clear it.", resultProviderKey(provider))
	}
}

func providerBaseURLDescription(provider string) string {
	switch ai.NormalizeProviderName(provider) {
	case ai.ProviderOpenAICompat:
		return "Required for OpenAI-compatible endpoints and stored at ai.openai_compat.base_url."
	default:
		return fmt.Sprintf("Optional override stored at ai.%s.base_url.", resultProviderKey(provider))
	}
}

func resultProviderKey(provider string) string {
	switch ai.NormalizeProviderName(provider) {
	case ai.ProviderOpenAICompat:
		return "openai_compat"
	default:
		return ai.NormalizeProviderName(provider)
	}
}
