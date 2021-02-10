package readline

import (
	"strconv"
	"strings"
)

// History is an interface to allow you to write your own history logging
// tools. eg sqlite backend instead of a file system.
// By default readline will just use the dummyLineHistory interface which only
// logs the history to memory ([]string to be precise).
type History interface {
	// Append takes the line and returns an updated number of lines or an error
	Write(string) (int, error)

	// GetLine takes the historic line number and returns the line or an error
	GetLine(int) (string, error)

	// Len returns the number of history lines
	Len() int

	// Dump returns everything in readline. The return is an interface{} because
	// not all LineHistory implementations will want to structure the history in
	// the same way. And since Dump() is not actually used by the readline API
	// internally, this methods return can be structured in whichever way is most
	// convenient for your own applications (or even just create an empty
	//function which returns `nil` if you don't require Dump() either)
	Dump() interface{}
}

// ExampleHistory is an example of a LineHistory interface:
type ExampleHistory struct {
	items []string
}

// Write to history
func (h *ExampleHistory) Write(s string) (int, error) {
	h.items = append(h.items, s)
	return len(h.items), nil
}

// GetLine returns a line from history
func (h *ExampleHistory) GetLine(i int) (string, error) {
	return h.items[i], nil
}

// Len returns the number of lines in history
func (h *ExampleHistory) Len() int {
	return len(h.items)
}

// Dump returns the entire history
func (h *ExampleHistory) Dump() interface{} {
	return h.items
}

// NullHistory is a null History interface for when you don't want line
// entries remembered eg password input.
type NullHistory struct{}

// Write to history
func (h *NullHistory) Write(s string) (int, error) {
	return 0, nil
}

// GetLine returns a line from history
func (h *NullHistory) GetLine(i int) (string, error) {
	return "", nil
}

// Len returns the number of lines in history
func (h *NullHistory) Len() int {
	return 0
}

// Dump returns the entire history
func (h *NullHistory) Dump() interface{} {
	return []string{}
}

// Browse historic lines:
func (rl *Instance) walkHistory(i int) {
	// Switch to correct history
	var history History
	if !rl.mainHist {
		history = rl.AltHistory
	} else {
		history = rl.History
	}

	switch rl.histPos + i {
	case -1, history.Len() + 1:
		return

	case history.Len():
		rl.clearLine()
		rl.histPos += i
		rl.line = []rune(rl.lineBuf)

	default:
		s, err := history.GetLine(rl.histPos + i)
		if err != nil {
			rl.resetHelpers()
			print("\r\n" + err.Error() + "\r\n")
			print(rl.prompt)
			return
		}

		if rl.histPos == history.Len() {
			rl.lineBuf = string(rl.line)
		}

		rl.clearLine()
		rl.histPos += i
		rl.line = []rune(s)
	}

	rl.echo()
	rl.pos = len(rl.line)
	if rl.pos > 1 {
		moveCursorForwards(rl.pos - 1)
	} else if rl.pos == 0 {
		moveCursorBackwards(1)
	}

	rl.updateHelpers()
}

// completeHistory - Populates a CompletionGroup with history and returns it the shell
// we populate only one group, so as to pass it to the main completion engine.
func (rl *Instance) completeHistory() (hist []*CompletionGroup) {

	hist = make([]*CompletionGroup, 1)
	hist[0] = &CompletionGroup{
		DisplayType: TabDisplayMap,
		MaxLength:   10,
	}

	// Switch to completion flux first
	var history History
	if !rl.mainHist {
		if rl.AltHistory == nil {
			return
		}
		history = rl.AltHistory
		hist[0].Name = "User history (all clients)"
		hist[0].Description = "All commands entered by the user, in all its consoles."
	} else {
		if rl.History == nil {
			return
		}
		history = rl.History
		hist[0].Name = "Console history"
		hist[0].Description = "All commands for this console only (identified by its ID)."
	}

	hist[0].init(rl)

	var (
		line string
		num  string
		err  error
	)

	rl.tcPrefix = string(rl.line) // We use the current full line for filtering

	for i := history.Len() - 1; i >= 0; i-- {
		line, err = history.GetLine(i)
		if err != nil {
			continue
		}

		if !strings.HasPrefix(line, rl.tcPrefix) {
			continue
		}

		line = strings.Replace(line, "\n", ` `, -1)[len(rl.line):]

		if hist[0].Descriptions[line] != "" {
			continue
		}

		hist[0].Suggestions = append(hist[0].Suggestions, line)
		num = strconv.Itoa(i)

		hist[0].Descriptions[line] = "\033[38;5;237m" + num + RESET
	}

	return
}
