package term

// MoveCursorUp moves the cursor up i lines.
func MoveCursorUp(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dA", i)
}

// MoveCursorDown moves the cursor down i lines.
func MoveCursorDown(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dB", i)
}

// MoveCursorForward moves the cursor forward i columns.
func MoveCursorForward(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dC", i)
}

// MoveCursorBackward moves the cursor backward i columns.
func MoveCursorBackward(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dD", i)
}
