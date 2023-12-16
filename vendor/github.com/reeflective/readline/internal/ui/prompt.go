package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/core"
	"github.com/reeflective/readline/internal/keymap"
	"github.com/reeflective/readline/internal/strutil"
	"github.com/reeflective/readline/internal/term"
)

// Prompt stores all prompt rendering/generation functions and is
// in charge of displaying them, as well as computing their offsets.
type Prompt struct {
	primaryF    func() string
	primaryRows int
	primaryCols int

	secondaryF func() string
	transientF func() string
	rightF     func() string
	tooltipF   func() string

	// True if some logs have printed asynchronously
	// since last loop. Check refresh prompt funcs.
	refreshing bool

	// Shell parameters
	line    *core.Line
	cursor  *core.Cursor
	keymaps *keymap.Engine
	opts    *inputrc.Config
}

// NewPrompt is a required constructor to initialize the prompt system.
func NewPrompt(line *core.Line, cursor *core.Cursor, keymaps *keymap.Engine, opts *inputrc.Config) *Prompt {
	return &Prompt{
		line:    line,
		cursor:  cursor,
		keymaps: keymaps,
		opts:    opts,
	}
}

// Primary uses a function returning the string to use as the primary prompt.
func (p *Prompt) Primary(prompt func() string) {
	p.primaryF = prompt
}

// Right uses a function returning the string to use as the right prompt.
func (p *Prompt) Right(prompt func() string) {
	p.rightF = prompt
}

// Secondary uses a function returning the prompt to use as the secondary prompt.
func (p *Prompt) Secondary(prompt func() string) {
	p.secondaryF = prompt
}

// Transient uses a function returning the prompt to use as a transient prompt.
func (p *Prompt) Transient(prompt func() string) {
	p.transientF = prompt
}

// Tooltip uses a function returning the prompt to use as a tooltip prompt.
func (p *Prompt) Tooltip(prompt func(word string) string) {
	if prompt == nil {
		return
	}

	// Wrap the user-provided function into a callback using out input line.
	p.tooltipF = func() string {
		var tooltipWord string

		shellWords := strings.Split(string(*p.line), " ")

		if len(shellWords) > 0 {
			tooltipWord = shellWords[0]
		}

		return prompt(tooltipWord)
	}
}

// PrimaryPrint prints the primary prompt string, excluding
// the last line if the primary prompt spans on several lines.
func (p *Prompt) PrimaryPrint() {
	p.refreshing = false

	if p.primaryF == nil {
		return
	}

	prompt := p.primaryF()

	prompt, lastPrompt := p.formatPrimaryLines(prompt)

	// Format the last line with the editing status.
	lastPrompt = p.formatLastPrompt(lastPrompt)

	// Print the various lines.
	if prompt != "" {
		fmt.Print(prompt)
	}

	fmt.Print(lastPrompt)

	// And compute coordinates
	p.primaryRows = strings.Count(prompt, "\n")
	p.primaryCols = strutil.RealLength(lastPrompt)

	if p.primaryCols > 0 {
		p.primaryCols--
	}
}

// PrimaryUsed returns the number of terminal rows on which
// the primary prompt string spans, excluding the last line
// if it contains newlines.
func (p *Prompt) PrimaryUsed() int {
	return p.primaryRows
}

// LastPrint prints the last line of the primary prompt, if the latter
// spans on several lines. If not, this function will actually print
// the entire primary prompt, and PrimaryPrint() will not print anything.
func (p *Prompt) LastPrint() {
	if p.primaryF == nil {
		return
	}

	// Only display the last line, but overwrite the number of
	// rows used since any redisplay of all lines but the last
	// will trigger their  own recomputation.
	lines := strings.Split(p.primaryF(), "\n")

	// Print the prompt and compute columns.
	if len(lines) == 0 {
		return
	}

	prompt := p.formatLastPrompt(lines[len(lines)-1])

	fmt.Print(prompt)

	p.primaryCols = strutil.RealLength(prompt)
	if p.primaryCols > 0 {
		p.primaryCols--
	}
}

// LastUsed returns the number of terminal columns used by the last
// part of the primary prompt (of the entire string if not multiline).
// This, in effect, returns the X coordinate at which the input line
// should be printed, and indentation for subsequent lines if several.
func (p *Prompt) LastUsed() int {
	if p.primaryF == nil {
		return 0
	}

	// Only display the last line, but overwrite the number of
	// rows used since any redisplay of all lines but the last
	// will trigger their  own recomputation.
	lines := strings.Split(p.primaryF(), "\n")
	if len(lines) == 0 {
		return 0
	}

	prompt := p.formatLastPrompt(lines[len(lines)-1])
	p.primaryCols = strutil.RealLength(prompt)

	if p.primaryCols > 0 {
		p.primaryCols--
	}

	return p.primaryCols
}

// RightPrint prints the right-sided prompt strings, which might be either
// a traditional RPROMPT string, or a tooltip prompt if any must be rendered.
// If force is true, whatever rprompt or tooltip exists will be printed.
// If false, only the rprompt, if it exists, will be printed.
func (p *Prompt) RightPrint(startColumn int, force bool) {
	var rprompt string

	if p.tooltipF != nil && force {
		rprompt = p.tooltipF()
	}

	if rprompt == "" && p.rightF != nil {
		rprompt = p.rightF()
	}

	if rprompt == "" {
		return
	}

	if prompt, canPrint := p.formatRightPrompt(rprompt, startColumn); canPrint {
		fmt.Print(prompt)
	} else {
		fmt.Print(term.ClearLineAfter)
	}
}

// TransientPrint prints the transient prompt.
func (p *Prompt) TransientPrint() {
	if p.transientF == nil {
		return
	}

	// Clean everything below where the prompt will be printed.
	term.MoveCursorBackwards(term.GetWidth())
	term.MoveCursorUp(p.primaryRows)
	fmt.Print(term.ClearScreenBelow)

	// And print the prompt
	fmt.Print(p.transientF())
}

// Refreshing returns true if the prompt is currently redisplaying
// itself (at least the primary prompt), or false if not.
func (p *Prompt) Refreshing() bool {
	return p.refreshing
}

func (p *Prompt) formatLastPrompt(prompt string) string {
	if !p.opts.GetBool("show-mode-in-prompt") {
		return prompt
	}

	var status string

	switch {
	case p.keymaps.IsEmacs():
		status = p.opts.GetString("emacs-mode-string")
	case p.keymaps.Main() == keymap.ViCommand:
		status = p.opts.GetString("vi-cmd-mode-string")
	case p.keymaps.Main() == keymap.ViInsert:
		status = p.opts.GetString("vi-ins-mode-string")
	}

	// Fix parsing of inputrc which sometimes preserves quotes on some
	// values, and remove bash readline begin/end non-printable delimiters.
	status = strings.Trim(status, "\"")

	begin := regexp.MustCompile(`\\1`)
	end := regexp.MustCompile(`\\2`)

	// Remove delimiters, and replace quoted escape sequences
	status = begin.ReplaceAllString(status, "")
	status = end.ReplaceAllString(status, "")
	status = strings.ReplaceAll(status, "\\e", "\x1b")

	return status + prompt
}

func (p *Prompt) formatRightPrompt(rprompt string, startColumn int) (prompt string, canPrint bool) {
	// Dimensions
	termWidth := term.GetWidth()
	promptLen := strutil.RealLength(rprompt)
	padLen := termWidth - startColumn - promptLen

	// Adjust padding when the last line is as large as terminal.
	if startColumn == termWidth {
		padLen = startColumn - promptLen
	}

	// Check that we have room for a right/tooltip prompt.
	canPrint = (startColumn+promptLen < termWidth) || startColumn == termWidth
	if canPrint {
		prompt = fmt.Sprintf("%s%s", strings.Repeat(" ", padLen), rprompt)
	}

	return
}

func (p *Prompt) formatPrimaryLines(prompt string) (multi, lastPrompt string) {
	// Get all the lines but the last.
	lines := strings.Split(prompt, "\n")

	if len(lines) > 1 {
		multi = strings.Join(lines[:len(lines)-1], "\n") + "\n"
		lastPrompt = lines[len(lines)-1]
	} else {
		lastPrompt, multi = prompt, ""
	}

	return multi, lastPrompt
}
