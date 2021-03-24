package readline

import (
	"strings"
)

func (rl *Instance) vimDelete(r []rune) {
	defer func() { rl.modeViMode = vimKeys }()

	switch r[0] {
	case 'b':
		rl.viDeleteByAdjust(rl.viJumpB(tokeniseLine))

	case 'B':
		rl.viDeleteByAdjust(rl.viJumpB(tokeniseSplitSpaces))

	case 'd':
		rl.clearLine()
		rl.resetHelpers()
		rl.getHintText()

	case 'e':
		rl.viDeleteByAdjust(rl.viJumpE(tokeniseLine) + 1)

	case 'E':
		rl.viDeleteByAdjust(rl.viJumpE(tokeniseSplitSpaces) + 1)

	case 'w':
		rl.viDeleteByAdjust(rl.viJumpW(tokeniseLine))

	case 'W':
		rl.viDeleteByAdjust(rl.viJumpW(tokeniseSplitSpaces))

	case '%':
		rl.viDeleteByAdjust(rl.viJumpBracket())

	case '$':
		rl.viDeleteByAdjust(len(rl.line) - rl.pos)

	case 27:
		if len(r) > 1 && '1' <= r[1] && r[1] <= '9' {
			if rl.vimDeleteToken(r[1]) {
				return
			}
		}
		fallthrough

	default:
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
		newLine = rl.line[:rl.pos]
		backOne = true
	case rl.pos+adjust == 0:
		newLine = rl.line[rl.pos:]
	case adjust < 0:
		newLine = append(rl.line[:rl.pos+adjust], rl.line[rl.pos:]...)
	default:
		newLine = append(rl.line[:rl.pos], rl.line[rl.pos+adjust:]...)
	}

	rl.line = newLine

	rl.updateHelpers()

	if adjust < 0 {
		rl.moveCursorByAdjust(adjust)
	}

	if backOne {
		rl.pos--
	}
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
