package readline

// ainsert - Alternative live reloading of the input line. This might be helpful for a few reasons:
// - We might add Vim status at the beginning of the prompt.
// - We might even do live prompt reloading, with other items.
// - We may emulate the ZSH's pattern, which adds the current completion highlight to the input line,
//   without modifying the search.
//
// Several things to modify therefore:
// - We have to account for the length of the prompt that we do not consider as user input.
// - Track this realtime.
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
