package forms

import (
	"errors"

	"github.com/charmbracelet/huh"
)

// ArmoryInstallOption represents an armory install option.
type ArmoryInstallOption struct {
	Value string
	Label string
}

// ArmoryInstallFormResult captures the selection from the armory install form.
type ArmoryInstallFormResult struct {
	SelectedNames []string
}

// ArmoryInstallForm prompts for armory packages or bundles to install.
func ArmoryInstallForm(options []ArmoryInstallOption) (*ArmoryInstallFormResult, error) {
	if len(options) == 0 {
		return nil, errors.New("armory install options are required")
	}

	selectOptions := make([]huh.Option[string], 0, len(options))
	for _, option := range options {
		if option.Value == "" {
			return nil, errors.New("armory install option value is required")
		}
		if option.Label == "" {
			return nil, errors.New("armory install option label is required")
		}
		selectOptions = append(selectOptions, huh.NewOption(option.Label, option.Value))
	}

	result := &ArmoryInstallFormResult{}

	field := huh.NewMultiSelect[string]().
		Title("Select packages or bundles to install").
		Description("Use space to select and enter to install.").
		Options(selectOptions...).
		Height(listHeight(len(selectOptions))).
		Value(&result.SelectedNames)

	form := huh.NewForm(huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}
