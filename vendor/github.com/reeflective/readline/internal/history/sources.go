package history

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/completion"
	"github.com/reeflective/readline/internal/core"
	"github.com/reeflective/readline/internal/ui"
)

// Sources manages and serves all history sources for the current shell.
type Sources struct {
	// Shell parameters
	line   *core.Line
	cursor *core.Cursor
	hint   *ui.Hint
	config *inputrc.Config

	// History sources
	list       map[string]Source // Sources of history lines
	names      []string          // Names of histories stored in rl.histories
	maxEntries int               // Inputrc configured maximum number of entries.
	sourcePos  int               // The index of the currently used history
	hpos       int               // Index used for navigating the history lines with arrows/j/k
	cpos       int               // A temporary cursor position used when searching/moving around.

	// Line changes history
	skip    bool                            // Skip saving the current line state.
	undoing bool                            // The last command executed was an undo.
	last    inputrc.Bind                    // The last command being ran.
	lines   map[string]map[int]*lineHistory // Each line in each history source has its own buffer history.

	// Lines accepted
	infer      bool      // If the last command ran needs to infer the history line.
	accepted   bool      // The line has been accepted and must be returned.
	acceptHold bool      // Should we reuse the same accepted line on the next loop.
	acceptLine core.Line // The line to return to the caller.
	acceptErr  error     // An error to return to the caller.
}

// NewSources is a required constructor for the history sources manager type.
func NewSources(line *core.Line, cur *core.Cursor, hint *ui.Hint, opts *inputrc.Config) *Sources {
	sources := &Sources{
		// History sources
		list: make(map[string]Source),
		// Line history
		lines: make(map[string]map[int]*lineHistory),
		// Shell parameters
		line:   line,
		cursor: cur,
		cpos:   -1,
		hpos:   -1,
		hint:   hint,
		config: opts,
	}

	sources.names = append(sources.names, defaultSourceName)
	sources.list[defaultSourceName] = new(memory)

	// Inputrc settings.
	sources.maxEntries = opts.GetInt("history-size")
	sizeSet := opts.GetString("history-size") != ""

	if sources.maxEntries == 0 && !sizeSet {
		sources.maxEntries = -1
	} else if sources.maxEntries == 0 && sizeSet {
		sources.maxEntries = 500
	}

	return sources
}

// Init initializes the history sources positions and buffers
// at the start of each readline loop. If the last command asked
// to infer a command line from the history, it is performed now.
func Init(hist *Sources) {
	defer func() {
		hist.accepted = false
		hist.acceptLine = nil
		hist.acceptErr = nil
		hist.cpos = -1
	}()

	if hist.acceptHold {
		hist.hpos = -1
		hist.line.Set(hist.acceptLine...)
		hist.cursor.Set(hist.line.Len())

		return
	}

	if !hist.infer {
		hist.hpos = -1
		undoHist := hist.getHistoryLineChanges()
		undoHist[hist.hpos] = &lineHistory{}

		return
	}

	switch hist.hpos {
	case -1:
	case 0:
		hist.InferNext()
	default:
		hist.Walk(-1)
	}

	hist.infer = false
}

// Add adds a source of history lines bound to a given name (printed above this source when used).
// If the shell currently has only an in-memory (default) history source available, the call will
// drop this source and replace it with the provided one. Following calls add to the list.
func (h *Sources) Add(name string, hist Source) {
	if len(h.list) == 1 && h.names[0] == defaultSourceName {
		delete(h.list, defaultSourceName)
		h.names = make([]string, 0)
	}

	h.names = append(h.names, name)
	h.list[name] = hist
}

// AddFromFile adds a command history source from a file path.
// The name is used when using/searching the history source.
func (h *Sources) AddFromFile(name, file string) {
	hist := new(fileHistory)
	hist.file = file
	hist.lines, _ = openHist(file)

	h.Add(name, hist)
}

// Delete deletes one or more history source by name.
// If no arguments are passed, all currently bound sources are removed.
func (h *Sources) Delete(sources ...string) {
	if len(sources) == 0 {
		h.list = make(map[string]Source)
		h.names = make([]string, 0)

		return
	}

	for _, name := range sources {
		delete(h.list, name)

		for i, hname := range h.names {
			if hname == name {
				h.names = append(h.names[:i], h.names[i+1:]...)
				break
			}
		}
	}

	h.sourcePos = 0
	if !h.infer {
		h.hpos = -1
	}
}

// Walk goes to the next or previous history line in the active source.
// If at the beginning of the history, the first history line is kept.
// If at the end of it, the main input buffer and cursor position is restored.
func (h *Sources) Walk(pos int) {
	history := h.Current()

	if history == nil || history.Len() == 0 {
		return
	}

	// Can't go back further than the first line.
	if h.hpos == history.Len() && pos == 1 {
		return
	}

	// Save the current line buffer if we are leaving it.
	if h.hpos == -1 && pos > 0 {
		h.skip = false
		h.Save()
		h.cpos = -1
		h.hpos = 0
	}

	h.hpos += pos

	switch {
	case h.hpos < -1:
		h.hpos = -1
		return
	case h.hpos == 0:
		h.restoreLineBuffer()
		return
	case h.hpos > history.Len():
		h.hpos = history.Len()
	}

	var line string
	var err error

	// When there is an available change history for
	// this line, use it instead of the fetched line.
	if hist := h.getLineHistory(); hist != nil && len(hist.items) > 0 {
		line = hist.items[len(hist.items)-1].line
	} else if line, err = history.GetLine(history.Len() - h.hpos); err != nil {
		h.hint.Set(color.FgRed + "history error: " + err.Error())
		return
	}

	// Update line buffer and cursor position.
	h.setLineCursorMatch(line)
}

// Fetch fetches the history event at the provided
// index position and makes it the current buffer.
func (h *Sources) Fetch(pos int) {
	history := h.Current()

	if history == nil || history.Len() == 0 {
		return
	}

	if pos < 0 || pos >= history.Len() {
		return
	}

	line, err := history.GetLine(pos)
	if err != nil {
		h.hint.Set(color.FgRed + "history error: " + err.Error())
		return
	}

	h.setLineCursorMatch(line)
}

// GetLast returns the last saved history line in the active history source.
func (h *Sources) GetLast() string {
	history := h.Current()

	if history == nil || history.Len() == 0 {
		return ""
	}

	last, err := history.GetLine(history.Len() - 1)
	if err != nil {
		return ""
	}

	return last
}

// Cycle checks for the next history source (if any) and makes it the active one.
// The active one is used in completions, and all history-related commands.
// If next is false, the engine cycles to the previous source.
func (h *Sources) Cycle(next bool) {
	switch next {
	case true:
		h.sourcePos++

		if h.sourcePos == len(h.names) {
			h.sourcePos = 0
		}
	case false:
		h.sourcePos--

		if h.sourcePos < 0 {
			h.sourcePos = len(h.names) - 1
		}
	}
}

// OnLastSource returns true if the currently active
// history source is the last one in the list.
func (h *Sources) OnLastSource() bool {
	return h.sourcePos == len(h.names)-1
}

// Current returns the current/active history source.
func (h *Sources) Current() Source {
	if len(h.list) == 0 {
		return nil
	}

	return h.list[h.names[h.sourcePos]]
}

// Write writes the accepted input line to all available sources.
// If infer is true, the next history initialization will automatically insert the next
// history line event after the first match of the line, which one is then NOT written.
func (h *Sources) Write(infer bool) {
	if infer {
		h.infer = true
		return
	}

	line := string(*h.line)

	if len(strings.TrimSpace(line)) == 0 {
		return
	}

	for _, history := range h.list {
		if history == nil {
			continue
		}

		// Don't write it if the history source has reached
		// the maximum number of lines allowed (inputrc)
		if h.maxEntries == 0 || h.maxEntries >= history.Len() {
			continue
		}

		var err error

		// Don't write the line if it's identical to the last one.
		last, err := history.GetLine(history.Len() - 1)
		if err == nil && last != "" && strings.TrimSpace(last) == strings.TrimSpace(line) {
			return
		}

		// Save the line and notify through hints if an error raised.
		_, err = history.Write(line)
		if err != nil {
			h.hint.Set(color.FgRed + err.Error())
		}
	}
}

// Accept is used to signal the line has been accepted by the user and must be
// returned to the readline caller. If hold is true, the line is preserved
// and redisplayed on the next loop. If infer, the line is not written to
// the history, but preserved as a line to match against on the next loop.
// If infer is false, the line is automatically written to active sources.
func (h *Sources) Accept(hold, infer bool, err error) {
	h.accepted = true
	h.acceptHold = hold
	h.acceptLine = *h.line
	h.acceptErr = err

	// Write the line to the history sources only when the line is not
	// returned along with an error (generally, a CtrlC/CtrlD keypress).
	if err == nil {
		h.Write(infer)
	}
}

// LineAccepted returns true if the user has accepted the line, signaling
// that the shell must return from its loop. The error can be nil, but may
// indicate a CtrlC/CtrlD style error.
func (h *Sources) LineAccepted() (bool, string, error) {
	if !h.accepted {
		return false, "", nil
	}

	line := string(h.acceptLine)

	// Revert all state changes to all lines.
	if h.config.GetBool("revert-all-at-newline") {
		for source := range h.lines {
			h.lines[source] = make(map[int]*lineHistory)
		}
	}

	return true, line, h.acceptErr
}

// InsertMatch replaces the buffer with the first history line matching the
// provided buffer, either as a substring (if regexp is true), or as a prefix.
// If the line argument is nil, the current line buffer is used to match against.
func (h *Sources) InsertMatch(line *core.Line, cur *core.Cursor, usePos, fwd, regexp bool) {
	if len(h.list) == 0 {
		return
	}

	if h.Current() == nil {
		return
	}

	// When the provided line is empty, we must use
	// the last known state of the main input line.
	line, cur = h.getLine(line, cur)
	preservePoint := cur.Pos() != 0

	// Don't go back to the beginning of
	// history if we are at the end of it.
	if fwd && h.hpos <= -1 {
		h.hpos = -1
		return
	}

	match, pos, found := h.match(line, cur, usePos, fwd, regexp)

	// If no match was found, return anyway, but if we were going forward
	// (down to the current input line), reinstore the main line buffer.
	if !found {
		if fwd {
			h.hpos = -1
			h.Undo()
		}

		return
	}

	// Update the line/cursor, and save the history position
	h.hpos = h.Current().Len() - pos
	h.line.Set([]rune(match)...)

	if preservePoint {
		h.cursor.Set(cur.Pos())
	} else {
		h.cursor.Set(h.line.Len())
	}
}

// InferNext finds a line matching the current line in the history,
// then finds the line event following it and, if any, inserts it.
func (h *Sources) InferNext() {
	if len(h.list) == 0 {
		return
	}

	history := h.Current()
	if history == nil {
		return
	}

	_, pos, found := h.match(h.line, nil, false, false, false)
	if !found {
		return
	}

	// If we have no match we return, or check for the next line.
	if history.Len() <= (history.Len()-pos)+1 {
		return
	}

	// Insert the next line
	line, err := history.GetLine(pos + 1)
	if err != nil {
		return
	}

	h.line.Set([]rune(line)...)
	h.cursor.Set(h.line.Len())
}

// Suggest returns the first line matching the current line buffer,
// so that caller can use for things like history autosuggestion.
// If no line matches the current line, it will return the latter.
func (h *Sources) Suggest(line *core.Line) core.Line {
	if len(h.list) == 0 || len(*line) == 0 {
		return *line
	}

	if h.Current() == nil {
		return *line
	}

	suggested, _, found := h.match(line, nil, false, false, false)
	if !found {
		return *line
	}

	return core.Line([]rune(suggested))
}

// Complete returns completions with the current history source values.
// If forward is true, the completions are proposed from the most ancient
// line in the history source to the most recent. If filter is true,
// only lines that match the current input line as a prefix are given.
func Complete(h *Sources, forward, filter bool, maxLines int, regex *regexp.Regexp) completion.Values {
	if len(h.list) == 0 {
		return completion.Values{}
	}

	history := h.Current()
	if history == nil {
		return completion.Values{}
	}

	h.hint.Set(color.Bold + color.FgCyanBright + h.names[h.sourcePos] + color.Reset)

	compLines := make([]completion.Candidate, 0)
	printedLines := make([]string, 0)

	// Set up iteration clauses
	var (
		histPos int
		done    func(i int) bool
		move    func(inc int) int
	)

	if forward {
		histPos = -1
		done = func(i int) bool { return i < history.Len()-1 && maxLines >= 0 }
		move = func(pos int) int { return pos + 1 }
	} else {
		histPos = history.Len()
		done = func(i int) bool { return i > 0 && maxLines >= 0 }
		move = func(pos int) int { return pos - 1 }
	}

	// And generate the completions.
	for done(histPos) {
		histPos = move(histPos)

		line, err := history.GetLine(histPos)
		if err != nil {
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		if filter && !strings.HasPrefix(line, string(*h.line)) {
			continue
		} else if regex != nil && !regex.MatchString(line) {
			continue
		}

		// If this history line is a duplicate of an existing one,
		// remove the existing one and keep this one as it's more recent.
		if yes, pos := contains(printedLines, line); yes {
			printedLines = append(printedLines[:pos], printedLines[pos+1:]...)
			printedLines = append(printedLines, line)

			continue
		}

		// Add to the list of printed lines if we have a new one.
		printedLines = append(printedLines, line)

		display := strings.ReplaceAll(line, "\n", ` `)

		// Proper pad for indexes
		indexStr := strconv.Itoa(histPos)
		pad := strings.Repeat(" ", len(strconv.Itoa(history.Len()))-len(indexStr))
		display = fmt.Sprintf("%s%s %s%s", color.Dim, indexStr+pad, color.DimReset, display)

		value := completion.Candidate{
			Display: display,
			Value:   line,
		}

		compLines = append(compLines, value)

		maxLines--
	}

	comps := completion.AddRaw(compLines)
	comps.NoSort["*"] = true
	comps.ListLong["*"] = true
	comps.PREFIX = string(*h.line)

	return comps
}

// Name returns the name of the currently active history source.
func (h *Sources) Name() string {
	return h.names[h.sourcePos]
}

func (h *Sources) match(match *core.Line, cur *core.Cursor, usePos, fwd, regex bool) (line string, pos int, found bool) {
	if len(h.list) == 0 {
		return line, pos, found
	}

	history := h.Current()
	if history == nil {
		return line, pos, found
	}

	// Set up iteration clauses
	var histPos int
	var done func(i int) bool
	var move func(inc int) int

	if fwd {
		histPos = -1
		done = func(i int) bool { return i < history.Len() }
		move = func(pos int) int { return pos + 1 }
	} else {
		histPos = history.Len()
		done = func(i int) bool { return i > 0 }
		move = func(pos int) int { return pos - 1 }
	}

	if usePos && h.hpos > -1 {
		histPos = history.Len() - h.hpos
	}

	for done(histPos) {
		// Fetch the next/prev line and adapt its length.
		histPos = move(histPos)

		histline, err := history.GetLine(histPos)
		if err != nil {
			return line, pos, found
		}

		cline := string(*match)
		if cur != nil && cur.Pos() < match.Len() {
			cline = cline[:cur.Pos()]
		}

		// Matching: either as substring (regex) or since beginning.
		switch {
		case regex:
			regexLine, err := regexp.Compile(regexp.QuoteMeta(cline))
			if err != nil {
				continue
			}

			// Go to next line if not matching as a substring.
			if !regexLine.MatchString(histline) {
				continue
			}

		default:
			// If too short or if not fully matching
			if len(histline) < len(cline) || (len(cline) > 0 && !strings.HasPrefix(histline, cline)) {
				continue
			}
		}

		// Else we have our history match.
		return histline, histPos, true
	}

	// We should have returned a match from the loop.
	return "", 0, false
}

// use the "main buffer" and its cursor if no line/cursor has been provided to match against.
func (h *Sources) getLine(line *core.Line, cur *core.Cursor) (*core.Line, *core.Cursor) {
	if h.hpos == -1 {
		skip := h.skip
		h.skip = false
		h.Save()
		h.skip = skip
	}

	if line == nil {
		line = new(core.Line)
		cur = core.NewCursor(line)

		hist := h.getHistoryLineChanges()
		if hist == nil {
			return line, cur
		}

		lh := hist[0]
		if lh == nil || len(lh.items) == 0 {
			return line, cur
		}

		undo := lh.items[len(lh.items)-1]
		line.Set([]rune(undo.line)...)
		cur.Set(undo.pos)
	}

	if cur == nil {
		cur = core.NewCursor(line)
	}

	return line, cur
}

// when walking down/up the lines (not search-matching them),
// this updates the buffer and the cursor position.
func (h *Sources) setLineCursorMatch(next string) {
	// Save the current cursor position when not saved before.
	if h.cpos == -1 && h.line.Len() > 0 && h.cursor.Pos() <= h.line.Len() {
		h.cpos = h.cursor.Pos()
	}

	h.line.Set([]rune(next)...)

	// Set cursor depending on inputrc options and line length.
	if h.config.GetBool("history-preserve-point") && h.line.Len() > h.cpos && h.cpos != -1 {
		h.cursor.Set(h.cpos)
	} else {
		h.cursor.Set(h.line.Len())
	}
}

func contains(s []string, e string) (bool, int) {
	for i, a := range s {
		if a == e {
			return true, i
		}
	}

	return false, 0
}

func removeDuplicates(source []string) []string {
	list := []string{}

	for _, item := range source {
		if no, _ := contains(list, item); no {
			list = append(list, item)
		}
	}

	return list
}
