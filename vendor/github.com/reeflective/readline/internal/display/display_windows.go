//go:build windows
// +build windows

package display

import (
	"github.com/reeflective/readline/internal/core"
)

// WatchResize redisplays the interface on terminal resize events on Windows.
// Currently not implemented, see related issue in repo: too buggy right now.
func WatchResize(eng *Engine) chan<- bool {
	resizeChannel := core.GetTerminalResize(eng.keys)
	done := make(chan bool, 1)

	go func() {
		for {
			select {
			case <-resizeChannel:
				// Weird behavior on Windows: when there is no autosuggested line,
				// the cursor moves at the end of the completions area, if non-empty.
				// We must manually go back to the input cursor position first.
				// LAST UPDATE: 02/08/25: On Windows 10 terminal, this seems to have
				// disappeared.
				// line, _ := eng.completer.Line()
				// if eng.completer.IsInserting() {
				// 	eng.suggested = *eng.line
				// } else {
				// 	eng.suggested = eng.histories.Suggest(eng.line)
				// }
				//
				// if eng.suggested.Len() <= line.Len() {
				// 	fmt.Println(term.HideCursor)
				//
				// 	compRows := completion.Coordinates(eng.completer)
				// 	if compRows <= eng.AvailableHelperLines() {
				// 		compRows++
				// 	}
				//
				// 	term.MoveCursorBackwards(term.GetWidth())
				// 	term.MoveCursorUp(compRows)
				// 	term.MoveCursorUp(ui.CoordinatesHint(eng.hint))
				// 	eng.cursorHintToLineStart()
				// 	eng.lineStartToCursorPos()
				// 	fmt.Println(term.ShowCursor)
				// }
				//
				eng.completer.GenerateCached()
				eng.Refresh()
			case <-done:
				return
			}
		}
	}()

	return done
}
