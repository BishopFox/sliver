// Package spinner provides a loading spinner.
package spinner

import (
	"cmp"
	"context"
	"io"
	"os"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2/internal/compat"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// Model is an alias to [compat.Model].
type Model = compat.Model

// Spinner represents a loading spinner.
// To get started simply create a new spinner and call `Run`.
//
//	s := spinner.New()
//	s.Run()
//
// ⣾  Loading...
type Spinner struct {
	spinner    spinner.Model
	action     func(ctx context.Context) error
	ctx        context.Context
	accessible bool
	title      string
	err        error
	teaOptions []tea.ProgramOption
	viewHook   compat.ViewHook
	theme      Theme
	output     io.Writer // acessible mode output
	input      io.Reader // acessible mode output
	hasDarkBg  bool
}

// Styles are the spinner styles.
type Styles struct {
	Spinner, Title lipgloss.Style
}

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

// ThemeDefault is the default theme.
func ThemeDefault(isDark bool) *Styles {
	lightDark := lipgloss.LightDark(isDark)
	title := lightDark(
		lipgloss.Color("#00020A"),
		lipgloss.Color("#FFFDF5"),
	)
	return &Styles{
		Spinner: lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2")),
		Title:   lipgloss.NewStyle().Foreground(title),
	}
}

// Type is a set of frames used in animating the spinner.
type Type spinner.Spinner

// Spinner [Type]s.
var (
	Line      = Type(spinner.Line)
	Dots      = Type(spinner.Dot)
	MiniDot   = Type(spinner.MiniDot)
	Jump      = Type(spinner.Jump)
	Points    = Type(spinner.Points)
	Pulse     = Type(spinner.Pulse)
	Globe     = Type(spinner.Globe)
	Moon      = Type(spinner.Moon)
	Monkey    = Type(spinner.Monkey)
	Meter     = Type(spinner.Meter)
	Hamburger = Type(spinner.Hamburger)
	Ellipsis  = Type(spinner.Ellipsis)
)

// Type sets the type of the spinner.
func (s *Spinner) Type(t Type) *Spinner {
	s.spinner.Spinner = spinner.Spinner(t)
	return s
}

// Title sets the title of the spinner.
func (s *Spinner) Title(title string) *Spinner {
	s.title = title
	return s
}

// WithOutput set the output for the spinner.
// Default is STDOUT when [Spinner.WithAccessible], STDERR otherwise.
func (s *Spinner) WithOutput(w io.Writer) *Spinner {
	s.teaOptions = append(s.teaOptions, tea.WithOutput(w))
	s.output = w
	return s
}

// WithInput set the input for the spinner.
// Default is STDIN.
func (s *Spinner) WithInput(r io.Reader) *Spinner {
	s.teaOptions = append(s.teaOptions, tea.WithInput(r))
	s.input = r
	return s
}

// WithViewHook allows to set a [compat.ViewHook].
func (s *Spinner) WithViewHook(hook compat.ViewHook) *Spinner {
	s.viewHook = hook
	return s
}

// Action sets the action of the spinner.
func (s *Spinner) Action(action func()) *Spinner {
	s.action = func(context.Context) error {
		action()
		return nil
	}
	return s
}

// ActionWithErr sets the action of the spinner.
//
// This is just like [Spinner.Action], but allows the action to use a `context.Context`
// and to return an error.
func (s *Spinner) ActionWithErr(action func(context.Context) error) *Spinner {
	s.action = action
	return s
}

// Context sets the context of the spinner.
func (s *Spinner) Context(ctx context.Context) *Spinner {
	s.ctx = ctx
	return s
}

// WithAccessible sets the spinner to be static.
func (s *Spinner) WithAccessible(accessible bool) *Spinner {
	s.accessible = accessible
	return s
}

// New creates a new spinner.
func New() *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return &Spinner{
		spinner: s,
		title:   "Loading...",
		theme:   ThemeFunc(ThemeDefault),
	}
}

// WithTheme sets the theme for the spinner.
func (s *Spinner) WithTheme(theme Theme) *Spinner {
	if theme == nil {
		return s
	}

	s.theme = theme
	return s
}

// Init initializes the spinner.
func (s *Spinner) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		s.spinner.Tick,
		func() tea.Msg {
			if s.action != nil {
				err := s.action(s.ctx)
				return doneMsg{err}
			}
			return nil
		},
	)
}

// Update updates the spinner.
func (s *Spinner) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		s.hasDarkBg = msg.IsDark()
	case doneMsg:
		s.err = msg.err
		return s, tea.Quit
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return s, tea.Interrupt
		}
	}

	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View returns the spinner view.
func (s *Spinner) View() string {
	styles := s.theme.Theme(s.hasDarkBg)
	s.spinner.Style = styles.Spinner
	var title string
	if s.title != "" {
		title = styles.Title.Render(s.title)
	}
	return s.spinner.View() + title
}

// Run runs the spinner.
func (s *Spinner) Run() error {
	if s.ctx == nil && s.action == nil {
		return nil
	}
	if s.ctx == nil {
		s.ctx = context.Background()
	}
	if err := s.ctx.Err(); err != nil {
		return err //nolint:wrapcheck
	}

	if s.accessible {
		out := cmp.Or[io.Writer](s.output, os.Stdout)
		in := cmp.Or[io.Reader](s.input, os.Stdin)
		return s.runAccessible(in, out)
	}

	opts := append(s.teaOptions, tea.WithContext(s.ctx))
	if s.output != nil {
		opts = append(opts, tea.WithOutput(s.output))
	}
	if s.input != nil {
		opts = append(opts, tea.WithInput(s.input))
	}
	m, err := tea.NewProgram(compat.ViewModel{
		Model:    s,
		ViewHook: s.viewHook,
	}, opts...).Run()
	mm := m.(compat.ViewModel).Model.(*Spinner)
	if mm.err != nil {
		return mm.err
	}
	return err //nolint:wrapcheck
}

// runAccessible runs the spinner in an accessible mode (statically).
func (s *Spinner) runAccessible(in io.Reader, out io.Writer) error {
	tin, iok := in.(term.File)
	tout, ook := out.(term.File)

	s.hasDarkBg = true
	if iok && ook {
		s.hasDarkBg = lipgloss.HasDarkBackground(tin, tout)
	}

	styles := s.theme.Theme(s.hasDarkBg)

	_, _ = io.WriteString(out, ansi.HideCursor)
	frame := s.spinner.Style.Render("...")
	title := styles.Title.Render(strings.TrimSuffix(s.title, "..."))
	_, _ = io.WriteString(out, title+frame)

	defer func() {
		_, _ = io.WriteString(out, ansi.ShowCursor)
	}()

	actionDone := make(chan error)
	if s.action != nil {
		go func() {
			actionDone <- s.action(s.ctx)
		}()
	}

	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err() //nolint:wrapcheck
		case err := <-actionDone:
			return err
		}
	}
}

type doneMsg struct {
	err error
}
