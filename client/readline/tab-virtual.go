package readline

import (
	"strings"
)

// insertVirtual - When a completion candidate is selected, we insert it virtually in the input line:
// this will not trigger further firltering against the other candidates. Each time this function
// is called, any previous candidate is dropped, after being used for moving the cursor around.
func (rl *Instance) insertVirtual(candidate []rune) {
	for {
		// I don't really understand why `0` is creaping in at the end of the
		// array but it only happens with unicode characters.
		if len(candidate) > 1 && candidate[len(candidate)-1] == 0 {
			candidate = candidate[:len(candidate)-1]
			continue
		}
		break
	}

	// We place the cursor back at the beginning of the previous virtual candidate
	rl.pos -= len(rl.currentComp)
	moveCursorBackwards(len(rl.currentComp) - 1)

	// We clear the current line
	print(strings.Repeat(" ", len(rl.lineComp)))
	moveCursorBackwards(len(rl.lineComp))

	// We delete the previous virtual completion, just
	// like we would delete a word in vim editing mode.
	if len(rl.currentComp) > 0 {
		rl.viDeleteByAdjustVirtual(rl.viJumpEVirtual(tokeniseSplitSpaces))
	}

	// We then keep a reference to the new candidate
	rl.currentComp = candidate

	// We should not have a remaining virtual completion line, so we delete it
	rl.lineComp = rl.line

	// Insert the new candidate in the virtual line.
	switch {
	case len(rl.lineComp) == 0:
		rl.lineComp = candidate
	case rl.pos == 0:
		rl.lineComp = append(candidate, rl.lineComp...)
	case rl.pos < len(rl.lineComp):
		r := append(candidate, rl.lineComp[rl.pos:]...)
		rl.lineComp = append(rl.lineComp[:rl.pos], r...)
	default:
		rl.lineComp = append(rl.lineComp, candidate...)
	}

	// We place the cursor at the end of our new virtually completed item
	rl.pos += len(candidate)
}

// updateVirtualCompletion - This function is called before most of our readline key handlers,
// and makes sure that the current completion (virtually inserted) is either inserted or dropped,
// and that all related parameters are reinitialized.
func (rl *Instance) updateVirtualCompletion() {

	// If we don't have a current virtual completion, there's nothing to do.
	// IMPORTANT: this MUST be first, to avoid nil problems with empty comps.
	if len(rl.currentComp) == 0 {
		return
	}

	// Get the current candidate and its group.
	//It contains info on how we must process it
	cur := rl.getCurrentGroup()
	if cur == nil {
		return
	}
	completion := cur.getCurrentCell()
	// if completion == "" {
	//         rl.line = rl.lineComp
	//         // Reset values
	//         rl.lineComp = []rune{}
	//         rl.currentComp = []rune{}
	//         return
	// }

	// We will only insert the net difference between prefix and completion.
	prefix := len(rl.tcPrefix)

	// If the current candidate is also already inserted, actualize it and return
	if completion == string(rl.currentComp) {
		if cur.TrimSlash {
			trimmed, hadSlash := trimTrailing(completion)
			if !hadSlash && rl.compAddSpace {
				trimmed = trimmed + " "
			}
			rl.insertVirtual([]rune(trimmed[prefix:]))
		} else {
			if rl.compAddSpace {
				completion = completion + " "
			}
			rl.insertVirtual([]rune(completion[prefix:]))
		}

		// Reset all relevant values
		rl.line = rl.lineComp
		rl.lineComp = []rune{}
		rl.currentComp = []rune{}
		rl.compAddSpace = false
		return
	}

	// Else, if candidate not yet inserted, insert as usual.
	if cur.TrimSlash {
		trimmed, hadSlash := trimTrailing(completion)
		if !hadSlash && rl.compAddSpace {
			trimmed = trimmed + " "
		}
		rl.insertVirtual([]rune(trimmed[prefix:]))
	} else {
		if rl.compAddSpace {
			completion = completion + " "
		}
		rl.insertVirtual([]rune(completion[prefix:]))
	}

	// Reset all relevant values
	rl.line = rl.lineComp
	rl.lineComp = []rune{}
	rl.currentComp = []rune{}
	rl.compAddSpace = false
}

// trimTrailing - When the group to which the current candidate
// belongs has TrimSlash enabled, we process the candidate.
// This is used when the completions are directory/file paths.
func trimTrailing(comp string) (trimmed string, hadSlash bool) {
	// Unix paths
	if strings.HasSuffix(comp, "/") {
		return strings.TrimSuffix(comp, "/"), true
	}
	// Windows paths
	if strings.HasSuffix(comp, "\\") {
		return strings.TrimSuffix(comp, "\\"), true
	}
	return comp, false
}

// viDeleteByAdjustVirtual - Same as viDeleteByAdjust, but for our virtually completed input line.
func (rl *Instance) viDeleteByAdjustVirtual(adjust int) {
	var (
		newLine []rune
		backOne bool
	)

	// Avoid doing anything if input line is empty.
	if len(rl.lineComp) == 0 {
		return
	}

	switch {
	case adjust == 0:
		rl.viUndoSkipAppend = true
		return
	case rl.pos+adjust == len(rl.lineComp)-1:
		newLine = rl.lineComp[:rl.pos]
		// backOne = true // Deleted, otherwise the completion moves back when we don't want to.
	case rl.pos+adjust == 0:
		newLine = rl.lineComp[rl.pos:]
	case adjust < 0:
		newLine = append(rl.lineComp[:rl.pos+adjust], rl.lineComp[rl.pos:]...)
	default:
		newLine = append(rl.lineComp[:rl.pos], rl.lineComp[rl.pos+adjust:]...)
	}

	moveCursorBackwards(rl.pos)
	print(strings.Repeat(" ", len(rl.lineComp)))
	moveCursorBackwards(len(rl.lineComp) - rl.pos)

	rl.lineComp = newLine

	rl.echo()

	if adjust < 0 {
		rl.moveCursorByAdjust(adjust)
	}

	if backOne {
		moveCursorBackwards(1)
		rl.pos--
	}
}

// viJumpEVirtual - Same as viJumpE, but for our virtually completed input line.
func (rl *Instance) viJumpEVirtual(tokeniser func([]rune, int) ([]string, int, int)) (adjust int) {
	split, index, pos := tokeniser(rl.lineComp, rl.pos)
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
