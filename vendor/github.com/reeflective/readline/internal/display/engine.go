package display

import (
	"fmt"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/completion"
	"github.com/reeflective/readline/internal/core"
	"github.com/reeflective/readline/internal/history"
	"github.com/reeflective/readline/internal/strutil"
	"github.com/reeflective/readline/internal/term"
	"github.com/reeflective/readline/internal/ui"
)

var (
	oneThirdTerminalHeight = 3
	halfTerminalHeight     = 2
)

// Engine handles all display operations: it refreshes the terminal
// interface and stores the necessary offsets of each components.
type Engine struct {
	// Operating parameters
	highlighter    func(line []rune) string
	startCols      int
	startRows      int
	lineCol        int
	lineRows       int
	cursorRow      int
	cursorCol      int
	hintRows       int
	compRows       int
	primaryPrinted bool

	// UI components
	keys      *core.Keys
	line      *core.Line
	suggested core.Line
	cursor    *core.Cursor
	selection *core.Selection
	histories *history.Sources
	prompt    *ui.Prompt
	hint      *ui.Hint
	completer *completion.Engine
	opts      *inputrc.Config
}

// NewEngine is a required constructor for the display engine.
func NewEngine(k *core.Keys, s *core.Selection, h *history.Sources, p *ui.Prompt, i *ui.Hint, c *completion.Engine, opts *inputrc.Config) *Engine {
	return &Engine{
		keys:      k,
		selection: s,
		histories: h,
		prompt:    p,
		hint:      i,
		completer: c,
		opts:      opts,
	}
}

// Init computes some base coordinates needed before displaying the line and helpers.
// The shell syntax highlighter is also provided here, since any consumer library will
// have bound it after instantiating a new shell instance.
func Init(e *Engine, highlighter func([]rune) string) {
	e.highlighter = highlighter
}

// Refresh recomputes and redisplays the entire readline interface, except
// the first lines of the primary prompt when the latter is a multiline one.
func (e *Engine) Refresh() {
	fmt.Print(term.HideCursor)

	// Go back to the first column, and if the primary prompt
	// was not printed yet, back up to the line's beginning row.
	term.MoveCursorBackwards(term.GetWidth())

	if !e.primaryPrinted {
		term.MoveCursorUp(e.cursorRow)
	}

	// Print either all or the last line of the prompt.
	e.prompt.LastPrint()

	// Get all positions required for the redisplay to come:
	// prompt end (thus indentation), cursor positions, etc.
	e.computeCoordinates(true)

	// Print the line, and any of the secondary and right prompts.
	e.displayLine()
	e.displayMultilinePrompts()

	// Display hints and completions, go back
	// to the start of the line, then to cursor.
	e.displayHelpers()
	e.cursorHintToLineStart()
	e.lineStartToCursorPos()
	fmt.Print(term.ShowCursor)
}

// PrintPrimaryPrompt redraws the primary prompt.
// There are relatively few cases where you want to use this.
// It is currently only used when using clear-screen commands.
func (e *Engine) PrintPrimaryPrompt() {
	e.prompt.PrimaryPrint()
	e.primaryPrinted = true
}

// ClearHelpers clears the hint and completion sections below the line.
func (e *Engine) ClearHelpers() {
	e.CursorBelowLine()
	fmt.Print(term.ClearScreenBelow)

	term.MoveCursorUp(1)
	term.MoveCursorUp(e.lineRows)
	term.MoveCursorDown(e.cursorRow)
	term.MoveCursorForwards(e.cursorCol)
}

// ResetHelpers cancels all active hints and completions.
func (e *Engine) ResetHelpers() {
	e.hint.Reset()
	e.completer.ClearMenu(true)
}

// AcceptLine redraws the current UI when the line has been accepted
// and returned to the caller. After clearing various things such as
// hints, completions and some right prompts, the shell will put the
// display at the start of the line immediately following the line.
func (e *Engine) AcceptLine() {
	e.CursorToLineStart()

	e.computeCoordinates(false)

	// Go back to the end of the non-suggested line.
	term.MoveCursorBackwards(term.GetWidth())
	term.MoveCursorDown(e.lineRows)
	term.MoveCursorForwards(e.lineCol)
	fmt.Print(term.ClearScreenBelow)

	// Reprint the right-side prompt if it's not a tooltip one.
	e.prompt.RightPrint(e.lineCol, false)

	// Go below this non-suggested line and clear everything.
	term.MoveCursorBackwards(term.GetWidth())
	fmt.Print(term.NewlineReturn)
}

// RefreshTransient goes back to the first line of the input buffer
// and displays the transient prompt, then redisplays the input line.
func (e *Engine) RefreshTransient() {
	if !e.opts.GetBool("prompt-transient") {
		return
	}

	// Go to the beginning of the primary prompt.
	e.CursorToLineStart()
	term.MoveCursorUp(e.prompt.PrimaryUsed())

	// And redisplay the transient/primary/line.
	e.prompt.TransientPrint()
	e.displayLine()
	fmt.Print(term.NewlineReturn)
}

// CursorToLineStart moves the cursor just after the primary prompt.
// This function should only be called when the cursor is on its
// "cursor" position on the input line.
func (e *Engine) CursorToLineStart() {
	term.MoveCursorBackwards(e.cursorCol)
	term.MoveCursorUp(e.cursorRow)
	term.MoveCursorForwards(e.startCols)
}

// CursorBelowLine moves the cursor to the leftmost
// column of the first row after the last line of input.
// This function should only be called when the cursor
// is on its "cursor" position on the input line.
func (e *Engine) CursorBelowLine() {
	term.MoveCursorUp(e.cursorRow)
	term.MoveCursorDown(e.lineRows)
	fmt.Print(term.NewlineReturn)
}

// lineStartToCursorPos can be used if the cursor is currently
// at the very start of the input line, that is just after the
// last character of the prompt.
func (e *Engine) lineStartToCursorPos() {
	term.MoveCursorDown(e.cursorRow)
	term.MoveCursorBackwards(term.GetWidth())
	term.MoveCursorForwards(e.cursorCol)
}

// cursor is on the line below the last line of input.
func (e *Engine) cursorHintToLineStart() {
	term.MoveCursorUp(1)
	term.MoveCursorUp(e.lineRows - e.cursorRow)
	e.CursorToLineStart()
}

func (e *Engine) computeCoordinates(suggested bool) {
	// Get the new input line and auto-suggested one.
	e.line, e.cursor = e.completer.Line()
	if e.completer.IsInserting() {
		e.suggested = *e.line
	} else {
		e.suggested = e.histories.Suggest(e.line)
	}

	// Get the position of the line's beginning by querying
	// the terminal for the cursor position.
	e.startCols, e.startRows = e.keys.GetCursorPos()

	if e.startCols > 0 {
		e.startCols--
	}

	// Cursor position might be misleading if invalid (negative).
	if e.startCols == -1 {
		e.startCols = e.prompt.LastUsed()
	}

	e.cursorCol, e.cursorRow = core.CoordinatesCursor(e.cursor, e.startCols)

	// Get the number of rows used by the line, and the end line X pos.
	if e.opts.GetBool("history-autosuggest") && suggested {
		e.lineCol, e.lineRows = core.CoordinatesLine(&e.suggested, e.startCols)
	} else {
		e.lineCol, e.lineRows = core.CoordinatesLine(e.line, e.startCols)
	}

	e.primaryPrinted = false
}

func (e *Engine) displayLine() {
	var line string

	// Apply user-defined highlighter to the input line.
	if e.highlighter != nil {
		line = e.highlighter(*e.line)
	} else {
		line = string(*e.line)
	}

	// Highlight matching parenthesis
	if e.opts.GetBool("blink-matching-paren") {
		core.HighlightMatchers(e.selection)
		defer core.ResetMatchers(e.selection)
	}

	// Apply visual selections highlighting if any
	line = e.highlightLine([]rune(line), *e.selection)

	// Get the subset of the suggested line to print.
	if len(e.suggested) > e.line.Len() && e.opts.GetBool("history-autosuggest") {
		line += color.Dim + color.Fmt(color.Fg+"242") + string(e.suggested[e.line.Len():]) + color.Reset
	}

	// Format tabs as spaces, for consistent display
	line = strutil.FormatTabs(line) + term.ClearLineAfter

	// And display the line.
	e.suggested.Set([]rune(line)...)
	core.DisplayLine(&e.suggested, e.startCols)

	// Adjust the cursor if the line fits exactly in the terminal width.
	if e.lineCol == 0 {
		fmt.Print(term.NewlineReturn)
		fmt.Print(term.ClearLineAfter)
	}
}

func (e *Engine) displayMultilinePrompts() {
	// If we have more than one line, write the columns.
	if e.line.Lines() > 1 {
		term.MoveCursorUp(e.lineRows)
		term.MoveCursorBackwards(term.GetWidth())
		e.prompt.MultilineColumnPrint()
	}

	// Then if we have a line at all, rewrite the last column
	// character with any secondary prompt available.
	if e.line.Lines() > 0 {
		term.MoveCursorBackwards(term.GetWidth())
		e.prompt.SecondaryPrint()
		term.MoveCursorBackwards(term.GetWidth())
		term.MoveCursorForwards(e.lineCol)
	}

	// Then prompt the right-sided prompt if possible.
	e.prompt.RightPrint(e.lineCol, true)
}

// displayHelpers renders the hint and completion sections.
// It assumes that the cursor is on the last line of input,
// and goes back to this same line after displaying this.
func (e *Engine) displayHelpers() {
	fmt.Print(term.NewlineReturn)

	// Recompute completions and hints if autocompletion is on.
	e.completer.Autocomplete()

	// Display hint and completions.
	ui.DisplayHint(e.hint)
	e.hintRows = ui.CoordinatesHint(e.hint)
	completion.Display(e.completer, e.AvailableHelperLines())
	e.compRows = completion.Coordinates(e.completer)

	// Go back to the first line below the input line.
	term.MoveCursorBackwards(term.GetWidth())
	term.MoveCursorUp(e.compRows)
	term.MoveCursorUp(ui.CoordinatesHint(e.hint))
}

// AvailableHelperLines returns the number of lines available below the hint section.
// It returns half the terminal space if we currently have less than 1/3rd of it below.
func (e *Engine) AvailableHelperLines() int {
	termHeight := term.GetLength()
	compLines := termHeight - e.startRows - e.lineRows - e.hintRows

	if compLines < (termHeight / oneThirdTerminalHeight) {
		compLines = (termHeight / halfTerminalHeight)
	}

	return compLines
}
