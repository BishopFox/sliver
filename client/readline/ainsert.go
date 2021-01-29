package readline

import "strings"

// ainsert - Alternative live reloading of the input line. This might be helpful for a few reasons:
// - We might add Vim status at the beginning of the prompt.                                                DONE
// - We might even do live prompt reloading, with other items.
// - We may emulate the ZSH's pattern, which adds the current completion highlight to the input line,
//   without modifying the search.
//
// Several things to modify therefore:
// - We have to account for the length of the prompt that we do not consider as user input.
// - Track this realtime.
//
// the function either inserts the completion into the real input line, or the virtual completion
// line, which can be seen on the input but is not (for the moment) the real one.
func (rl *Instance) ainsert(r []rune) {
	for {
		// I don't really understand why `0` is creaping in at the end of the
		// array but it only happens with unicode characters.
		if len(r) > 1 && r[len(r)-1] == 0 {
			r = r[:len(r)-1]
			continue
		}
		break
	}

	// First determine if the current virtual completion
	// is longer than the previous one. If yes we must
	// clean the input line a bit first.
	diff := len(rl.lineComp) - len(rl.line) - len(r)
	if diff > 0 {
		moveCursorBackwards(rl.pos)
		print(strings.Repeat(" ", len(rl.lineComp)))
	}

	// Reset virtually completed line
	rl.lineComp = rl.line
	rl.echo()

	switch {
	default:
		rl.pos = len(rl.line)
		rl.lineComp = append(rl.line, r...)
	}

	// The cursor goes to the end of the completion.
	rl.pos += len(r)
	moveCursorForwards(len(r) - 1)
}

// each time we have to "realize" the virtual completion line,
// this function takes charge of it.
func (rl *Instance) insertVirtual(comp string) {
	prefix := len(rl.tcPrefix)

	// default case is that lineComp is longer than the line
	if len(rl.lineComp) > len(rl.line) {
		rl.line = rl.lineComp

		// Or we just insert
	} else {
		rl.insert([]rune(comp[prefix:]))
	}
}
