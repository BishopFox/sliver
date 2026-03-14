package forms

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/server/ai"
	"github.com/charmbracelet/huh"
)

// AIConfigFormResult captures the server-side AI configuration collected from the form.
type AIConfigFormResult struct {
	Provider      string
	Model         string
	ThinkingLevel string
	APIKey        string
	BaseURL       string
}

// AIConfig prompts for the server-side AI configuration stored in server.yaml.
func AIConfig(result *AIConfigFormResult) error {
	if result == nil {
		return errors.New("AI config result is required")
	}

	normalizeAIConfigResult(result)

	providerOptions := []huh.Option[string]{
		huh.NewOption("Anthropic", ai.ProviderAnthropic),
		huh.NewOption("OpenAI", ai.ProviderOpenAI),
	}
	thinkingOptions := []huh.Option[string]{
		huh.NewOption("Provider default", ""),
		huh.NewOption("Disabled", "disabled"),
		huh.NewOption("Low", "low"),
		huh.NewOption("Medium", "medium"),
		huh.NewOption("High", "high"),
	}

	providerForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AI provider").
				Description("Choose the default AI provider stored in server.yaml.").
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
	if err := providerForm.Run(); err != nil {
		return err
	}

	detailsForm := huh.NewForm(
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
				Description("Optional reasoning/thinking level to store in the ai block.").
				Options(thinkingOptions...).
				Height(listHeight(len(thinkingOptions))).
				Value(&result.ThinkingLevel),
			huh.NewInput().
				TitleFunc(func() string {
					return fmt.Sprintf("%s API key", providerDisplayName(result.Provider))
				}, &result.Provider).
				DescriptionFunc(func() string {
					return fmt.Sprintf("Stored at ai.%s.api_key. Leave blank to clear it.", result.Provider)
				}, &result.Provider).
				EchoMode(huh.EchoModePassword).
				Value(&result.APIKey),
			huh.NewInput().
				TitleFunc(func() string {
					return fmt.Sprintf("%s base URL", providerDisplayName(result.Provider))
				}, &result.Provider).
				DescriptionFunc(func() string {
					return fmt.Sprintf("Optional override stored at ai.%s.base_url.", result.Provider)
				}, &result.Provider).
				Placeholder("provider default").
				Value(&result.BaseURL),
		),
	)
	if err := detailsForm.Run(); err != nil {
		return err
	}

	normalizeAIConfigResult(result)
	return nil
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
	switch result.ThinkingLevel {
	case "", "disabled", "low", "medium", "high":
	default:
		result.ThinkingLevel = ""
	}
	result.APIKey = strings.TrimSpace(result.APIKey)
	result.BaseURL = strings.TrimSpace(result.BaseURL)
}

func providerDisplayName(provider string) string {
	switch ai.NormalizeProviderName(provider) {
	case ai.ProviderAnthropic:
		return "Anthropic"
	case ai.ProviderOpenAI:
		return "OpenAI"
	default:
		return "AI"
	}
}
