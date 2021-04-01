package readline

import (
	"strconv"
)

// InputMode - The shell input mode
type InputMode int

const (
	// Vim - Vim editing mode
	Vim = iota
	// Emacs - Emacs (classic) editing mode
	Emacs
)

type viMode int

const (
	vimInsert viMode = iota
	vimReplaceOnce
	vimReplaceMany
	vimDelete
	vimKeys
)

const (
	vimInsertStr      = "[I]"
	vimReplaceOnceStr = "[V]"
	vimReplaceManyStr = "[R]"
	vimDeleteStr      = "[D]"
	vimKeysStr        = "[N]"
)

// vi - Apply a key to a Vi action. Note that as in the rest of the code, all cursor movements
// have been moved away, and only the rl.pos is adjusted: when echoing the input line, the shell
// will compute the new cursor pos accordingly.
func (rl *Instance) vi(r rune) {

	switch r {
	case 'a':
		if len(rl.line) > 0 {
			// moveCursorForwards(1)
			rl.pos++
		}
		rl.modeViMode = vimInsert
		rl.viIteration = ""
		rl.viUndoSkipAppend = true

	case 'A':
		if len(rl.line) > 0 {
			// moveCursorForwards(len(rl.line) - rl.pos)
			rl.pos = len(rl.line)
		}
		rl.modeViMode = vimInsert
		rl.viIteration = ""
		rl.viUndoSkipAppend = true

	case 'b':
		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpB(tokeniseLine))
		}

	case 'B':
		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpB(tokeniseSplitSpaces))
		}

	case 'd':
		rl.modeViMode = vimDelete
		rl.viUndoSkipAppend = true

	case 'D':
		// moveCursorBackwards(rl.pos)
		// print(strings.Repeat(" ", len(rl.line)))

		// moveCursorBackwards(len(rl.line) - rl.pos)
		rl.line = rl.line[:rl.pos]
		// rl.echo()

		// moveCursorBackwards(2)
		rl.pos--
		rl.updateHelpers()
		rl.viIteration = ""

	case 'e':
		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpE(tokeniseLine))
		}

	case 'E':
		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpE(tokeniseSplitSpaces))
		}

	case 'h':
		if rl.pos > 0 {
			// moveCursorBackwards(1)
			rl.pos--
		}
		rl.viUndoSkipAppend = true

	case 'i':
		rl.modeViMode = vimInsert
		rl.viIteration = ""
		rl.viUndoSkipAppend = true

	case 'I':
		rl.modeViMode = vimInsert
		rl.viIteration = ""
		rl.viUndoSkipAppend = true
		// moveCursorBackwards(rl.pos)
		rl.pos = 0

	case 'l':
		if (rl.modeViMode == vimInsert && rl.pos < len(rl.line)) ||
			(rl.modeViMode != vimInsert && rl.pos < len(rl.line)-1) {
			// moveCursorForwards(1)
			rl.pos++
		}
		rl.viUndoSkipAppend = true

	case 'p':
		// paste after
		rl.viUndoSkipAppend = true
		rl.pos++
		// moveCursorForwards(1)
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.insert([]rune(rl.viYankBuffer))
		}
		rl.pos--
		// moveCursorBackwards(1)

	case 'P':
		// paste before
		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.insert([]rune(rl.viYankBuffer))
		}

	case 'r':
		rl.modeViMode = vimReplaceOnce
		rl.viIteration = ""
		rl.viUndoSkipAppend = true

	case 'R':
		rl.modeViMode = vimReplaceMany
		rl.viIteration = ""
		rl.viUndoSkipAppend = true

	case 'u':
		rl.undoLast()
		rl.viUndoSkipAppend = true

	case 'v':
		rl.clearHelpers()
		var multiline []rune
		if rl.GetMultiLine == nil {
			multiline = rl.line
		} else {
			multiline = rl.GetMultiLine(rl.line)
		}

		new, err := rl.launchEditor(multiline)
		if err != nil || len(new) == 0 || string(new) == string(multiline) {
			rl.viUndoSkipAppend = true
			return
		}
		rl.clearLine()
		rl.line = new

	case 'w':
		rl.viUndoSkipAppend = true
		// If the input line is empty, we don't do anything
		if rl.pos == 0 && len(rl.line) == 0 {
			return
		}
		// Else get iterations and move
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpW(tokeniseLine))
		}

	case 'W':
		// If the input line is empty, we don't do anything
		if rl.pos == 0 && len(rl.line) == 0 {
			return
		}
		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpW(tokeniseSplitSpaces))
		}

	case 'x':
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.delete()
		}
		if rl.pos == len(rl.line) && len(rl.line) > 0 {
			// moveCursorBackwards(1)
			rl.pos--
		}

	case 'y', 'Y':
		rl.viYankBuffer = string(rl.line)
		rl.viUndoSkipAppend = true

	case '[':
		rl.viUndoSkipAppend = true
		rl.moveCursorByAdjust(rl.viJumpPreviousBrace())

	case ']':
		rl.viUndoSkipAppend = true
		rl.moveCursorByAdjust(rl.viJumpNextBrace())

	case '$':
		// moveCursorForwards(len(rl.line) - rl.pos)
		rl.pos = len(rl.line)
		rl.viUndoSkipAppend = true

	case '%':
		rl.viUndoSkipAppend = true
		rl.moveCursorByAdjust(rl.viJumpBracket())

	case 'j':
		// Set the main history as the one we navigate, by default
		rl.mainHist = true
		rl.walkHistory(-1)
	case 'k':
		// Set the main history as the one we navigate, by default
		rl.mainHist = true
		rl.walkHistory(1)
	default:
		if r <= '9' && '0' <= r {
			rl.viIteration += string(r)
		}
		rl.viUndoSkipAppend = true
	}
}

func (rl *Instance) getViIterations() int {
	i, _ := strconv.Atoi(rl.viIteration)
	if i < 1 {
		i = 1
	}
	rl.viIteration = ""
	return i
}

func (rl *Instance) refreshVimStatus() {
	rl.computePrompt()
	// rl.clearHelpers()
	rl.updateHelpers()
	// rl.renderHelpers()
}

// viHintMessage - lmorg's way of showing Vim status is to overwrite the hint.
// Currently not used, as there is a possibility to show the current Vim mode in the prompt.
func (rl *Instance) viHintMessage() {
	switch rl.modeViMode {
	case vimKeys:
		rl.hintText = []rune("-- VIM KEYS -- (press `i` to return to normal editing mode)")
	case vimInsert:
		rl.hintText = []rune("-- INSERT --")
	case vimReplaceOnce:
		rl.hintText = []rune("-- REPLACE CHARACTER --")
	case vimReplaceMany:
		rl.hintText = []rune("-- REPLACE --")
	case vimDelete:
		rl.hintText = []rune("-- DELETE --")
	default:
		rl.getHintText()
	}

	rl.clearHelpers()
	rl.renderHelpers()
}

func (rl *Instance) viJumpB(tokeniser func([]rune, int) ([]string, int, int)) (adjust int) {
	split, index, pos := tokeniser(rl.line, rl.pos)
	switch {
	case len(split) == 0:
		return
	case index == 0 && pos == 0:
		return
	case pos == 0:
		adjust = len(split[index-1])
	default:
		adjust = pos
	}
	return adjust * -1
}

func (rl *Instance) viJumpE(tokeniser func([]rune, int) ([]string, int, int)) (adjust int) {
	split, index, pos := tokeniser(rl.line, rl.pos)
	if len(split) == 0 {
		return
	}

	word := rTrimWhiteSpace(split[index])

	switch {
	case len(split) == 0:
		return
	case index == len(split)-1 && pos >= len(word)-1:
		return
	case pos >= len(word)-1:
		word = rTrimWhiteSpace(split[index+1])
		adjust = len(split[index]) - pos
		adjust += len(word) - 1
	default:
		adjust = len(word) - pos - 1
	}
	return
}

func (rl *Instance) viJumpW(tokeniser func([]rune, int) ([]string, int, int)) (adjust int) {
	split, index, pos := tokeniser(rl.line, rl.pos)
	switch {
	case len(split) == 0:
		return
	case index+1 == len(split):
		if len(split) == 1 {
			// If there is only one word in input, don't add a useless backspace
			adjust = len(rl.line) - rl.pos
		} else {
			// Otherwise add it
			adjust = len(rl.line) - 1 - rl.pos
		}
	default:
		adjust = len(split[index]) - pos
	}
	return
}

func (rl *Instance) viJumpPreviousBrace() (adjust int) {
	if rl.pos == 0 {
		return 0
	}

	for i := rl.pos - 1; i != 0; i-- {
		if rl.line[i] == '{' {
			return i - rl.pos
		}
	}

	return 0
}

func (rl *Instance) viJumpNextBrace() (adjust int) {
	if rl.pos >= len(rl.line)-1 {
		return 0
	}

	for i := rl.pos + 1; i < len(rl.line); i++ {
		if rl.line[i] == '{' {
			return i - rl.pos
		}
	}

	return 0
}

func (rl *Instance) viJumpBracket() (adjust int) {
	split, index, pos := tokeniseBrackets(rl.line, rl.pos)
	switch {
	case len(split) == 0:
		return
	case pos == 0:
		adjust = len(split[index])
	default:
		adjust = pos * -1
	}
	return
}
