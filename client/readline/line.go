package readline

import (
	"strings"
)

func (rl *Instance) echo() {

	// Then we print the prompt, and the line,
	switch {
	case rl.PasswordMask > 0:
		print(strings.Repeat(string(rl.PasswordMask), len(rl.line)) + " ")

	default:
		// Go back to prompt position, and clear everything below
		moveCursorBackwards(GetTermWidth())
		moveCursorUp(rl.posY)
		print(seqClearScreenBelow)

		// Print the prompt
		print(string(rl.mlnPrompt))

		// Assemble the line, taking virtual completions into account
		var line []rune
		if len(rl.currentComp) > 0 {
			line = rl.lineComp
		} else {
			line = rl.line
		}

		// Print the input line with optional syntax highlighting
		if rl.SyntaxHighlighter != nil {
			print(rl.SyntaxHighlighter(line) + " ")
		} else {
			print(string(line) + " ")
		}

		// Update references with new coordinates only now, because
		// the new line may be longer/shorter than the previous one.
		rl.updateReferences()

		// Go back to the current cursor position, with new coordinates
		moveCursorBackwards(GetTermWidth())
		moveCursorUp(rl.fullY)
		moveCursorDown(rl.posY)
		moveCursorForwards(rl.posX)
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

	// We can ONLY have three fondamentally different cases:
	switch {
	// The line is empty
	case len(rl.line) == 0:
		rl.line = r

	// We are inserting somewhere in the middle
	case rl.pos < len(rl.line):
		r := append(r, rl.line[rl.pos:]...)
		rl.line = append(rl.line[:rl.pos], r...)

	// We are at the end of the input line
	case rl.pos == len(rl.line):
		rl.line = append(rl.line, r...)
	}

	rl.pos += len(r)

	// This should also update the rl.pos
	rl.updateHelpers()
}

func (rl *Instance) delete() {
	switch {
	case len(rl.line) == 0:
		return
	case rl.pos == 0:
		rl.line = rl.line[1:]
	case rl.pos > len(rl.line):
		rl.backspace()
	case rl.pos == len(rl.line):
		rl.pos--
		rl.line = rl.line[:rl.pos]
	default:
		rl.line = append(rl.line[:rl.pos], rl.line[rl.pos+1:]...)
	}

	rl.updateHelpers()
}

func (rl *Instance) clearLine() {
	if len(rl.line) == 0 {
		return
	}

	// We need to go back to prompt
	moveCursorUp(rl.posY)
	moveCursorBackwards(GetTermWidth())
	moveCursorForwards(rl.promptLen)

	// Clear everything after & below the cursor
	print(seqClearScreenBelow)

	// Real input line
	rl.line = []rune{}
	rl.lineComp = []rune{}
	rl.pos = 0
	rl.posX = 0
	rl.fullX = 0
	rl.posY = 0
	rl.fullY = 0

	// Completions are also reset
	rl.clearVirtualComp()
}
