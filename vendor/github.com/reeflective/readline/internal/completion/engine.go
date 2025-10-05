package completion

import (
	"regexp"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/core"
	"github.com/reeflective/readline/internal/keymap"
	"github.com/reeflective/readline/internal/ui"
)

// Engine is responsible for all completion tasks: generating, computing,
// displaying and updating completion values and inserted candidates.
type Engine struct {
	config        *inputrc.Config // The inputrc contains options relative to completion.
	cached        Completer       // A cached completer function to use when updating.
	autoCompleter Completer       // Completer used by things like autocomplete
	hint          *ui.Hint        // The completions can feed hint/usage messages

	// Line parameters
	keys       *core.Keys      // The input keys reader
	line       *core.Line      // The real input line of the shell.
	cursor     *core.Cursor    // The cursor of the shell.
	selection  *core.Selection // The line selection
	compLine   *core.Line      // A line that might include a virtually inserted candidate.
	compCursor *core.Cursor    // The adjusted cursor.
	keymap     *keymap.Engine  // The main/local keymaps of the shell

	// Completion parameters
	groups      []*group      // All of our suggestions tree is in here
	sm          SuffixMatcher // The suffix matcher is kept for removal after actually inserting the candidate.
	selected    Candidate     // The currently selected item, not yet a real part of the input line.
	prefix      string        // The current tab completion prefix against which to build candidates
	suffix      string        // The current word suffix
	inserted    []rune        // The selected candidate (inserted in line) without prefix or suffix.
	usedY       int           // Comprehensive size offset (terminal rows) of the currently built completions.
	auto        bool          // Is the engine autocompleting ?
	autoForce   bool          // Special autocompletion mode (isearch-style)
	skipDisplay bool          // Don't display completions if there are some.

	// Incremental search
	IsearchRegex       *regexp.Regexp // Holds the current search regex match
	isearchBuf         *core.Line     // The isearch minibuffer
	isearchCur         *core.Cursor   // Cursor position in the minibuffer.
	isearchName        string         // What is being incrementally searched for.
	isearchInsert      bool           // Whether to insert the first match in the line
	isearchForward     bool           // Match results in forward order, or backward.
	isearchSubstring   bool           // Match results as a substring (regex), or as a prefix.
	isearchReplaceLine bool           // Replace the current line with the search result
	isearchStartBuf    string         // The buffer before starting isearch
	isearchStartCursor int            // The cursor position before starting isearch
	isearchLast        string         // The last non-incremental buffer.
	isearchModeExit    keymap.Mode    // The main keymap to restore after exiting isearch
}

// NewEngine initializes a new completion engine with the shell operating parameters.
func NewEngine(h *ui.Hint, km *keymap.Engine, o *inputrc.Config) *Engine {
	return &Engine{
		config: o,
		hint:   h,
		keymap: km,
	}
}

// Init is used once at shell creation time to pass further parameters to the engine.
func Init(eng *Engine, k *core.Keys, l *core.Line, cur *core.Cursor, s *core.Selection, comp Completer) {
	eng.keys = k
	eng.line = l
	eng.cursor = cur
	eng.selection = s
	eng.compLine = l
	eng.compCursor = cur
	eng.autoCompleter = comp
}

// Generate uses a list of completions to group/order and prepares completions before printing them.
// If either no completions or only one is available after all constraints are applied, the engine
// will automatically insert/accept and/or reset itself.
func (e *Engine) Generate(completions Values) {
	e.prepare(completions)

	if e.noCompletions() {
		e.ClearMenu(true)
	}

	// Incremental search is a special case, because the user may
	// want to keep searching for another match, so we don't drop
	// the completion list and exit the incremental search mode.
	if e.hasUniqueCandidate() && e.keymap.Local() != keymap.Isearch {
		e.acceptCandidate()
		e.ClearMenu(true)
	}
}

// GenerateWith generates completions with a completer function, itself cached
// so that the next time it must update its results, it can reuse this completer.
func (e *Engine) GenerateWith(completer Completer) {
	e.cached = completer
	if e.cached == nil {
		return
	}

	// Call the provided/cached completer
	// and use the completions as normal
	e.Generate(e.cached())
}

// GenerateCached simply recomputes the grid of completions with the pool
// of completions already in memory. This might produce a bigger/smaller/
// different completion grid, for example if it's called on terminal resize.
func (e *Engine) GenerateCached() {
	e.GenerateWith(e.cached)
}

// SkipDisplay avoids printing completions below the
// input line, but still enables cycling through them.
func (e *Engine) SkipDisplay() {
	e.skipDisplay = true
}

// Select moves the completion selector by some X or Y value,
// and updates the inserted candidate in the input line.
func (e *Engine) Select(row, column int) {
	grp := e.currentGroup()

	if grp == nil || len(grp.rows) == 0 {
		return
	}

	// Ensure the completion keymaps are set.
	e.adjustSelectKeymap()

	// Some keys used to move around completions
	// will influence the coordinates' offsets.
	row, column = e.adjustCycleKeys(row, column)

	// If we already have an inserted candidate
	// remove it before inserting the new one.
	if len(e.selected.Value) > 0 {
		e.cancelCompletedLine()
	}

	defer e.refreshLine()

	// Move the selector
	done, next := grp.moveSelector(row, column)
	if !done {
		return
	}

	var newGrp *group

	if next {
		e.cycleNextGroup()
		newGrp = e.currentGroup()
		newGrp.firstCell()
	} else {
		e.cyclePreviousGroup()
		newGrp = e.currentGroup()
		newGrp.lastCell()
	}
}

// SelectTag allows to select the first value of the next tag (next=true),
// or the last value of the previous tag (next=false).
func (e *Engine) SelectTag(next bool) {
	// Ensure the completion keymaps are set.
	e.adjustSelectKeymap()

	if len(e.groups) <= 1 {
		return
	}

	// If the completion candidate is not empty,
	// it's also inserted in the line, so remove it.
	if len(e.selected.Value) > 0 {
		e.cancelCompletedLine()
	}

	// In the end we will update the line with the
	// newly/currently selected completion candidate.
	defer e.refreshLine()

	if next {
		e.cycleNextGroup()
		newGrp := e.currentGroup()
		newGrp.firstCell()
	} else {
		e.cyclePreviousGroup()
		newGrp := e.currentGroup()
		newGrp.firstCell()
	}
}

// Cancel exits the current completions with the following behavior:
// - If inserted is true, any inserted candidate is removed.
// - If cached is true, any cached completer function is dropped.
//
// This function does not exit the completion keymap, so
// any active completion menu will still be kept/displayed.
func (e *Engine) Cancel(inserted, cached bool) {
	if cached {
		e.cached = nil
		e.hint.Reset()
	}

	if len(e.selected.Value) == 0 && !inserted {
		return
	}

	defer e.cancelCompletedLine()

	// Either drop the inserted candidate,
	// or make it part of the real input line.
	if inserted {
		e.compLine.Set(*e.line...)
		e.compCursor.Set(e.cursor.Pos())
	} else {
		e.line.Set(*e.compLine...)
		e.cursor.Set(e.compCursor.Pos())
	}
}

// ResetForce drops any currently inserted candidate from the line,
// drops any cached completer function and generated list, and exits
// the incremental-search mode.
// All those steps are performed whether or not the engine is active.
// If revertLine is true, the line will be reverted to its original state.
func (e *Engine) ResetForce() {
	e.Cancel(!e.autoForce, true)
	e.ClearMenu(true)

	revertLine := e.keymap.Local() == keymap.Isearch ||
		e.keymap.Local() == keymap.MenuSelect

	e.IsearchStop(revertLine)
}

// Reset accepts the currently inserted candidate (if any), clears the current
// list of completions and exits the incremental-search mode if active.
// If the completion engine was not active to begin with, nothing will happen.
func (e *Engine) Reset() {
	e.autoForce = false
	if !e.IsActive() {
		e.ClearMenu(true)
		return
	}

	e.Cancel(false, true)
	e.ClearMenu(true)
	e.IsearchStop(false)
}

// ClearMenu exits the current completion keymap (if set) and clears
// the current list of generated completions (if completions is true).
func (e *Engine) ClearMenu(completions bool) {
	e.skipDisplay = false

	e.resetValues(completions, false)

	if e.keymap.Local() == keymap.MenuSelect {
		e.keymap.SetLocal("")
	}
}

// IsActive indicates if the engine is currently in possession of a
// non-empty list of generated completions (following all constraints).
func (e *Engine) IsActive() bool {
	completing := e.keymap.Local() == keymap.MenuSelect

	isearching := e.keymap.Local() == keymap.Isearch ||
		e.auto || e.autoForce

	nonIsearching, _, _ := e.NonIncrementallySearching()

	return (completing || isearching) && !nonIsearching
}

// IsInserting returns true if a candidate is currently virtually inserted.
func (e *Engine) IsInserting() bool {
	return e.selected.Value != ""
}

// Matches returns the number of completion candidates
// matching the current line/settings requirements.
func (e *Engine) Matches() int {
	comps, _ := e.completionCount()
	return comps
}

// Line returns the relevant input line at the time this function is called:
// if a candidate is currently selected, the line returned is the one containing
// the candidate. If no candidate is selected, the normal input line is returned.
// When the line returned is the completed one, the corresponding, adjusted cursor.
func (e *Engine) Line() (*core.Line, *core.Cursor) {
	if len(e.selected.Value) > 0 {
		return e.compLine, e.compCursor
	}

	return e.line, e.cursor
}

// Autocomplete generates the correct completions in autocomplete mode.
// We don't do it when we are currently in the completion keymap,
// since that means completions have already been computed.
func (e *Engine) Autocomplete() {
	e.auto = e.needsAutoComplete()

	// Clear the current completion list when we are at the
	// beginning of the line, and not currently completing.
	if e.auto || (!e.IsActive() && e.cursor.Pos() == 0) {
		e.resetValues(true, false)
	}

	// We are not auto when either: autocomplete is disabled,
	// incremental-search mode is active, or a completion is
	// currently selected in the menu.
	if !e.auto {
		return
	}

	// Regenerate the completions.
	if e.cached != nil {
		e.prepare(e.cached())
	} else if e.autoCompleter != nil {
		e.prepare(e.autoCompleter())
	}
}

// AutocompleteForce forces as-you-type autocomplete on the
// real input line, even if the current cursor position is 0.
func (e *Engine) AutocompleteForce() {
	e.autoForce = true
}

// AutoCompleting returns true if the completion engine is an
// autocompletion mode that has been triggered by a particular
// command (like history-search-forward/backward).
func (e *Engine) AutoCompleting() bool {
	return e.keymap.Local() == keymap.Isearch || e.autoForce
}
