package readline

import (
	"strings"
)

type undoItem struct {
	line string
	pos  int
}

func (rl *Instance) undoAppendHistory() {
	defer func() { rl.viUndoSkipAppend = false }()

	if rl.viUndoSkipAppend {
		return
	}

	rl.viUndoHistory = append(rl.viUndoHistory, undoItem{
		line: string(rl.line),
		pos:  rl.pos,
	})
}

func (rl *Instance) undoLast() {
	var undo undoItem
	for {
		// fmt.Println("|", len(rl.viUndoHistory), "|")
		if len(rl.viUndoHistory) == 0 {
			return
		}
		undo = rl.viUndoHistory[len(rl.viUndoHistory)-1]
		rl.viUndoHistory = rl.viUndoHistory[:len(rl.viUndoHistory)-1]
		if string(undo.line) != string(rl.line) {
			break
		}
	}

	rl.clearHelpers()

	moveCursorBackwards(rl.pos)
	print(strings.Repeat(" ", len(rl.line)))
	moveCursorBackwards(len(rl.line))
	moveCursorForwards(undo.pos)

	rl.line = []rune(undo.line)
	rl.pos = undo.pos

	rl.echo()

	if rl.modeViMode != vimInsert && len(rl.line) > 0 && rl.pos == len(rl.line) {
		rl.pos--
		moveCursorBackwards(1)
	}

}
