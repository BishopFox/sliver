package huh

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
	catppuccin "github.com/catppuccin/go"
)

// Theme represents a theme for a huh.
type Theme interface {
	Theme(isDark bool) *Styles
}

// ThemeFunc is a function that returns a new theme.
type ThemeFunc func(isDark bool) *Styles

// Theme implements the Theme interface.
func (f ThemeFunc) Theme(isDark bool) *Styles {
	return f(isDark)
}

// Styles is a collection of styles for components of the form.
// Themes can be applied to a form using the WithTheme option.
type Styles struct {
	Form           FormStyles
	Group          GroupStyles
	FieldSeparator lipgloss.Style
	Blurred        FieldStyles
	Focused        FieldStyles
	Help           help.Styles
}

// FormStyles are the styles for a form.
type FormStyles struct {
	Base lipgloss.Style
}

// GroupStyles are the styles for a group.
type GroupStyles struct {
	Base        lipgloss.Style
	Title       lipgloss.Style
	Description lipgloss.Style
}

// FieldStyles are the styles for input fields.
type FieldStyles struct {
	Base           lipgloss.Style
	Title          lipgloss.Style
	Description    lipgloss.Style
	ErrorIndicator lipgloss.Style
	ErrorMessage   lipgloss.Style

	// Select styles.
	SelectSelector lipgloss.Style // Selection indicator
	Option         lipgloss.Style // Select options
	NextIndicator  lipgloss.Style
	PrevIndicator  lipgloss.Style

	// FilePicker styles.
	Directory lipgloss.Style
	File      lipgloss.Style

	// Multi-select styles.
	MultiSelectSelector lipgloss.Style
	SelectedOption      lipgloss.Style
	SelectedPrefix      lipgloss.Style
	UnselectedOption    lipgloss.Style
	UnselectedPrefix    lipgloss.Style

	// Textinput and teatarea styles.
	TextInput TextInputStyles

	// Confirm styles.
	FocusedButton lipgloss.Style
	BlurredButton lipgloss.Style

	// Card styles.
	Card      lipgloss.Style
	NoteTitle lipgloss.Style
	Next      lipgloss.Style
}

// TextInputStyles are the styles for text inputs.
type TextInputStyles struct {
	Cursor      lipgloss.Style
	CursorText  lipgloss.Style
	Placeholder lipgloss.Style
	Prompt      lipgloss.Style
	Text        lipgloss.Style
}

const (
	buttonPaddingHorizontal = 2
	buttonPaddingVertical   = 0
)

// ThemeBase returns a new base theme with general styles to be inherited by
// other themes.
func ThemeBase(bool) *Styles {
	var t Styles

	t.Form.Base = lipgloss.NewStyle()
	t.Group.Base = lipgloss.NewStyle()
	t.FieldSeparator = lipgloss.NewStyle().SetString("\n\n")

	button := lipgloss.NewStyle().
		Padding(buttonPaddingVertical, buttonPaddingHorizontal).
		MarginRight(1)

	// Focused styles.
	t.Focused.Base = lipgloss.NewStyle().PaddingLeft(1).BorderStyle(lipgloss.ThickBorder()).BorderLeft(true)
	t.Focused.Card = t.Focused.Base
	t.Focused.ErrorIndicator = lipgloss.NewStyle().SetString(" *")
	t.Focused.ErrorMessage = lipgloss.NewStyle().SetString(" *")
	t.Focused.SelectSelector = lipgloss.NewStyle().SetString("> ")
	t.Focused.NextIndicator = lipgloss.NewStyle().MarginLeft(1).SetString("→")
	t.Focused.PrevIndicator = lipgloss.NewStyle().MarginRight(1).SetString("←")
	t.Focused.MultiSelectSelector = lipgloss.NewStyle().SetString("> ")
	t.Focused.SelectedPrefix = lipgloss.NewStyle().SetString("[•] ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().SetString("[ ] ")
	t.Focused.FocusedButton = button.Foreground(lipgloss.Color("0")).Background(lipgloss.Color("7"))
	t.Focused.BlurredButton = button.Foreground(lipgloss.Color("7")).Background(lipgloss.Color("0"))
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	t.Help = help.New().Styles

	// Blurred styles.
	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.MultiSelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	return &t
}

// ThemeCharm returns a new theme based on the Charm color scheme.
func ThemeCharm(isDark bool) *Styles {
	t := ThemeBase(isDark)
	lightDark := lipgloss.LightDark(isDark)

	var (
		normalFg = lightDark(lipgloss.Color("252"), lipgloss.Color("235"))
		indigo   = lightDark(lipgloss.Color("#5A56E0"), lipgloss.Color("#7571F9"))
		cream    = lightDark(lipgloss.Color("#FFFDF5"), lipgloss.Color("#FFFDF5"))
		fuchsia  = lipgloss.Color("#F780E2")
		green    = lightDark(lipgloss.Color("#02BA84"), lipgloss.Color("#02BF87"))
		red      = lightDark(lipgloss.Color("#FF4672"), lipgloss.Color("#ED567A"))
	)

	t.Focused.Base = t.Focused.Base.BorderForeground(lipgloss.Color("238"))
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(indigo).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(indigo).Bold(true).MarginBottom(1)
	t.Focused.Directory = t.Focused.Directory.Foreground(indigo)
	t.Focused.Description = t.Focused.Description.Foreground(lightDark(lipgloss.Color(""), lipgloss.Color("243")))
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(fuchsia)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(fuchsia)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(fuchsia)
	t.Focused.Option = t.Focused.Option.Foreground(normalFg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(fuchsia)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("#02CF92"), lipgloss.Color("#02A877"))).SetString("✓ ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color(""), lipgloss.Color("243"))).SetString("• ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(normalFg)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(cream).Background(fuchsia)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(normalFg).Background(lightDark(lipgloss.Color("237"), lipgloss.Color("252")))

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(green)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(lightDark(lipgloss.Color("248"), lipgloss.Color("238")))
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(fuchsia)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}

// ThemeDracula returns a new theme based on the Dracula color scheme.
func ThemeDracula(isDark bool) *Styles {
	t := ThemeBase(isDark)

	var (
		background = lipgloss.Color("#282a36")
		selection  = lipgloss.Color("#44475a")
		foreground = lipgloss.Color("#f8f8f2")
		comment    = lipgloss.Color("#6272a4")
		green      = lipgloss.Color("#50fa7b")
		purple     = lipgloss.Color("#bd93f9")
		red        = lipgloss.Color("#ff5555")
		yellow     = lipgloss.Color("#f1fa8c")
	)

	t.Focused.Base = t.Focused.Base.BorderForeground(selection)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(purple)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(purple)
	t.Focused.Description = t.Focused.Description.Foreground(comment)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.Directory = t.Focused.Directory.Foreground(purple)
	t.Focused.File = t.Focused.File.Foreground(foreground)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(yellow)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(yellow)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(yellow)
	t.Focused.Option = t.Focused.Option.Foreground(foreground)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(yellow)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(foreground)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(comment)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(yellow).Background(purple).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(foreground).Background(background)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(yellow)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(comment)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(yellow)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}

// ThemeBase16 returns a new theme based on the base16 color scheme.
func ThemeBase16(isDark bool) *Styles {
	t := ThemeBase(isDark)

	t.Focused.Base = t.Focused.Base.BorderForeground(lipgloss.Color("8"))
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(lipgloss.Color("6"))
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(lipgloss.Color("6"))
	t.Focused.Directory = t.Focused.Directory.Foreground(lipgloss.Color("6"))
	t.Focused.Description = t.Focused.Description.Foreground(lipgloss.Color("8"))
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(lipgloss.Color("9"))
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(lipgloss.Color("9"))
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(lipgloss.Color("3"))
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(lipgloss.Color("3"))
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(lipgloss.Color("3"))
	t.Focused.Option = t.Focused.Option.Foreground(lipgloss.Color("7"))
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(lipgloss.Color("3"))
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(lipgloss.Color("2"))
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(lipgloss.Color("2"))
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(lipgloss.Color("7"))
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("7")).Background(lipgloss.Color("5"))
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(lipgloss.Color("7")).Background(lipgloss.Color("0"))

	t.Focused.TextInput.Cursor.Foreground(lipgloss.Color("5"))
	t.Focused.TextInput.Placeholder.Foreground(lipgloss.Color("8"))
	t.Focused.TextInput.Prompt.Foreground(lipgloss.Color("3"))

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(lipgloss.Color("8"))
	t.Blurred.Title = t.Blurred.NoteTitle.Foreground(lipgloss.Color("8"))

	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(lipgloss.Color("8"))
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(lipgloss.Color("7"))

	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

// ThemeCatppuccin returns a new theme based on the Catppuccin color scheme.
func ThemeCatppuccin(isDark bool) *Styles {
	t := ThemeBase(isDark)

	flavour := catppuccin.Latte
	if isDark {
		flavour = catppuccin.Mocha
	}
	var (
		base     = flavour.Base()
		text     = flavour.Text()
		subtext1 = flavour.Subtext1()
		subtext0 = flavour.Subtext0()
		overlay1 = flavour.Overlay1()
		overlay0 = flavour.Overlay0()
		green    = flavour.Green()
		red      = flavour.Red()
		pink     = flavour.Pink()
		mauve    = flavour.Mauve()
		cursor   = flavour.Rosewater()
	)

	t.Focused.Base = t.Focused.Base.BorderForeground(subtext1)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(mauve)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(mauve)
	t.Focused.Directory = t.Focused.Directory.Foreground(mauve)
	t.Focused.Description = t.Focused.Description.Foreground(subtext0)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(pink)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(pink)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(pink)
	t.Focused.Option = t.Focused.Option.Foreground(text)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(pink)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(text)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(text)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(base).Background(pink)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(text).Background(base)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(cursor)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(overlay0)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(pink)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base

	t.Help.Ellipsis = t.Help.Ellipsis.Foreground(subtext0)
	t.Help.ShortKey = t.Help.ShortKey.Foreground(subtext0)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(overlay1)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(subtext0)
	t.Help.FullKey = t.Help.FullKey.Foreground(subtext0)
	t.Help.FullDesc = t.Help.FullDesc.Foreground(overlay1)
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(subtext0)

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description
	return t
}
