package completion

import (
	"github.com/reeflective/readline/internal/core"
	"github.com/reeflective/readline/internal/strutil"
)

// CompleteSyntax updates the line with either user-defined syntax completers, or with the builtin ones.
func (e *Engine) CompleteSyntax(completer func([]rune, int) ([]rune, int)) {
	if completer == nil {
		return
	}

	line := []rune(*e.line)
	pos := e.cursor.Pos() - 1

	newLine, newPos := completer(line, pos)
	if string(newLine) == string(line) {
		return
	}

	newPos++

	e.line.Set(newLine...)
	e.cursor.Set(newPos)
}

// AutopairInsertOrJump checks if the character to be inserted in the line is a pair character.
// If the character is an opening one, its inserted along with its closing equivalent.
// If it's a closing one and the next character in line is the same, the cursor jumps over it.
func AutopairInsertOrJump(key rune, line *core.Line, cur *core.Cursor) (skipInsert bool) {
	matcher, closer := strutil.SurroundType(key)

	if !matcher {
		return
	}

	switch {
	case closer && cur.Char() == key:
		skipInsert = true

		cur.Inc()
	case closer && key != '\'' && key != '"':
		return
	default:
		_, closeChar := strutil.MatchSurround(key)
		line.Insert(cur.Pos(), closeChar)
	}

	return
}

// AutopairDelete checks if the character under the cursor is an opening pair
// character which is immediately followed by its closing equivalent. If yes,
// the closing character is removed.
func AutopairDelete(line *core.Line, cur *core.Cursor) {
	if cur.Pos() == 0 {
		return
	}

	toDelete := (*line)[cur.Pos()-1]
	isPair, _ := strutil.SurroundType(toDelete)
	matcher := strutil.IsSurround(toDelete, cur.Char())

	// Cut the (closing) rune under the cursor.
	if isPair && matcher {
		line.CutRune(cur.Pos())
	}
}
