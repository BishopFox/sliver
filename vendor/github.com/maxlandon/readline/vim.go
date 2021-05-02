package readline

import (
	"fmt"
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

var (
	// registerFreeKeys - Some Vim keys don't act on/ aren't affected by registers,
	// and using these keys will automatically cancel any active register.
	// NOTE: Don't forget to update if you add Vim bindings !!
	registerFreeKeys = []rune{'a', 'A', 'h', 'i', 'I', 'j', 'k', 'l', 'r', 'R', 'u', 'v', '$', '%', '[', ']'}

	// validRegisterKeys - All valid register IDs (keys) for read/write Vim registers
	validRegisterKeys = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/-\""
)

// vi - Apply a key to a Vi action. Note that as in the rest of the code, all cursor movements
// have been moved away, and only the rl.pos is adjusted: when echoing the input line, the shell
// will compute the new cursor pos accordingly.
func (rl *Instance) vi(r rune) {

	// Check if we are in register mode. If yes, and for some characters,
	// we select the register and exit this func immediately.
	if rl.registers.registerSelectWait {
		for _, char := range validRegisterKeys {
			if r == char {
				rl.registers.setActiveRegister(r)
				return
			}
		}
	}

	// If we are on register mode and one is already selected,
	// check if the key stroke to be evaluated is acting on it
	// or not: if not, we cancel the active register now.
	if rl.registers.onRegister {
		for _, char := range registerFreeKeys {
			if char == r {
				rl.registers.resetRegister()
			}
		}
	}

	// Then evaluate the key.
	switch r {
	case 'a':
		if len(rl.line) > 0 {
			rl.pos++
		}
		rl.modeViMode = vimInsert
		rl.viIteration = ""
		rl.viUndoSkipAppend = true

	case 'A':
		if len(rl.line) > 0 {
			rl.pos = len(rl.line)
		}
		rl.modeViMode = vimInsert
		rl.viIteration = ""
		rl.viUndoSkipAppend = true

	case 'b':
		if rl.viIsYanking {
			vii := rl.getViIterations()
			rl.saveToRegisterTokenize(tokeniseLine, rl.viJumpB, vii)
			rl.viIsYanking = false
			return
		}
		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpB(tokeniseLine))
		}

	case 'B':
		if rl.viIsYanking {
			vii := rl.getViIterations()
			rl.saveToRegisterTokenize(tokeniseSplitSpaces, rl.viJumpB, vii)
			rl.viIsYanking = false
			return
		}
		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpB(tokeniseSplitSpaces))
		}

	case 'd':
		rl.modeViMode = vimDelete
		rl.viUndoSkipAppend = true

	case 'D':
		rl.saveBufToRegister(rl.line[rl.pos-1:])
		rl.line = rl.line[:rl.pos]
		// Only go back if there is an input
		if len(rl.line) > 0 {
			rl.pos--
		}
		rl.resetHelpers()
		rl.updateHelpers()
		rl.viIteration = ""

	case 'e':
		if rl.viIsYanking {
			vii := rl.getViIterations()
			rl.saveToRegisterTokenize(tokeniseLine, rl.viJumpE, vii)
			rl.viIsYanking = false
			return
		}

		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpE(tokeniseLine))
		}

	case 'E':
		if rl.viIsYanking {
			vii := rl.getViIterations()
			rl.saveToRegisterTokenize(tokeniseSplitSpaces, rl.viJumpE, vii)
			rl.viIsYanking = false
			return
		}

		rl.viUndoSkipAppend = true
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpE(tokeniseSplitSpaces))
		}

	case 'h':
		if rl.pos > 0 {
			rl.pos--
		}
		rl.viUndoSkipAppend = true

	case 'i':
		rl.modeViMode = vimInsert
		rl.viIteration = ""
		rl.viUndoSkipAppend = true
		rl.registers.resetRegister()

	case 'I':
		rl.modeViMode = vimInsert
		rl.viIteration = ""
		rl.viUndoSkipAppend = true
		rl.pos = 0

	case 'j':
		// Set the main history as the one we navigate, by default
		rl.mainHist = true
		rl.walkHistory(-1)
	case 'k':
		// Set the main history as the one we navigate, by default
		rl.mainHist = true
		rl.walkHistory(1)

	case 'l':
		if (rl.modeViMode == vimInsert && rl.pos < len(rl.line)) ||
			(rl.modeViMode != vimInsert && rl.pos < len(rl.line)-1) {
			rl.pos++
		}
		rl.viUndoSkipAppend = true

	case 'p':
		// paste after the cursor position
		rl.viUndoSkipAppend = true
		rl.pos++

		buffer := rl.pasteFromRegister()
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.insert(buffer)
		}
		rl.pos--

	case 'P':
		// paste before
		rl.viUndoSkipAppend = true
		buffer := rl.pasteFromRegister()
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.insert(buffer)
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

		// Keep the previous cursor position
		prev := rl.pos

		new, err := rl.launchEditor(multiline)
		if err != nil || len(new) == 0 || string(new) == string(multiline) {
			fmt.Println(err)
			rl.viUndoSkipAppend = true
			return
		}

		// Clean the shell and put the new buffer, with adjusted pos if needed.
		rl.clearLine()
		rl.line = new
		if prev > len(rl.line) {
			rl.pos = len(rl.line) - 1
		} else {
			rl.pos = prev
		}

	case 'w':
		// If we were not yanking
		rl.viUndoSkipAppend = true
		// If the input line is empty, we don't do anything
		if rl.pos == 0 && len(rl.line) == 0 {
			return
		}

		// If we were yanking, we forge the new yank buffer
		// and return without moving the cursor.
		if rl.viIsYanking {
			vii := rl.getViIterations()
			rl.saveToRegisterTokenize(tokeniseLine, rl.viJumpW, vii)
			rl.viIsYanking = false
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

		if rl.viIsYanking {
			vii := rl.getViIterations()
			rl.saveToRegisterTokenize(tokeniseSplitSpaces, rl.viJumpW, vii)
			rl.viIsYanking = false
			return
		}
		vii := rl.getViIterations()
		for i := 1; i <= vii; i++ {
			rl.moveCursorByAdjust(rl.viJumpW(tokeniseSplitSpaces))
		}

	case 'x':
		vii := rl.getViIterations()

		// We might be on an active register, but not yanking...
		rl.saveToRegister(vii)

		// Delete the chars in the line anyway
		for i := 1; i <= vii; i++ {
			rl.deleteX()
		}
		if rl.pos == len(rl.line) && len(rl.line) > 0 {
			rl.pos--
		}

	case 'y':
		if rl.viIsYanking {
			rl.saveBufToRegister(rl.line)
			rl.viIsYanking = false
		}
		rl.viIsYanking = true
		rl.viUndoSkipAppend = true

	case 'Y':
		rl.saveBufToRegister(rl.line)
		rl.viUndoSkipAppend = true

	case '[':
		if rl.viIsYanking {
			rl.saveToRegister(rl.viJumpPreviousBrace())
			rl.viIsYanking = false
			return
		}
		rl.viUndoSkipAppend = true
		rl.moveCursorByAdjust(rl.viJumpPreviousBrace())

	case ']':
		if rl.viIsYanking {
			rl.saveToRegister(rl.viJumpNextBrace())
			rl.viIsYanking = false
			return
		}
		rl.viUndoSkipAppend = true
		rl.moveCursorByAdjust(rl.viJumpNextBrace())

	case '$':
		if rl.viIsYanking {
			rl.saveBufToRegister(rl.line[rl.pos:])
			rl.viIsYanking = false
			return
		}
		rl.pos = len(rl.line)
		rl.viUndoSkipAppend = true

	case '%':
		if rl.viIsYanking {
			rl.saveToRegister(rl.viJumpBracket())
			rl.viIsYanking = false
			return
		}
		rl.viUndoSkipAppend = true
		rl.moveCursorByAdjust(rl.viJumpBracket())

	case '"':
		// We might be on a register already, so reset it,
		// and then wait again for a new register ID.
		if rl.registers.onRegister {
			rl.registers.resetRegister()
		}
		rl.registers.registerSelectWait = true

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
	rl.updateHelpers()
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

func (rl *Instance) viJumpB(tokeniser tokeniser) (adjust int) {
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

func (rl *Instance) viJumpE(tokeniser tokeniser) (adjust int) {
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

func (rl *Instance) viJumpW(tokeniser tokeniser) (adjust int) {
	split, index, pos := tokeniser(rl.line, rl.pos)
	switch {
	case len(split) == 0:
		return
	case index+1 == len(split):
		adjust = len(rl.line) - rl.pos
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
