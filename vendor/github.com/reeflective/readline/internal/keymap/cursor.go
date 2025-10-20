package keymap

import (
	"fmt"
	"strings"
)

// CursorStyle is the style of the cursor
// in a given input mode/submode.
type CursorStyle string

// String - Implements fmt.Stringer.
func (c CursorStyle) String() string {
	cursor, found := cursors[c]
	if !found {
		return string(cursorUserDefault)
	}

	return cursor
}

const (
	cursorBlock             CursorStyle = "block"
	cursorUnderline         CursorStyle = "underline"
	cursorBeam              CursorStyle = "beam"
	cursorBlinkingBlock     CursorStyle = "blinking-block"
	cursorBlinkingUnderline CursorStyle = "blinking-underline"
	cursorBlinkingBeam      CursorStyle = "blinking-beam"
	cursorUserDefault       CursorStyle = "default"
)

var cursors = map[CursorStyle]string{
	cursorBlock:             "\x1b[2 q",
	cursorUnderline:         "\x1b[4 q",
	cursorBeam:              "\x1b[6 q",
	cursorBlinkingBlock:     "\x1b[1 q",
	cursorBlinkingUnderline: "\x1b[3 q",
	cursorBlinkingBeam:      "\x1b[5 q",
	cursorUserDefault:       "\x1b[0 q",
}

var defaultCursors = map[Mode]CursorStyle{
	ViInsert:  cursorBlinkingBeam,
	Vi:        cursorBlinkingBeam,
	ViCommand: cursorBlinkingBlock,
	ViOpp:     cursorBlinkingUnderline,
	Visual:    cursorBlock,
	Emacs:     cursorBlinkingBlock,
}

// PrintCursor prints the cursor for the given keymap mode,
// either default value or the one specified in inputrc file.
func (m *Engine) PrintCursor(keymap Mode) {
	var cursor CursorStyle

	// Check for a configured cursor in .inputrc file.
	cursorOptname := "cursor-" + string(keymap)
	modeSet := strings.TrimSpace(m.config.GetString(cursorOptname))

	if _, valid := cursors[CursorStyle(modeSet)]; valid {
		fmt.Print(cursors[CursorStyle(modeSet)])
		return
	}

	if defaultCur, valid := defaultCursors[keymap]; valid {
		fmt.Print(cursors[defaultCur])
		return
	}

	fmt.Print(cursors[cursor])
}
