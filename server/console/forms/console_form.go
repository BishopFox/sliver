package forms

import (
	"errors"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func newConsoleForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...)
}

func runConsoleForm(form *huh.Form) error {
	if form == nil {
		return errors.New("form is required")
	}

	ttyIn, ttyOut, err := tea.OpenTTY()
	if err == nil {
		if ttyIn != nil {
			defer ttyIn.Close()
		}
		if ttyOut != nil && (ttyIn == nil || ttyOut.Fd() != ttyIn.Fd()) {
			defer ttyOut.Close()
		}
		if ttyIn != nil {
			form.WithInput(ttyIn)
		}
		if ttyOut != nil {
			form.WithOutput(ttyOut)
		}
	}

	return form.Run()
}
