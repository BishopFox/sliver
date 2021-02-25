package readline

// DISPLAY ------------------------------------------------------------
// All cursorMoveFunctions move the cursor as it is seen by the user.
// This means that they are not used to keep any reference point when
// when we internally move around clearning and printing things

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

func (rl *Instance) backspace() {
	if len(rl.line) == 0 || rl.pos == 0 {
		return
	}

	rl.delete()
}

func (rl *Instance) moveCursorByAdjust(adjust int) {
	switch {
	case adjust > 0:
		rl.pos += adjust
	case adjust < 0:
		rl.pos += adjust
	}

	if rl.modeViMode != vimInsert && (rl.pos == len(rl.line)-1) && len(rl.line) > 0 {
		rl.pos--
	}
}
