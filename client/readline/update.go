package readline

import (
	"errors"
	"fmt"
	"strings"
)

// RefreshMultiline - This function can be called by the user program to refresh the first line of the prompt,
// if the latter is a 2-line (multiline) prompt. This function should refresh the prompt "in place", which
// means it renders it directly where it was: it does not print a new one below.
// The offset param can be used to adjust the number of lines to clear upward, in case there are things the
// shell cannot know. Set offset to 0 if you don't use it.
func (rl *Instance) RefreshMultiline(prompt string, printPrompt bool, offset int, clearLine bool) (err error) {

	if !rl.Multiline {
		return errors.New("readline error: refresh cannot happen, prompt is not multiline")
	}

	// We adjust cursor movement, depending on which mode we're currently in.
	if !rl.modeTabCompletion {
		rl.tcUsedY = 1
	} else if rl.modeTabCompletion && rl.modeAutoFind { // Account for the hint line
		rl.tcUsedY = 0
	} else {
		rl.tcUsedY = 1
	}

	// Add user-provided offset
	rl.tcUsedY += offset

	print(seqClearLine) // Clear the current line, which might be longer than what's overwritten
	moveCursorUp(rl.hintY + rl.tcUsedY)
	moveCursorBackwards(GetTermWidth())
	print("\r\n" + seqClearScreenBelow) // We add this to clear everything below offset.

	// Print first line of prompt if asked to
	if prompt != "" {
		rl.prompt = prompt
	}
	if printPrompt {
		fmt.Println(rl.prompt)
		rl.renderHelpers()
	}

	// If input line was empty, check that we clear it from detritus
	// The three lines are borrowed from clearLine(), we don't need more.
	if clearLine {
		rl.clearLine()
	}

	return
}

// computePrompt - At any moment, returns prompt actualized with Vim status
func (rl *Instance) computePrompt() (prompt []rune) {

	// Add custom prompt string if provided by user
	if rl.MultilinePrompt != "" {
		prompt = append(prompt, []rune(rl.MultilinePrompt)...)
	}

	// If ModeVimEnabled, append it.
	if rl.ShowVimMode {

		switch rl.modeViMode {
		case vimKeys:
			prompt = append(prompt, []rune(vimKeysStr)...)
		case vimInsert:
			prompt = append(prompt, []rune(vimInsertStr)...)
		case vimReplaceOnce:
			prompt = append(prompt, []rune(vimReplaceOnceStr)...)
		case vimReplaceMany:
			prompt = append(prompt, []rune(vimReplaceManyStr)...)
		case vimDelete:
			prompt = append(prompt, []rune(vimDeleteStr)...)
		}

		// Process colors
		prompt = rl.colorizeVimPrompt(prompt)
		// Add the arrow
		prompt = append(prompt, rl.mlnArrow...)
	}

	rl.mlnPrompt = prompt
	rl.promptLen = len(rl.mlnPrompt)

	return
}

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
	var length int

	// We use either the normal prompt, or the multiline one
	if !rl.Multiline {
		length = len(rl.prompt)
	} else {
		length = len(rl.MultilinePrompt)
	}

	// If the user wants Vim status
	if rl.ShowVimMode {
		length += 3                // 3 for [N]
		length += len(rl.mlnArrow) // 3: ' > '
	}

	// move the cursor
	moveCursorForwards(length + rl.pos)
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
	// fmt.Println(rl.pos)
	if len(rl.line) == 0 || rl.pos == 0 {
		return
	}

	moveCursorBackwards(1)
	// fmt.Println(len(rl.line))
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
	moveCursorBackwards(len(rl.mlnPrompt) + rl.pos)

	switch {
	case rl.PasswordMask > 0:
		print(strings.Repeat(string(rl.PasswordMask), len(rl.line)) + " ")

	case rl.SyntaxHighlighter == nil:
		print(string(rl.mlnPrompt))
		print(string(rl.line) + " ")

	default:
		print(string(rl.mlnPrompt))
		print(rl.SyntaxHighlighter(rl.line) + " ")
	}

	moveCursorBackwards(len(rl.line) - rl.pos)
}

func (rl *Instance) clearLine() {
	if len(rl.line) == 0 {
		return
	}

	moveCursorBackwards(rl.pos)
	print(strings.Repeat(" ", len(rl.line)))
	moveCursorBackwards(len(rl.line))

	rl.line = []rune{}
	rl.pos = 0
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
}

func (rl *Instance) renderHelpers() {

	rl.echo() // Added by me, so that prompt always appear when new line
	rl.writeHintText()
	rl.writeTabCompletion()

	moveCursorUp(rl.hintY + rl.tcUsedY)
	moveCursorBackwards(GetTermWidth())

	moveCursorToLinePos(rl)
}

func (rl *Instance) updateHelpers() {
	rl.tcOffset = 0
	rl.getHintText()
	if rl.modeTabCompletion {
		rl.getTabCompletion()
	}
	rl.clearHelpers()
	rl.renderHelpers()
}
