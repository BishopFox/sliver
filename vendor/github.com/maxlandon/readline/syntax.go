package readline

func (rl *Instance) syntaxCompletion() {
	if rl.SyntaxCompleter == nil {
		return
	}

	newLine, newPos := rl.SyntaxCompleter(rl.line, rl.pos-1)
	if string(newLine) == string(rl.line) {
		return
	}

	newPos++

	rl.line = newLine
	rl.echo()
	moveCursorForwards(newPos - rl.pos - 1)
	moveCursorBackwards(rl.pos - newPos + 1)
	rl.pos = newPos
}
