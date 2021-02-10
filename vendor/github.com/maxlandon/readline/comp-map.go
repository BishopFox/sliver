package readline

import (
	"fmt"
	"strconv"
)

// initMap - Map display details. Called each time we want to be sure to have
// a working completion group either immediately, or later on. Generally defered.
func (g *CompletionGroup) initMap(rl *Instance) {

	// We make the map anyway, especially if we need to use it later
	if g.Descriptions == nil {
		g.Descriptions = make(map[string]string)
	}

	// Compute size of each completion item box. Group independent
	g.tcMaxLength = 1
	for i := range g.Suggestions {
		if len(g.Descriptions[g.Suggestions[i]]) > g.tcMaxLength {
			g.tcMaxLength = len(g.Descriptions[g.Suggestions[i]])
		}
	}

	g.tcPosX = 0
	g.tcPosY = 0
	g.tcOffset = 0

	// Number of lines allowed to be printed for group
	if len(g.Suggestions) > g.MaxLength {
		g.tcMaxY = g.MaxLength
	} else {
		g.tcMaxY = len(g.Suggestions)
	}
}

// moveTabMapHighlight - Moves the highlighting for currently selected completion item (map display)
func (g *CompletionGroup) moveTabMapHighlight(rl *Instance, x, y int) (done bool, next bool) {

	g.tcPosY += x
	g.tcPosY += y

	// Lines
	if g.tcPosY < 1 {
		if rl.tabCompletionReverse {
			if g.tcOffset > 0 {
				g.tcPosY = 1
				g.tcOffset--
			} else {
				return true, false
			}
		}
	}
	if g.tcPosY > g.tcMaxY {
		g.tcPosY--
		g.tcOffset++
	}

	if g.tcOffset+g.tcPosY < 1 && len(g.Suggestions) > 0 {
		g.tcPosY = g.tcMaxY
		g.tcOffset = len(g.Suggestions) - g.tcMaxY
	}
	if g.tcOffset < 0 {
		g.tcOffset = 0
	}

	if g.tcOffset+g.tcPosY > len(g.Suggestions) {
		return true, true
	}
	return false, false
}

// writeMap - A map or list completion string
func (g *CompletionGroup) writeMap(rl *Instance) (comp string) {

	// Title is not printed for history
	if rl.modeAutoFind && rl.modeTabFind && rl.searchMode == HistoryFind {
		if len(g.Suggestions) == 0 {
			rl.hintText = []rune(fmt.Sprintf("\n%s%s%s %s", DIM, RED,
				"No command history source, or empty", RESET))
		}
	} else {
		comp += "\n"
		if g.Name != "" {
			// Print group title (changes with line returns depending on type)
			comp += fmt.Sprintf(" %s%s%s %s", BOLD, YELLOW, g.Name, RESET)
			rl.tcUsedY++
		}
	}

	termWidth := GetTermWidth()
	if termWidth < 20 {
		// terminal too small. Probably better we do nothing instead of crash
		// We are more conservative than lmorg, and push it to 20 instead of 10
		return
	}

	// Set all necessary dimensions
	maxLength := g.tcMaxLength
	if maxLength > termWidth-9 {
		maxLength = termWidth - 9
	}
	maxDescWidth := termWidth - maxLength - 4

	cellWidth := strconv.Itoa(maxLength)
	itemWidth := strconv.Itoa(maxDescWidth)
	y := 0

	// Highlighting function
	highlight := func(y int) string {
		if y == g.tcPosY && g.isCurrent {
			return seqCtermFg255 + seqFgBlackBright
		}
		return ""
	}

	// String formating
	var item, description string
	for i := g.tcOffset; i < len(g.Suggestions); i++ {
		y++ // Consider new item
		if y > g.tcMaxY {
			break
		}

		item = g.Suggestions[i]

		if len(item) > maxDescWidth {
			item = item[:maxDescWidth-3] + "..."
		}

		description = g.Descriptions[g.Suggestions[i]]
		if len(description) > maxLength {
			description = description[:maxLength-3] + "..."
		}

		comp += fmt.Sprintf("\r\n%-"+cellWidth+"s %s %-"+itemWidth+"s %s",
			description, highlight(y), item, seqReset)
	}

	// Add the equivalent of this group's size to final screen clearing
	if len(g.Suggestions) > g.MaxLength {
		rl.tcUsedY += g.MaxLength + 1
	} else {
		rl.tcUsedY += len(g.Suggestions) + 1
	}

	// Special case: history search handles titles differently.
	if rl.modeAutoFind && rl.modeTabFind && rl.searchMode == HistoryFind {
		rl.tcUsedY--
	}

	return
}
