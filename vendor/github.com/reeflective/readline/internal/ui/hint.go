package ui

import (
	"fmt"
	"strings"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/strutil"
	"github.com/reeflective/readline/internal/term"
)

// Hint is in charge of printing the usage messages below the input line.
// Various other UI components have access to it so that they can feed
// specialized usage messages to it, like completions.
type Hint struct {
	text       []rune
	persistent []rune
	cleanup    bool
	temp       bool
	set        bool
}

// Set sets the hint message to the given text.
// Generally, this hint message will persist until either a command
// or the completion system overwrites it, or if hint.Reset() is called.
func (h *Hint) Set(hint string) {
	h.text = []rune(hint)
	h.set = true
}

// SetTemporary sets a hint message that will be cleared at the next keypress
// or command being run, which generally coincides with the next redisplay.
func (h *Hint) SetTemporary(hint string) {
	h.text = []rune(hint)
	h.set = true
	h.temp = true
}

// Persist adds a hint message to be persistently
// displayed until hint.ResetPersist() is called.
func (h *Hint) Persist(hint string) {
	h.persistent = []rune(hint)
}

// Text returns the current hint text.
func (h *Hint) Text() string {
	return string(h.text)
}

// Len returns the length of the current hint.
// This is generally used by consumers to know if there already
// is an active hint, in which case they might want to append to
// it instead of overwriting it altogether (like in isearch mode).
func (h *Hint) Len() int {
	return len(h.text)
}

// Reset removes the hint message.
func (h *Hint) Reset() {
	h.text = make([]rune, 0)
	h.temp = false
	h.set = false
}

// ResetPersist drops the persistent hint section.
func (h *Hint) ResetPersist() {
	h.cleanup = len(h.persistent) > 0
	h.persistent = make([]rune, 0)
}

// DisplayHint prints the hint (persistent and/or temporary) sections.
func DisplayHint(hint *Hint) {
	if hint.temp && hint.set {
		hint.set = false
	} else if hint.temp {
		hint.Reset()
	}

	if len(hint.text) == 0 && len(hint.persistent) == 0 {
		if hint.cleanup {
			fmt.Print(term.ClearLineAfter)
		}

		hint.cleanup = false

		return
	}

	text := hint.renderHint()

	if strutil.RealLength(text) == 0 {
		return
	}

	text += term.ClearLineAfter + color.Reset

	if len(text) > 0 {
		fmt.Print(text)
	}
}

func (h *Hint) renderHint() (text string) {
	if len(h.persistent) > 0 {
		text += string(h.persistent) + term.NewlineReturn
	}

	if len(h.text) > 0 {
		text += string(h.text) + term.NewlineReturn
	}

	if strutil.RealLength(text) == 0 {
		return
	}

	// Ensure cross-platform, real display newline.
	text = strings.ReplaceAll(text, term.NewlineReturn, term.ClearLineAfter+term.NewlineReturn)

	return text
}

// CoordinatesHint returns the number of terminal rows used by the hint.
func CoordinatesHint(hint *Hint) int {
	text := hint.renderHint()

	// Nothing to do if no real text
	text = strings.TrimSuffix(text, term.ClearLineAfter+term.NewlineReturn)

	if strutil.RealLength(text) == 0 {
		return 0
	}

	// Otherwise compute the real length/span.
	usedY := 0
	lines := strings.Split(text, term.ClearLineAfter)

	for i, line := range lines {
		x, y := strutil.LineSpan([]rune(line), i, 0)
		if x != 0 {
			y++
		}

		usedY += y
	}

	return usedY
}
