package huh

import (
	"cmp"
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2/internal/accessibility"
	"charm.land/lipgloss/v2"
)

// Confirm is a form confirm field.
type Confirm struct {
	accessor Accessor[bool]
	key      string
	id       int

	// customization
	title       Eval[string]
	description Eval[string]
	affirmative string
	negative    string

	// error handling
	validate func(bool) error
	err      error

	// state
	focused bool

	// options
	width           int
	height          int
	inline          bool
	theme           Theme
	hasDarkBg       bool
	keymap          ConfirmKeyMap
	buttonAlignment lipgloss.Position
}

// NewConfirm returns a new confirm field.
func NewConfirm() *Confirm {
	return &Confirm{
		accessor:        &EmbeddedAccessor[bool]{},
		id:              nextID(),
		title:           Eval[string]{cache: make(map[uint64]string)},
		description:     Eval[string]{cache: make(map[uint64]string)},
		affirmative:     "Yes",
		negative:        "No",
		validate:        func(bool) error { return nil },
		buttonAlignment: lipgloss.Center,
	}
}

// Validate sets the validation function of the confirm field.
func (c *Confirm) Validate(validate func(bool) error) *Confirm {
	c.validate = validate
	return c
}

// Error returns the error of the confirm field.
func (c *Confirm) Error() error {
	return c.err
}

// Skip returns whether the confirm should be skipped or should be blocking.
func (*Confirm) Skip() bool {
	return false
}

// Zoom returns whether the input should be zoomed.
func (*Confirm) Zoom() bool {
	return false
}

// Affirmative sets the affirmative value of the confirm field.
func (c *Confirm) Affirmative(affirmative string) *Confirm {
	c.affirmative = affirmative
	return c
}

// Negative sets the negative value of the confirm field.
func (c *Confirm) Negative(negative string) *Confirm {
	c.negative = negative
	return c
}

// Value sets the value of the confirm field.
func (c *Confirm) Value(value *bool) *Confirm {
	return c.Accessor(NewPointerAccessor(value))
}

// Accessor sets the accessor of the confirm field.
func (c *Confirm) Accessor(accessor Accessor[bool]) *Confirm {
	c.accessor = accessor
	return c
}

// Key sets the key of the confirm field.
func (c *Confirm) Key(key string) *Confirm {
	c.key = key
	return c
}

// Title sets the title of the confirm field.
func (c *Confirm) Title(title string) *Confirm {
	c.title.val = title
	c.title.fn = nil
	return c
}

// TitleFunc sets the title func of the confirm field.
func (c *Confirm) TitleFunc(f func() string, bindings any) *Confirm {
	c.title.fn = f
	c.title.bindings = bindings
	return c
}

// Description sets the description of the confirm field.
func (c *Confirm) Description(description string) *Confirm {
	c.description.val = description
	c.description.fn = nil
	return c
}

// DescriptionFunc sets the description function of the confirm field.
func (c *Confirm) DescriptionFunc(f func() string, bindings any) *Confirm {
	c.description.fn = f
	c.description.bindings = bindings
	return c
}

// Inline sets whether the field should be inline.
func (c *Confirm) Inline(inline bool) *Confirm {
	c.inline = inline
	return c
}

// Focus focuses the confirm field.
func (c *Confirm) Focus() tea.Cmd {
	c.focused = true
	return nil
}

// Blur blurs the confirm field.
func (c *Confirm) Blur() tea.Cmd {
	c.focused = false
	c.err = c.validate(c.accessor.Get())
	return nil
}

// KeyBinds returns the help message for the confirm field.
func (c *Confirm) KeyBinds() []key.Binding {
	return []key.Binding{c.keymap.Toggle, c.keymap.Prev, c.keymap.Submit, c.keymap.Next, c.keymap.Accept, c.keymap.Reject}
}

// Init initializes the confirm field.
func (c *Confirm) Init() tea.Cmd {
	return nil
}

// Update updates the confirm field.
func (c *Confirm) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		c.hasDarkBg = msg.IsDark()
	case updateFieldMsg:
		if ok, hash := c.title.shouldUpdate(); ok {
			c.title.bindingsHash = hash
			if !c.title.loadFromCache() {
				c.title.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updateTitleMsg{id: c.id, title: c.title.fn(), hash: hash}
				})
			}
		}
		if ok, hash := c.description.shouldUpdate(); ok {
			c.description.bindingsHash = hash
			if !c.description.loadFromCache() {
				c.description.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updateDescriptionMsg{id: c.id, description: c.description.fn(), hash: hash}
				})
			}
		}

	case updateTitleMsg:
		if msg.id == c.id && msg.hash == c.title.bindingsHash {
			c.title.val = msg.title
			c.title.loading = false
		}
	case updateDescriptionMsg:
		if msg.id == c.id && msg.hash == c.description.bindingsHash {
			c.description.val = msg.description
			c.description.loading = false
		}
	case tea.KeyPressMsg:
		c.err = nil
		switch {
		case key.Matches(msg, c.keymap.Toggle):
			if c.negative == "" {
				break
			}
			c.accessor.Set(!c.accessor.Get())
		case key.Matches(msg, c.keymap.Prev):
			cmds = append(cmds, PrevField)
		case key.Matches(msg, c.keymap.Next, c.keymap.Submit):
			cmds = append(cmds, NextField)
		case key.Matches(msg, c.keymap.Accept):
			c.accessor.Set(true)
			cmds = append(cmds, NextField)
		case key.Matches(msg, c.keymap.Reject):
			c.accessor.Set(false)
			cmds = append(cmds, NextField)
		}
	}

	return c, tea.Batch(cmds...)
}

func (c *Confirm) activeStyles() *FieldStyles {
	theme := c.theme
	if theme == nil {
		theme = ThemeFunc(ThemeCharm)
	}
	if c.focused {
		return &theme.Theme(c.hasDarkBg).Focused
	}
	return &theme.Theme(c.hasDarkBg).Blurred
}

// View renders the confirm field.
func (c *Confirm) View() string {
	styles := c.activeStyles()
	maxWidth := c.width - styles.Base.GetHorizontalFrameSize()

	var wroteHeader bool
	var sb strings.Builder
	if c.title.val != "" {
		sb.WriteString(styles.Title.Render(wrap(c.title.val, maxWidth)))
		wroteHeader = true
	}
	if c.err != nil {
		sb.WriteString(styles.ErrorIndicator.String())
		wroteHeader = true
	}

	if c.description.val != "" {
		description := styles.Description.Render(wrap(c.description.val, maxWidth))
		if !c.inline && (c.description.val != "" || c.description.fn != nil) {
			sb.WriteString("\n")
		}
		sb.WriteString(description)
		wroteHeader = true
	}

	if !c.inline && wroteHeader {
		sb.WriteString("\n")
		sb.WriteString("\n")
	}

	var negative string
	var affirmative string
	if c.negative != "" {
		if c.accessor.Get() {
			affirmative = styles.FocusedButton.Render(c.affirmative)
			negative = styles.BlurredButton.Render(c.negative)
		} else {
			affirmative = styles.BlurredButton.Render(c.affirmative)
			negative = styles.FocusedButton.Render(c.negative)
		}
		c.keymap.Reject.SetHelp("n", c.negative)
	} else {
		affirmative = styles.FocusedButton.Render(c.affirmative)
		c.keymap.Reject.SetEnabled(false)
	}

	c.keymap.Accept.SetHelp("y", c.affirmative)

	buttonsRow := lipgloss.JoinHorizontal(c.buttonAlignment, affirmative, negative)

	promptWidth := lipgloss.Width(sb.String())
	buttonsWidth := lipgloss.Width(buttonsRow)

	renderWidth := max(buttonsWidth, promptWidth)

	style := lipgloss.NewStyle().Width(renderWidth).Align(c.buttonAlignment)

	sb.WriteString(style.Render(buttonsRow))
	return styles.Base.Width(c.width).Height(c.height).
		Render(sb.String())
}

// Run runs the confirm field in accessible mode.
func (c *Confirm) Run() error {
	return Run(c)
}

// RunAccessible runs the confirm field in accessible mode.
func (c *Confirm) RunAccessible(w io.Writer, r io.Reader) error {
	styles := c.activeStyles()
	defaultValue := c.GetValue().(bool)
	opts := "[y/N]"
	if defaultValue {
		opts = "[Y/n]"
	}
	prompt := styles.Title.
		PaddingRight(1).
		Render(cmp.Or(c.title.val, "Choose"), opts)
	c.accessor.Set(accessibility.PromptBool(w, r, prompt, defaultValue))
	return nil
}

func (c *Confirm) String() string {
	if c.accessor.Get() {
		return c.affirmative
	}
	return c.negative
}

// WithTheme sets the theme of the confirm field.
func (c *Confirm) WithTheme(theme Theme) Field {
	if c.theme != nil {
		return c
	}
	c.theme = theme
	return c
}

// WithKeyMap sets the keymap of the confirm field.
func (c *Confirm) WithKeyMap(k *KeyMap) Field {
	c.keymap = k.Confirm
	return c
}

// WithWidth sets the width of the confirm field.
func (c *Confirm) WithWidth(width int) Field {
	c.width = width
	return c
}

// WithHeight sets the height of the confirm field.
func (c *Confirm) WithHeight(height int) Field {
	c.height = height
	return c
}

// WithPosition sets the position of the confirm field.
func (c *Confirm) WithPosition(p FieldPosition) Field {
	c.keymap.Prev.SetEnabled(!p.IsFirst())
	c.keymap.Next.SetEnabled(!p.IsLast())
	c.keymap.Submit.SetEnabled(p.IsLast())
	return c
}

// WithButtonAlignment sets the button position of the confirm field.
func (c *Confirm) WithButtonAlignment(p lipgloss.Position) *Confirm {
	c.buttonAlignment = p
	return c
}

// GetKey returns the key of the field.
func (c *Confirm) GetKey() string {
	return c.key
}

// GetValue returns the value of the field.
func (c *Confirm) GetValue() any {
	return c.accessor.Get()
}
