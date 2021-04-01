package readline

import "regexp"

// SetHintText - a nasty function to force writing a new hint text. It does not update helpers, it just renders
// them, so the hint will survive until the helpers (thus including the hint) will be updated/recomputed.
func (rl *Instance) SetHintText(s string) {
	rl.hintText = []rune(s)
	rl.renderHelpers()
}

func (rl *Instance) getHintText() {

	if !rl.modeAutoFind && !rl.modeTabFind {
		// Return if no hints provided by the user/engine
		if rl.HintText == nil {
			rl.resetHintText()
			return
		}
		// The hint text also works with the virtual completion line system.
		// This way, the hint is also refreshed depending on what we are pointing
		// at with our cursor.
		rl.hintText = rl.HintText(rl.getCompletionLine())
	}
}

// writeHintText - only writes the hint text and computes its offsets.
func (rl *Instance) writeHintText() {
	if len(rl.hintText) == 0 {
		rl.hintY = 0
		return
	}

	width := GetTermWidth()

	// Wraps the line, and counts the number of newlines in the string,
	// adjusting the offset as well.
	re := regexp.MustCompile(`\r?\n`)
	newlines := re.Split(string(rl.hintText), -1)
	offset := len(newlines)

	wrapped, hintLen := WrapText(string(rl.hintText), width)
	offset += hintLen
	rl.hintY = offset

	hintText := string(wrapped)

	if len(hintText) > 0 {
		print("\r" + rl.HintFormatting + string(hintText) + seqReset)
	}
}

func (rl *Instance) resetHintText() {
	rl.hintY = 0
	rl.hintText = []rune{}
}
