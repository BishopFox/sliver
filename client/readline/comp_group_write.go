package readline

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/evilsocket/islazy/tui"
)

// writeCompletion - This function produces a formatted string containing all appropriate items
// and according to display settings. This string is then appended to the main completion string.
func (g *CompletionGroup) writeCompletion(rl *Instance) (comp string) {

	// Avoids empty groups in suggestions
	if len(g.Suggestions) == 0 {
		return
	}

	// Depending on display type we produce the approriate string
	switch g.DisplayType {

	case TabDisplayGrid:
		comp += g.writeGrid(rl)
	case TabDisplayMap:
		comp += g.writeMap(rl)
	case TabDisplayList:
		comp += g.writeList(rl)
	}

	// If at the end, for whatever reason, we have a string consisting
	// only of the group's name/description, we don't append it to
	// completions and therefore return ""
	if comp == "" {
		return ""
	}

	return
}

// writeGrid - A grid completion string
func (g *CompletionGroup) writeGrid(rl *Instance) (comp string) {

	// Print group title
	comp += fmt.Sprintf("\n %s%s%s %s\n", tui.BOLD, tui.YELLOW, g.Name, tui.RESET)

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

// writeList - A list completion string
func (g *CompletionGroup) writeList(rl *Instance) (comp string) {

	// Print group title (changes with line returns depending on type)
	comp += fmt.Sprintf("\n %s%s%s %s", tui.BOLD, tui.YELLOW, g.Name, tui.RESET)

	termWidth := GetTermWidth()
	if termWidth < 20 {
		// terminal too small. Probably better we do nothing instead of crash
		// We are more conservative than lmorg, and push it to 20 instead of 10
		return
	}

	// Suggestion cells dimensions
	maxLength := g.tcMaxLength
	if maxLength > termWidth-9 {
		maxLength = termWidth - 9
	}
	cellWidth := strconv.Itoa(maxLength)

	// Alternative suggestion cells dimensions
	maxLengthAlt := g.tcMaxLengthAlt
	if maxLengthAlt > termWidth-9 {
		maxLengthAlt = termWidth - 9
	}
	cellWidthAlt := strconv.Itoa(maxLengthAlt)

	// Descriptions cells dimensions
	maxDescWidth := termWidth - maxLength - maxLengthAlt - 4

	// function highlights the cell depending on current selector place.
	highlight := func(y int, x int) string {
		if y == g.tcPosY && x == g.tcPosX && g.isCurrent {
			return seqCtermFg255 + seqFgBlackBright
		}
		return ""
	}

	// For each line in completions
	y := 0
	for i := g.tcOffset; i < len(g.Suggestions); i++ {
		y++ // Consider next item
		if y > g.tcMaxY {
			break
		}

		// Main suggestion
		item := g.Suggestions[i]
		if len(item) > maxLength {
			item = item[:maxLength-3] + "..."
		}
		sugg := fmt.Sprintf("\r\n%s%-"+cellWidth+"s", highlight(y, 1), item)

		// Alt suggestion
		alt, ok := g.SuggestionsAlt[item]
		if ok {
			alt = fmt.Sprintf(" %s%"+cellWidthAlt+"s", highlight(y, 2), alt)
		} else {
			// Else, make an empty cell
			alt = strings.Repeat(" ", maxLengthAlt+2) // + 2 to keep account of spaces
		}

		// Description
		description := g.Descriptions[g.Suggestions[i]]
		if len(description) > maxDescWidth {
			description = description[:maxDescWidth-3] + "..."
		}

		// Total completion line
		comp += sugg + seqReset + alt + " " + seqReset + description
	}

	// Add the equivalent of this group's size to final screen clearing
	// Cannot be set with MaxLength when printing lists
	rl.tcUsedY += len(g.Suggestions) + 1

	// Add the equivalent of this group's size to final screen clearing
	// Can be set and used only if no alterative completions have been given.
	// if len(g.SuggestionsAlt) == 0 {
	//         if len(g.Suggestions) > g.MaxLength {
	//                 rl.tcUsedY += g.MaxLength + 2 // Do is required for lists
	//         } else {
	//                 rl.tcUsedY += len(g.Suggestions) + 1
	//         }
	// } else {
	//         rl.tcUsedY += len(g.Suggestions) + 1
	// }

	// Special case: history search handles titles differently.
	if rl.modeAutoFind && rl.modeTabFind && rl.searchMode == HistoryFind {
		rl.tcUsedY--
	}

	return
}

// writeMap - A map or list completion string
func (g *CompletionGroup) writeMap(rl *Instance) (comp string) {

	// Title is not printed for history
	if rl.modeAutoFind && rl.modeTabFind && rl.searchMode == HistoryFind {
		if len(g.Suggestions) == 0 {
			rl.hintText = []rune(fmt.Sprintf("\n%s%s%s %s", tui.DIM, tui.RED,
				"No command history source, or empty", tui.RESET))
		}
	} else {
		// Print group title (changes with line returns depending on type)
		comp += fmt.Sprintf("\n %s%s%s %s", tui.BOLD, tui.YELLOW, g.Name, tui.RESET)
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
