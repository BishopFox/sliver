package forms

import (
	"errors"

	"github.com/charmbracelet/huh"
)

// ArmoryUpdateOption represents an update option in the armory update form.
type ArmoryUpdateOption struct {
	ID    string
	Label string
}

// ArmoryUpdateFormResult captures the selection from the armory update form.
type ArmoryUpdateFormResult struct {
	SelectedIDs []string
}

// ArmoryUpdateForm prompts for armory updates to apply.
func ArmoryUpdateForm(options []ArmoryUpdateOption) (*ArmoryUpdateFormResult, error) {
	if len(options) == 0 {
		return nil, errors.New("armory update options are required")
	}

	selectOptions := make([]huh.Option[string], 0, len(options))
	for _, option := range options {
		if option.ID == "" {
			return nil, errors.New("armory update option id is required")
		}
		if option.Label == "" {
			return nil, errors.New("armory update option label is required")
		}
		selectOptions = append(selectOptions, huh.NewOption(option.Label, option.ID))
	}

	result := &ArmoryUpdateFormResult{}
	field := huh.NewMultiSelect[string]().
		Title("Select updates to apply").
		Description("Use space to select and enter to apply.").
		Options(selectOptions...).
		Height(listHeight(len(selectOptions))).
		Value(&result.SelectedIDs)

	form := huh.NewForm(huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}
