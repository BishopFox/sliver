package spinner

import (
	"cmp"
	"context"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

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
	titleStyle lipgloss.Style
	output     io.Writer
	err        error
}

type Type spinner.Spinner

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

// Output set the output for the spinner.
// Default is STDOUT when [Spinner.Accessible], STDERR otherwise.
func (s *Spinner) Output(w io.Writer) *Spinner {
	s.output = w
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

// Style sets the style of the spinner.
func (s *Spinner) Style(style lipgloss.Style) *Spinner {
	s.spinner.Style = style
	return s
}

// TitleStyle sets the title style of the spinner.
func (s *Spinner) TitleStyle(style lipgloss.Style) *Spinner {
	s.titleStyle = style
	return s
}

// Accessible sets the spinner to be static.
func (s *Spinner) Accessible(accessible bool) *Spinner {
	s.accessible = accessible
	return s
}

// New creates a new spinner.
func New() *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot

	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2"))

	return &Spinner{
		spinner:    s,
		title:      "Loading...",
		titleStyle: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#00020A", Dark: "#FFFDF5"}),
	}
}

// Init initializes the spinner.
func (s *Spinner) Init() tea.Cmd {
	return tea.Batch(s.spinner.Tick, func() tea.Msg {
		if s.action != nil {
			err := s.action(s.ctx)
			return doneMsg{err}
		}
		return nil
	})
}

// Update updates the spinner.
func (s *Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		s.err = msg.err
		return s, tea.Quit
	case tea.KeyMsg:
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
	var title string
	if s.title != "" {
		title = s.titleStyle.Render(s.title)
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
		return err
	}

	if s.accessible {
		out := cmp.Or[io.Writer](s.output, os.Stdout)
		return s.runAccessible(out)
	}

	m, err := tea.NewProgram(
		s,
		tea.WithContext(s.ctx),
		tea.WithOutput(s.output),
		tea.WithInput(nil),
	).Run()
	mm := m.(*Spinner)
	if mm.err != nil {
		return mm.err
	}
	return err
}

// runAccessible runs the spinner in an accessible mode (statically).
func (s *Spinner) runAccessible(out io.Writer) error {
	output := termenv.NewOutput(out)
	output.HideCursor()
	frame := s.spinner.Style.Render("...")
	title := s.titleStyle.Render(strings.TrimSuffix(s.title, "..."))
	_, _ = io.WriteString(out, title+frame)

	defer func() {
		output.ShowCursor()
		output.CursorBack(len(frame) + len(title))
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
			return s.ctx.Err()
		case err := <-actionDone:
			return err
		}
	}
}

type doneMsg struct {
	err error
}
