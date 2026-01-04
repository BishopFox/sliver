package forms

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"
)

const defaultSelectHeight = 10

// ErrUserAborted exposes the huh user-abort sentinel for callers.
var ErrUserAborted = huh.ErrUserAborted

// SelectOperator prompts for a single operator selection.
func SelectOperator(title string, operators []string, value *string) error {
	if value == nil {
		return errors.New("operator selection value is required")
	}
	if len(operators) == 0 {
		return errors.New("operator selection options are required")
	}

	ensureSelectedValue(operators, value)

	field := huh.NewSelect[string]().
		Title(title).
		Options(makeStringOptions(operators)...).
		Height(listHeight(len(operators))).
		Value(value).
		Validate(func(val string) error {
			if strings.TrimSpace(val) == "" {
				return errors.New("selection required")
			}
			return nil
		})

	form := huh.NewForm(huh.NewGroup(field))
	return form.Run()
}

func ensureSelectedValue(options []string, value *string) {
	if value == nil || len(options) == 0 {
		return
	}
	if strings.TrimSpace(*value) == "" {
		*value = options[0]
		return
	}
	for _, option := range options {
		if option == *value {
			return
		}
	}
	*value = options[0]
}

func makeStringOptions(options []string) []huh.Option[string] {
	converted := make([]huh.Option[string], 0, len(options))
	for _, option := range options {
		converted = append(converted, huh.NewOption(option, option))
	}
	return converted
}

func listHeight(count int) int {
	if count < 1 {
		return 1
	}
	if count > defaultSelectHeight {
		return defaultSelectHeight
	}
	return count
}
