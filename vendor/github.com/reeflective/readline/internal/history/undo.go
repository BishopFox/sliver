package history

import (
	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/core"
)

// lineHistory contains all state changes for a given input line,
// whether it is the current input line or one of the history ones.
type lineHistory struct {
	pos   int
	items []undoItem
}

type undoItem struct {
	line string
	pos  int
}

// Save saves the current line and cursor position as an undo state item.
// If this was called while the shell was in the middle of its undo history
// (eg. the caller has undone one or more times), all undone steps are dropped.
func (h *Sources) Save() {
	defer h.Reset()

	if h.skip {
		return
	}

	// Get the undo states for the current line.
	line := h.getLineHistory()
	if line == nil {
		return
	}

	// When the line is identical to the previous undo, we just update
	// the cursor position if it's a different one.
	if len(line.items) > 0 && line.items[len(line.items)-1].line == string(*h.line) {
		line.items[len(line.items)-1].pos = h.cursor.Pos()
		return
	}

	// When we add an item to the undo history, the history
	// is cut from the current undo hist position onwards.
	if line.pos > len(line.items) {
		line.pos = len(line.items)
	}

	line.items = line.items[:len(line.items)-line.pos]

	// Make a copy of the cursor and ensure its position.
	cur := core.NewCursor(h.line)
	cur.Set(h.cursor.Pos())
	cur.CheckCommand()

	// And save the item.
	line.items = append(line.items, undoItem{
		line: string(*h.line),
		pos:  cur.Pos(),
	})
}

// SkipSave will not save the current line when the target command is done
// (more precisely, the next call to h.Save() will have no effect).
// This function is not useful is most cases, as call to saves will efficiently
// compare the line with the last saved state, and will not add redundant ones.
func (h *Sources) SkipSave() {
	h.skip = true
}

// SaveWithCommand is only meant to be called in the main readline loop of the shell,
// and not from within commands themselves: it does the same job as Save(), but also
// keeps the command that has just been executed.
func (h *Sources) SaveWithCommand(bind inputrc.Bind) {
	h.last = bind
	h.Save()
}

// Undo restores the line and cursor position to their last saved state.
func (h *Sources) Undo() {
	h.skip = true
	h.undoing = true

	// Get the undo states for the current line.
	line := h.getLineHistory()
	if line == nil || len(line.items) == 0 {
		return
	}

	var undo undoItem

	// When undoing, we loop through preceding undo items
	// as long as they are identical to the current line.
	for {
		line.pos++

		// Exit if we reached the end.
		if line.pos > len(line.items) {
			line.pos = len(line.items)
			return
		}

		// Break as soon as we find a non-matching line.
		undo = line.items[len(line.items)-line.pos]
		if undo.line != string(*h.line) {
			break
		}
	}

	// Use the undo we found
	h.line.Set([]rune(undo.line)...)
	h.cursor.Set(undo.pos)
}

// Revert goes back to the initial state of the line, which is what it was
// like when the shell started reading user input. Note that this state might
// be a line that was inferred, accept-and-held from the previous readline run.
func (h *Sources) Revert() {
	line := h.getLineHistory()
	if line == nil || len(line.items) == 0 {
		return
	}

	// Reuse the first saved state.
	undo := line.items[0]

	h.line.Set([]rune(undo.line)...)
	h.cursor.Set(undo.pos)

	// And reset everything
	line.items = make([]undoItem, 0)

	h.Reset()
}

// Redo cancels an undo action if any has been made, or if
// at the begin of the undo history, restores the original
// line's contents as their were before starting undoing.
func (h *Sources) Redo() {
	h.skip = true
	h.undoing = true

	line := h.getLineHistory()
	if line == nil || len(line.items) == 0 {
		return
	}

	line.pos--

	if line.pos < 1 {
		return
	}

	undo := line.items[len(line.items)-line.pos]
	h.line.Set([]rune(undo.line)...)
	h.cursor.Set(undo.pos)
}

// Last returns the last command ran by the shell.
func (h *Sources) Last() inputrc.Bind {
	return h.last
}

// Pos returns the current position in the undo history, which is
// equal to its length minus the number of previous undo calls.
func (h *Sources) Pos() int {
	lh := h.getLineHistory()
	if lh == nil {
		return 0
	}

	return lh.pos
}

// Reset will reset the current position in the list
// of undo items, but will not delete any of them.
func (h *Sources) Reset() {
	h.skip = false

	line := h.getLineHistory()
	if line == nil {
		return
	}

	if !h.undoing {
		line.pos = 0
	}

	h.undoing = false
}

// Always returns a non-nil map, whether or not a history source is found.
func (h *Sources) getHistoryLineChanges() map[int]*lineHistory {
	history := h.Current()
	if history == nil {
		return map[int]*lineHistory{}
	}

	// Get the state changes of all history lines
	// for the current history source.
	source := h.names[h.sourcePos]

	hist := h.lines[source]
	if hist == nil {
		h.lines[source] = make(map[int]*lineHistory)
		hist = h.lines[source]
	}

	return hist
}

func (h *Sources) getLineHistory() *lineHistory {
	hist := h.getHistoryLineChanges()
	if hist == nil {
		return &lineHistory{}
	}

	// Compute the position of the current line in the history.
	linePos := -1

	history := h.Current()
	if h.hpos > -1 && history != nil {
		linePos = history.Len() - h.hpos
	}

	if hist[linePos] == nil {
		hist[linePos] = &lineHistory{}
	}

	// Return the state changes of the current line.
	return hist[linePos]
}

func (h *Sources) restoreLineBuffer() {
	h.hpos = -1

	hist := h.getHistoryLineChanges()
	if hist == nil {
		return
	}

	// Get the undo states for the line buffer
	// (the last one, not any of the history ones)
	lh := hist[h.hpos]
	if lh == nil || len(lh.items) == 0 {
		return
	}

	undo := lh.items[len(lh.items)-1]

	// Restore the line to the last known state.
	h.line.Set([]rune(undo.line)...)
	h.cursor.Set(undo.pos)
}
