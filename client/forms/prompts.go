package forms

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

const defaultSelectHeight = 10

func getTerminalWidth() int {
	// Try to get actual terminal width
	if width, _, err := term.GetSize(1); err == nil && width > 0 {
		return width
	}
	// Fall back to a reasonable default
	return 200
}

// Confirm prompts for a yes/no answer.
func Confirm(title string, value *bool) error {
	if value == nil {
		return errors.New("confirm value is required")
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Value(value),
		),
	)

	return form.Run()
}

// Input prompts for a single-line string.
func Input(title string, value *string) error {
	if value == nil {
		return errors.New("input value is required")
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Value(value),
		),
	)

	return form.Run()
}

// Text prompts for multi-line input.
func Text(title string, value *string) error {
	if value == nil {
		return errors.New("text value is required")
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title(title).
				ExternalEditor(false).
				Value(value),
		),
	)

	return form.Run()
}

// Select prompts for a single selection from options.
func Select(title string, options []string, value *string) error {
	return selectPrompt(title, options, value, false)
}

// SelectRequired prompts for a single selection and requires a non-empty value.
func SelectRequired(title string, options []string, value *string) error {
	return selectPrompt(title, options, value, true)
}

// MultiSelect prompts for selecting multiple values.
func MultiSelect(title string, options []string, value *[]string) error {
	if value == nil {
		return errors.New("multi-select value is required")
	}
	if len(options) == 0 {
		return errors.New("multi-select options are required")
	}

	field := huh.NewMultiSelect[string]().
		Title(title).
		Options(makeStringOptions(options)...).
		Height(listHeight(len(options))).
		Value(value)

	form := huh.NewForm(huh.NewGroup(field)).WithWidth(getTerminalWidth())
	return form.Run()
}

func selectPrompt(title string, options []string, value *string, required bool) error {
	if value == nil {
		return errors.New("select value is required")
	}
	if len(options) == 0 {
		return errors.New("select options are required")
	}

	// Save the original value in case the form is cancelled
	originalValue := *value
	ensureSelectedValue(options, value)

	field := huh.NewSelect[string]().
		Title(title).
		Options(makeStringOptions(options)...).
		Height(listHeight(len(options))).
		Value(value)

	if required {
		field.Validate(func(val string) error {
			if strings.TrimSpace(val) == "" {
				return errors.New("selection required")
			}
			return nil
		})
	}

	form := huh.NewForm(huh.NewGroup(field)).WithWidth(getTerminalWidth())
	err := form.Run()

	// On error restore the originalValue and return err
	if err != nil {
		*value = originalValue
		return err
	}

	return err
}

func ensureSelectedValue(options []string, value *string) {
	if value == nil || len(options) == 0 {
		return
	}
	if *value == "" {
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
