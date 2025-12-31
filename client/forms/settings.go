package forms

import (
	"errors"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/charmbracelet/huh"
)

// SettingsFormResult captures the inputs needed to update client settings.
type SettingsFormResult struct {
	TableStyle        string
	AutoAdult         bool
	BeaconAutoResults bool
	SmallTermWidth    string
	AlwaysOverflow    bool
	VimMode           bool
	UserConnect       bool
	ConsoleLogs       bool
}

// SettingsForm prompts for client settings and returns the collected values.
func SettingsForm(settings *assets.ClientSettings, tableStyleOptions []string) (*SettingsFormResult, error) {
	if settings == nil {
		return nil, errors.New("settings are required")
	}
	if len(tableStyleOptions) == 0 {
		return nil, errors.New("no table styles available")
	}

	result := &SettingsFormResult{
		TableStyle:        settings.TableStyle,
		AutoAdult:         settings.AutoAdult,
		BeaconAutoResults: settings.BeaconAutoResults,
		SmallTermWidth:    strconv.Itoa(settings.SmallTermWidth),
		AlwaysOverflow:    settings.AlwaysOverflow,
		VimMode:           settings.VimMode,
		UserConnect:       settings.UserConnect,
		ConsoleLogs:       settings.ConsoleLogs,
	}
	if result.TableStyle == "" {
		result.TableStyle = tableStyleOptions[0]
	}

	styleOptions := make([]huh.Option[string], 0, len(tableStyleOptions))
	for _, option := range tableStyleOptions {
		styleOptions = append(styleOptions, huh.NewOption(option, option))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Table style").
				Options(styleOptions...).
				Height(len(styleOptions)).
				Value(&result.TableStyle),
			huh.NewInput().
				Title("Small terminal width").
				Description("Omit some table columns when the terminal is narrower than this value").
				Value(&result.SmallTermWidth).
				Validate(validateRequiredPositiveInt),
			huh.NewConfirm().
				Title("Always overflow tables").
				Description("Disable table pagination").
				Value(&result.AlwaysOverflow),
		).Title("Display"),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Auto accept OPSEC warnings").
				Description("Skip confirmation prompts for risky actions").
				Value(&result.AutoAdult),
			huh.NewConfirm().
				Title("Auto display beacon results").
				Description("Show beacon task results when tasks complete").
				Value(&result.BeaconAutoResults),
			huh.NewConfirm().
				Title("Vim navigation mode").
				Description("Use vim-style navigation in the console").
				Value(&result.VimMode),
			huh.NewConfirm().
				Title("User connect events").
				Description("Show operator connect and disconnect events").
				Value(&result.UserConnect),
			huh.NewConfirm().
				Title("Console logs").
				Description("Log console output to disk").
				Value(&result.ConsoleLogs),
		).Title("Behavior"),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}

func validateRequiredPositiveInt(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("value required")
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return errors.New("must be a whole number")
	}
	if parsed < 1 {
		return errors.New("must be 1 or greater")
	}
	return nil
}
