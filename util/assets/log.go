package assets

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	huhspinner "github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type styles struct {
	title   lipgloss.Style
	section lipgloss.Style
	tag     lipgloss.Style
	key     lipgloss.Style
	success lipgloss.Style
	warn    lipgloss.Style
	error   lipgloss.Style
	muted   lipgloss.Style
	spinner lipgloss.Style
}

type logger struct {
	verbose        bool
	quiet          bool
	out            io.Writer
	err            io.Writer
	styles         styles
	spinnerEnabled bool
	section        string
	sectionCount   int
}

func newLogger(verbose, quiet, noColor bool) *logger {
	out := os.Stdout
	err := os.Stderr

	var styled styles
	if noColor {
		styled = styles{
			title:   lipgloss.NewStyle(),
			section: lipgloss.NewStyle(),
			tag:     lipgloss.NewStyle(),
			key:     lipgloss.NewStyle(),
			success: lipgloss.NewStyle(),
			warn:    lipgloss.NewStyle(),
			error:   lipgloss.NewStyle(),
			muted:   lipgloss.NewStyle(),
			spinner: lipgloss.NewStyle(),
		}
	} else {
		styled = styles{
			title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
			section: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")),
			tag:     lipgloss.NewStyle().Foreground(lipgloss.Color("69")),
			key:     lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
			success: lipgloss.NewStyle().Foreground(lipgloss.Color("82")),
			warn:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
			error:   lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
			muted:   lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
			spinner: lipgloss.NewStyle().Foreground(lipgloss.Color("69")),
		}
	}

	return &logger{
		verbose:        verbose,
		quiet:          quiet,
		out:            out,
		err:            err,
		styles:         styled,
		spinnerEnabled: !verbose && !quiet && isTerminal(out),
	}
}

func (l *logger) Header(title string) {
	if l.quiet {
		return
	}
	fmt.Fprintln(l.out, l.styles.title.Render(title))
}

func (l *logger) Meta(key, value string) {
	if l.quiet {
		return
	}
	label := l.styles.key.Render(fmt.Sprintf("%s:", strings.ToLower(key)))
	fmt.Fprintf(l.out, "%s %s\n", label, value)
}

func (l *logger) Section(title string) {
	l.section = strings.ToLower(title)
	if l.quiet {
		return
	}
	if l.sectionCount > 0 {
		fmt.Fprintln(l.out)
	}
	l.sectionCount++
	fmt.Fprintln(l.out, l.styles.section.Render(title))
}

func (l *logger) ClearSection() {
	l.section = ""
}

func (l *logger) Logf(format string, args ...any) {
	if l.quiet {
		return
	}
	msg := fmt.Sprintf(format, args...)
	if msg == "" {
		fmt.Fprintln(l.out)
		return
	}
	fmt.Fprintln(l.out, l.withPrefix(msg))
}

func (l *logger) VLogf(format string, args ...any) {
	if l.quiet || !l.verbose {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(l.out, l.withPrefix(l.styles.muted.Render(msg)))
}

func (l *logger) Successf(format string, args ...any) {
	if l.quiet {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(l.out, l.withPrefix(l.styles.success.Render(msg)))
}

func (l *logger) Warnf(format string, args ...any) {
	if l.quiet {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(l.out, l.withPrefix(l.styles.warn.Render(msg)))
}

func (l *logger) Errorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(l.err, l.withPrefix(l.styles.error.Render(msg)))
}

func (l *logger) RunSpinner(label string, action func() error) error {
	if l.quiet || !l.spinnerEnabled {
		return action()
	}

	spin := huhspinner.New().
		Type(huhspinner.Pulse).
		Title(label).
		TitleStyle(l.styles.muted).
		Style(l.styles.spinner).
		Output(l.out).
		ActionWithErr(func(context.Context) error {
			return action()
		})

	return spin.Run()
}

func (l *logger) withPrefix(msg string) string {
	prefix := l.prefix()
	if prefix == "" {
		return msg
	}
	return prefix + " " + msg
}

func (l *logger) prefix() string {
	if l.section == "" {
		return ""
	}
	return l.styles.tag.Render("[" + l.section + "]")
}

func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
