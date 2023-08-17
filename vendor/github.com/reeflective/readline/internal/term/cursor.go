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

// MoveCursorForwards moves the cursor forward i columns.
func MoveCursorForwards(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dC", i)
}

// MoveCursorBackwards moves the cursor backward i columns.
func MoveCursorBackwards(i int) {
	if i < 1 {
		return
	}

	printf("\x1b[%dD", i)
}
