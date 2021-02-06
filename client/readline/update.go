package readline

import (
	"strings"
)

func moveCursorUp(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dA", i)
}

func moveCursorDown(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dB", i)
}

func moveCursorForwards(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dC", i)
}

func moveCursorBackwards(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dD", i)
}

// moveCursorToLinePos - Must calculate the length of the prompt, realtime
// and for all contexts/needs, and move the cursor appropriately
func moveCursorToLinePos(rl *Instance) {
	moveCursorForwards(rl.promptLen + rl.pos)
	return
}

func (rl *Instance) moveCursorByAdjust(adjust int) {
	switch {
	case adjust > 0:
		moveCursorForwards(adjust)
		rl.pos += adjust
	case adjust < 0:
		moveCursorBackwards(adjust * -1)
		rl.pos += adjust
	}

	if rl.modeViMode != vimInsert && rl.pos == len(rl.line) && len(rl.line) > 0 {
		moveCursorBackwards(1)
		rl.pos--
	}
}

func (rl *Instance) insert(r []rune) {
	for {
		// I don't really understand why `0` is creaping in at the end of the
		// array but it only happens with unicode characters.
		if len(r) > 1 && r[len(r)-1] == 0 {
			r = r[:len(r)-1]
			continue
		}
		break
	}

	switch {
	case len(rl.line) == 0:
		rl.line = r
	case rl.pos == 0:
		rl.line = append(r, rl.line...)
	case rl.pos < len(rl.line):
		r := append(r, rl.line[rl.pos:]...)
		rl.line = append(rl.line[:rl.pos], r...)
	default:
		rl.line = append(rl.line, r...)
	}

	rl.echo()

	rl.pos += len(r)
	moveCursorForwards(len(r) - 1)

	if rl.modeViMode == vimInsert {
		rl.updateHelpers()
	}
}

func (rl *Instance) backspace() {
	if len(rl.line) == 0 || rl.pos == 0 {
		return
	}

	moveCursorBackwards(1)
	rl.pos--
	rl.delete()
}

func (rl *Instance) delete() {
	switch {
	case len(rl.line) == 0:
		return
	case rl.pos == 0:
		rl.line = rl.line[1:]
		rl.echo()
		moveCursorBackwards(1)
	case rl.pos > len(rl.line):
		rl.backspace()
	case rl.pos == len(rl.line):
		rl.line = rl.line[:rl.pos]
		rl.echo()
		moveCursorBackwards(1)
	default:
		rl.line = append(rl.line[:rl.pos], rl.line[rl.pos+1:]...)
		rl.echo()
		moveCursorBackwards(1)
	}

	rl.updateHelpers()
}

func (rl *Instance) echo() {

	// We move the cursor back to the very beginning of the line:
	// prompt + cursor position
	moveCursorBackwards(rl.promptLen + rl.pos)

	switch {
	case rl.PasswordMask > 0:
		print(strings.Repeat(string(rl.PasswordMask), len(rl.line)) + " ")

	case rl.SyntaxHighlighter == nil:
		print(string(rl.mlnPrompt))

		// Depending on the presence of a virtually completed item,
		// print either the virtual line or the real one.
		if len(rl.currentComp) > 0 {
			line := rl.lineComp[:rl.pos]
			line = append(line, rl.lineRemain...)
			print(string(line) + " ")
		} else {
			print(string(rl.line) + " ")
			moveCursorBackwards(len(rl.line) - rl.pos)
		}

	default:
		print(string(rl.mlnPrompt))

		// Depending on the presence of a virtually completed item,
		// print either the virtual line or the real one.
		if len(rl.currentComp) > 0 {
			line := rl.lineComp[:rl.pos]
			line = append(line, rl.lineRemain...)
			print(rl.SyntaxHighlighter(line) + " ")
		} else {
			print(rl.SyntaxHighlighter(rl.line) + " ")
			moveCursorBackwards(len(rl.line) - rl.pos)
		}
	}

	// moveCursorBackwards(len(rl.line) - rl.pos)
}

func (rl *Instance) clearLine() {
	if len(rl.line) == 0 {
		return
	}

	var lineLen int
	if len(rl.lineComp) > len(rl.line) {
		lineLen = len(rl.lineComp)
	} else {
		lineLen = len(rl.line)
	}

	moveCursorBackwards(rl.pos)
	print(strings.Repeat(" ", lineLen))
	moveCursorBackwards(lineLen)

	// Real input line
	rl.line = []rune{}
	rl.pos = 0

	// Completions are also reset
	rl.clearVirtualComp()
}

func (rl *Instance) resetHelpers() {
	rl.modeAutoFind = false
	rl.clearHelpers()
	rl.resetHintText()
	rl.resetTabCompletion()
}

func (rl *Instance) clearHelpers() {
	print("\r\n" + seqClearScreenBelow)
	moveCursorUp(1)
	moveCursorToLinePos(rl)

	// Reset some values
	rl.lineComp = []rune{}
	rl.currentComp = []rune{}
}

func (rl *Instance) renderHelpers() {

	rl.echo()

	// If we are waiting for confirmation (too many comps),
	// do not overwrite the confirmation question hint.
	if !rl.compConfirmWait {
		// We also don't overwrite if in tab find mode, which has a special hint.
		if !rl.modeAutoFind {
			rl.getHintText()
		}
		// We write the hint anyway
		rl.writeHintText()
	}

	rl.writeTabCompletion()
	moveCursorUp(rl.tcUsedY)

	if !rl.compConfirmWait {
		moveCursorUp(rl.hintY)
	}
	moveCursorBackwards(GetTermWidth())

	moveCursorToLinePos(rl)
}

// This one has the advantage of not stacking hints and completions, pretty balanced.
// However there is a problem with it when we use completion while being in the middle of the line.
// func (rl *Instance) renderHelpers() {
//
//         rl.echo() // Added by me, so that prompt always appear when new line
//
//         // If we are waiting for confirmation (too many comps), do not overwrite the hints.
//         if !rl.compConfirmWait {
//                 rl.getHintText()
//                 rl.writeHintText()
//                 moveCursorUp(rl.hintY)
//         }
//
//         rl.writeTabCompletion()
//         moveCursorUp(rl.tcUsedY)

//         moveCursorBackwards(GetTermWidth())
//         moveCursorToLinePos(rl)
// }

func (rl *Instance) updateHelpers() {
	rl.tcOffset = 0
	rl.getHintText()
	if rl.modeTabCompletion {
		rl.getTabCompletion()
	}
	rl.clearHelpers()
	rl.renderHelpers()
}
