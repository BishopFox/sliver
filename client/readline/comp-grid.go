package readline

import (
	"fmt"
	"strconv"
)

// initGrid - Grid display details. Called each time we want to be sure to have
// a working completion group either immediately, or later on. Generally defered.
func (g *CompletionGroup) initGrid(rl *Instance) {

	// Compute size of each completion item box
	tcMaxLength := 1
	for i := range g.Suggestions {
		if len(g.Suggestions[i]) > tcMaxLength {
			tcMaxLength = len([]rune(g.Suggestions[i]))
		}
	}

	g.tcPosX = 0
	g.tcPosY = 1
	g.tcOffset = 0

	// Max number of columns
	g.tcMaxX = GetTermWidth() / (tcMaxLength + 2)
	if g.tcMaxX < 1 {
		g.tcMaxX = 1 // avoid a divide by zero error
	}

	// Maximum number of lines
	maxY := len(g.Suggestions) / g.tcMaxX
	rest := len(g.Suggestions) % g.tcMaxX
	if rest != 0 && maxY != 1 {
		maxY++
	}
	if maxY > g.MaxLength {
		g.tcMaxY = g.MaxLength
	} else {
		g.tcMaxY = maxY
	}
}

// moveTabGridHighlight - Moves the highlighting for currently selected completion item (grid display)
func (g *CompletionGroup) moveTabGridHighlight(rl *Instance, x, y int) (done bool, next bool) {

	g.tcPosX += x
	g.tcPosY += y

	// Columns
	if g.tcPosX < 1 {
		if g.tcPosY == 1 && rl.tabCompletionReverse {
			g.tcPosX = 1
			g.tcPosY = 0
		} else {
			// This is when multiple ligns, not yet on first one.
			g.tcPosX = g.tcMaxX
			g.tcPosY--
		}
	}
	if g.tcPosY > g.tcMaxY {
		g.tcPosY = 1
		return true, true
	}

	// If we must move to next line in same group
	if g.tcPosX > g.tcMaxX {
		g.tcPosX = 1
		g.tcPosY++
	}

	// Real max number of suggestions.
	max := g.tcMaxX * g.tcMaxY
	if max > len(g.Suggestions) {
		max = len(g.Suggestions)
	}

	// We arrived at the end of suggestions. This condition can never be triggered
	// while going in the reverse order, only forward, so no further checks in it.
	if (g.tcMaxX*(g.tcPosY-1))+g.tcPosX > max {
		return true, true
	}

	// In case we are reverse cycling and currently selecting the first item,
	// we adjust the coordinates to point to the last item and return
	// We set g.tcPosY because the printer needs to get the a candidate nonetheless.
	if rl.tabCompletionReverse && g.tcPosX == 1 && g.tcPosY == 0 {
		g.tcPosY = 1
		return true, false
	}

	// By default, come back to this group for next item.
	return false, false
}

// writeGrid - A grid completion string
func (g *CompletionGroup) writeGrid(rl *Instance) (comp string) {

	// If group title, print it and adjust offset.
	if g.Name != "" {
		comp += fmt.Sprintf("\n %s%s%s %s\n", BOLD, YELLOW, g.Name, RESET)
		rl.tcUsedY++
	}

	cellWidth := strconv.Itoa((GetTermWidth() / g.tcMaxX) - 2)
	x := 0
	y := 1

	for i := range g.Suggestions {
		x++
		if x > g.tcMaxX {
			x = 1
			y++
			if y > g.tcMaxY {
				y--
				break
			} else {
				comp += "\r\n"
			}
		}

		if (x == g.tcPosX && y == g.tcPosY) && (g.isCurrent) {
			comp += seqCtermFg255 + seqFgBlackBright
		}

		comp += fmt.Sprintf("%-"+cellWidth+"s %s", g.Suggestions[i], seqReset)
	}

	// Add the equivalent of this group's size to final screen clearing.
	// This is either the max allowed print size for this group, or its actual size if inferior.
	if g.MaxLength < y {
		rl.tcUsedY += g.MaxLength
	} else {
		rl.tcUsedY += y
	}

	return
}
