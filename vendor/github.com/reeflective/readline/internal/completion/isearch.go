package completion

import (
	"regexp"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/core"
	"github.com/reeflective/readline/internal/keymap"
)

// IsearchStart starts incremental search (fuzzy-finding)
// with values matching the isearch minibuffer as a regexp.
func (e *Engine) IsearchStart(name string, autoinsert, replaceLine bool) {
	// Prepare all buffers and cursors.
	e.isearchInsert = autoinsert
	e.isearchReplaceLine = replaceLine

	e.isearchStartBuf = string(*e.line)
	e.isearchStartCursor = e.cursor.Pos()

	e.isearchBuf = new(core.Line)
	e.isearchCur = core.NewCursor(e.isearchBuf)

	// Prepare all keymaps and modes.
	e.auto = true
	e.keymap.SetLocal(keymap.Isearch)
	e.adaptIsearchInsertMode()

	// Hints
	e.isearchName = name
	e.hint.Set(color.Bold + color.FgCyan + e.isearchName + " (isearch): " + color.Reset + string(*e.isearchBuf))
}

// IsearchStop exists the incremental search mode,
// and drops the currently used regexp matcher.
// If revertLine is true, the original line is restored.
func (e *Engine) IsearchStop(revertLine bool) {
	// Reset all buffers and cursors.
	e.isearchBuf = nil
	e.IsearchRegex = nil
	e.isearchCur = nil

	// Reset the original line when needed.
	if e.isearchReplaceLine && revertLine {
		e.line.Set([]rune(e.isearchStartBuf)...)
		e.cursor.Set(e.isearchStartCursor)
	}

	e.isearchStartBuf = ""
	e.isearchStartCursor = 0
	e.isearchReplaceLine = false

	// And clear all related completion keymaps/modes.
	e.auto = false
	e.autoForce = false
	e.keymap.SetLocal("")
	e.resetIsearchInsertMode()
}

// GetBuffer returns the correct input line buffer (and its cursor/
// selection) depending on the context and active components:
// - If in non/incremental-search mode, the minibuffer.
// - If a completion is currently inserted, the completed line.
// - If neither of the above, the normal input line.
func (e *Engine) GetBuffer() (*core.Line, *core.Cursor, *core.Selection) {
	// Non/Incremental search buffer
	searching, _, _ := e.NonIncrementallySearching()

	if e.keymap.Local() == keymap.Isearch || searching {
		selection := core.NewSelection(e.isearchBuf, e.isearchCur)
		return e.isearchBuf, e.isearchCur, selection
	}

	// Completed line (with inserted candidate)
	if len(e.selected.Value) > 0 {
		return e.compLine, e.compCursor, e.selection
	}

	// Or completer inactive, normal input line.
	return e.line, e.cursor, e.selection
}

// UpdateIsearch recompiles the isearch buffer as a regex and
// filters matching candidates in the available completions.
func (e *Engine) UpdateIsearch() {
	searching, _, _ := e.NonIncrementallySearching()

	if e.keymap.Local() != keymap.Isearch && !searching {
		return
	}

	// If we have a virtually inserted candidate, it's because the
	// last action was a tab-complete selection: we don't need to
	// refresh the list of matches, as the minibuffer did not change,
	// and because it would make our currently selected comp to drop.
	if len(e.selected.Value) > 0 {
		return
	}

	// Update helpers depending on the search/minibuffer mode.
	if e.keymap.Local() == keymap.Isearch {
		e.updateIncrementalSearch()
	} else {
		e.updateNonIncrementalSearch()
	}
}

// NonIsearchStart starts a non-incremental, fake search mode:
// it does not produce or tries to match against completions,
// but uses a minibuffer similarly to incremental search mode.
func (e *Engine) NonIsearchStart(name string, repeat, forward, substring bool) {
	if repeat {
		e.isearchBuf = new(core.Line)
		e.isearchBuf.Set([]rune(e.isearchLast)...)
	} else {
		e.isearchBuf = new(core.Line)
	}

	e.isearchCur = core.NewCursor(e.isearchBuf)
	e.isearchCur.Set(e.isearchBuf.Len())

	e.isearchName = name
	e.isearchForward = forward
	e.isearchSubstring = substring

	e.keymap.NonIncrementalSearchStart()
	e.adaptIsearchInsertMode()
}

// NonIsearchStop exits the non-incremental search mode.
func (e *Engine) NonIsearchStop() {
	e.isearchLast = string(*e.isearchBuf)
	e.isearchBuf = nil
	e.IsearchRegex = nil
	e.isearchCur = nil
	e.isearchForward = false
	e.isearchSubstring = false

	// Reset keymap and helpers
	e.keymap.NonIncrementalSearchStop()
	e.resetIsearchInsertMode()
	e.hint.Reset()
}

// NonIncrementallySearching returns true if the completion engine
// is currently using a minibuffer for non-incremental search mode.
func (e *Engine) NonIncrementallySearching() (searching, forward, substring bool) {
	searching = e.isearchCur != nil && e.keymap.Local() != keymap.Isearch
	forward = e.isearchForward
	substring = e.isearchSubstring

	return
}

func (e *Engine) updateIncrementalSearch() {
	var regexStr string
	if hasUpper(*e.isearchBuf) {
		regexStr = string(*e.isearchBuf)
	} else {
		regexStr = "(?i)" + string(*e.isearchBuf)
	}

	var err error
	e.IsearchRegex, err = regexp.Compile(regexStr)

	if err != nil {
		e.hint.Set(color.FgRed + "Failed to compile i-search regexp")
	}

	// Refresh completions with the current minibuffer as a filter.
	e.GenerateWith(e.cached)

	// And filter out the completions.
	for _, g := range e.groups {
		g.updateIsearch(e)
	}

	// Update the hint section.
	isearchHint := color.Bold + color.FgCyan + e.isearchName + " (inc-search)"

	if e.Matches() == 0 {
		isearchHint += color.Reset + color.Bold + color.FgRed + " (no matches)"
	}

	isearchHint += ": " + color.Reset + color.Bold + string(*e.isearchBuf) + color.Reset + "_"

	e.hint.Set(isearchHint)

	// And update the inserted candidate if autoinsert is enabled.
	if e.isearchInsert && e.Matches() > 0 && e.isearchBuf.Len() > 0 {
		// History incremental searches must replace the whole line.
		if e.isearchReplaceLine {
			e.prefix = ""
			e.line.Set()
			e.cursor.Set(0)
		}

		e.Select(1, 0)
	} else if e.isearchReplaceLine {
		// Else no matches, restore the original line.
		e.line.Set([]rune(e.isearchStartBuf)...)
		e.cursor.Set(e.isearchStartCursor)
	}
}

func (e *Engine) updateNonIncrementalSearch() {
	isearchHint := color.Bold + color.FgCyan + e.isearchName +
		" (non-inc-search): " + color.Reset + color.Bold + string(*e.isearchBuf) + color.Reset + "_"
	e.hint.Set(isearchHint)
}

func (e *Engine) adaptIsearchInsertMode() {
	e.isearchModeExit = e.keymap.Main()

	if !e.keymap.IsEmacs() && e.keymap.Main() != keymap.ViInsert {
		e.keymap.SetMain(keymap.ViInsert)
	}
}

func (e *Engine) resetIsearchInsertMode() {
	if e.isearchModeExit == "" {
		return
	}

	if e.keymap.Main() != e.isearchModeExit {
		e.keymap.SetMain(string(e.isearchModeExit))
		e.isearchModeExit = ""
	}

	if e.keymap.Main() == keymap.ViCommand {
		e.cursor.CheckCommand()
	}
}
