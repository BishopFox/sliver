package theme

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// HuhTheme returns a huh Theme derived from the current Sliver client theme.
//
// This is used for all interactive TUI forms in the client so that colors are
// consistent with the rest of the console styling (theme.yaml).
func HuhTheme() *huh.Theme {
	t := huh.ThemeBase()

	border := DefaultMod(300)
	text := DefaultMod(900)
	dim := DefaultMod(500)
	faint := DefaultMod(400)

	primary := Primary()
	accent := Secondary()

	success := Success()
	danger := Danger()

	// Focused styles.
	t.Focused.Base = t.Focused.Base.BorderForeground(border)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(primary).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(primary).Bold(true).MarginBottom(1)
	t.Focused.Description = t.Focused.Description.Foreground(dim)
	t.Focused.Directory = t.Focused.Directory.Foreground(primary)
	t.Focused.File = t.Focused.File.Foreground(text)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(danger)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(danger)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(accent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(accent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(accent)
	t.Focused.Option = t.Focused.Option.Foreground(text)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(accent)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(success)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(success)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(text)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(faint)

	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(text).Background(primary)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(text).Background(lipgloss.Color(DefaultMod(100)))

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(primary)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(faint)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(accent)

	// Blurred styles (same palette, but without the focused border/indicators).
	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.MultiSelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	// Group styles.
	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}
