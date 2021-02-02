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
	g.tcPosY = 0
	g.tcOffset = 0

	g.tcMaxX = GetTermWidth() / (tcMaxLength + 2)
	if g.tcMaxX < 1 {
		g.tcMaxX = 1 // avoid a divide by zero error
	}
	if g.MaxLength == 0 {
		g.MaxLength = 10 // Handle default value if not set
	}
	g.tcMaxY = g.MaxLength

}

// moveTabGridHighlight - Moves the highlighting for currently selected completion item (grid display)
func (g *CompletionGroup) moveTabGridHighlight(rl *Instance, x, y int) (done bool) {

	g.tcPosX += x
	g.tcPosY += y

	// Columns
	if g.tcPosX < 1 {
		g.tcPosX = g.tcMaxX
		g.tcPosY--
	}
	if g.tcPosX > g.tcMaxX {
		g.tcPosX = 1
		g.tcPosY++
	}

	// Lines
	if g.tcPosY < 1 {
		g.tcPosY = rl.tcUsedY
	}
	if g.tcPosY > rl.tcUsedY {
		g.tcPosY = 1
		return true
	}

	if (g.tcMaxX*(g.tcPosY-1))+g.tcPosX > len(g.Suggestions) {
		if x < 0 {
			g.tcPosX = len(g.Suggestions) - (g.tcMaxX * (g.tcPosY - 1))
		}

		if x > 0 {
			g.tcPosX = 1
			g.tcPosY = 1
		}

		if y < 0 {
			g.tcPosY--
		}

		if y > 0 {
			g.tcPosY = 1
		}

		return true
	}

	return false
}

// writeGrid - A grid completion string
func (g *CompletionGroup) writeGrid(rl *Instance) (comp string) {

	// Print group title
	comp += fmt.Sprintf("\n %s%s%s %s\n", BOLD, YELLOW, g.Name, RESET)

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
		rl.tcUsedY += g.MaxLength + 1 // + 1 for title
	} else {
		rl.tcUsedY += y + 1
	}

	return
}
