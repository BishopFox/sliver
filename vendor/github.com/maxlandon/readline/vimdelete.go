package readline

import (
	"strings"
)

// vimDelete -
func (rl *Instance) viDelete(r rune) {

	// We are allowed to type iterations after a delete ('d') command.
	// in which case we don't exit the delete mode. The next thing typed
	// will thus be dispatched back here (like "2d4 then w).
	if !(r <= '9' && '0' <= r) {
		defer func() { rl.modeViMode = vimKeys }()
	}

	switch r {
	case 'b':
		vii := rl.getViIterations()
		rl.saveToRegisterTokenize(tokeniseLine, rl.viJumpB, vii)
		for i := 1; i <= vii; i++ {
			rl.viDeleteByAdjust(rl.viJumpB(tokeniseLine))
		}

	case 'B':
		vii := rl.getViIterations()
		rl.saveToRegisterTokenize(tokeniseSplitSpaces, rl.viJumpB, vii)
		for i := 1; i <= vii; i++ {
			rl.viDeleteByAdjust(rl.viJumpB(tokeniseSplitSpaces))
		}

	case 'd':
		rl.saveBufToRegister(rl.line)
		rl.clearLine()
		rl.resetHelpers()
		rl.getHintText()

	case 'e':
		vii := rl.getViIterations()
		rl.saveToRegisterTokenize(tokeniseLine, rl.viJumpE, vii)
		for i := 1; i <= vii; i++ {
			rl.viDeleteByAdjust(rl.viJumpE(tokeniseLine) + 1)
		}

	case 'E':
		vii := rl.getViIterations()
		rl.saveToRegisterTokenize(tokeniseSplitSpaces, rl.viJumpE, vii)
		for i := 1; i <= vii; i++ {
			rl.viDeleteByAdjust(rl.viJumpE(tokeniseSplitSpaces) + 1)
		}

	case 'w':
		vii := rl.getViIterations()
		rl.saveToRegisterTokenize(tokeniseLine, rl.viJumpW, vii)
		for i := 1; i <= vii; i++ {
			rl.viDeleteByAdjust(rl.viJumpW(tokeniseLine))
		}

	case 'W':
		vii := rl.getViIterations()
		rl.saveToRegisterTokenize(tokeniseSplitSpaces, rl.viJumpW, vii)
		for i := 1; i <= vii; i++ {
			rl.viDeleteByAdjust(rl.viJumpW(tokeniseSplitSpaces))
		}

	case '%':
		rl.saveToRegister(rl.viJumpBracket())
		rl.viDeleteByAdjust(rl.viJumpBracket())

	case '$':
		rl.saveBufToRegister(rl.line[rl.pos:])
		rl.viDeleteByAdjust(len(rl.line) - rl.pos)
		// Only go back if there is an input
		if len(rl.line) > 0 {
			rl.pos--
		}

	case '[':
		rl.saveToRegister(rl.viJumpPreviousBrace())
		rl.viDeleteByAdjust(rl.viJumpPreviousBrace())

	case ']':
		rl.saveToRegister(rl.viJumpNextBrace())
		rl.viDeleteByAdjust(rl.viJumpNextBrace())

	default:
		if r <= '9' && '0' <= r {
			rl.viIteration += string(r)
		}
		rl.viUndoSkipAppend = true
	}
}

func (rl *Instance) viDeleteByAdjust(adjust int) {
	var (
		newLine []rune
		backOne bool
	)

	// Avoid doing anything if input line is empty.
	if len(rl.line) == 0 {
		return
	}

	switch {
	case adjust == 0:
		rl.viUndoSkipAppend = true
		return
	case rl.pos+adjust == len(rl.line)-1:
		// This case should normally happen only when we met ALL THOSE CONDITIONS:
		// - We are currently in Insert Mode
		// - Appending to the end of the line (the cusor pos is len(line) + 1)
		// - We just deleted a single-lettered word from the input line.
		//
		// We must therefore ake a little adjustment (the -1), otherwise this
		// single letter is kept in the input line while it should be deleted.
		newLine = rl.line[:rl.pos-1]
		if adjust != -1 {
			backOne = true
		}

	case rl.pos+adjust == 0:
		newLine = rl.line[rl.pos:]
	case adjust < 0:
		newLine = append(rl.line[:rl.pos+adjust], rl.line[rl.pos:]...)
	default:
		newLine = append(rl.line[:rl.pos], rl.line[rl.pos+adjust:]...)
	}

	rl.line = newLine

	// rl.updateHelpers()

	if adjust < 0 {
		rl.moveCursorByAdjust(adjust)
	}

	if backOne {
		rl.pos--
	}

	rl.updateHelpers()
}

func (rl *Instance) vimDeleteToken(r rune) bool {
	tokens, _, _ := tokeniseSplitSpaces(rl.line, 0)
	pos := int(r) - 48 // convert ASCII to integer
	if pos > len(tokens) {
		return false
	}

	s := string(rl.line)
	newLine := strings.Replace(s, tokens[pos-1], "", -1)
	if newLine == s {
		return false
	}

	rl.line = []rune(newLine)

	rl.updateHelpers()

	if rl.pos > len(rl.line) {
		rl.pos = len(rl.line) - 1
	}

	return true
}
