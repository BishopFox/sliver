package readline

import (
	"fmt"
	"strconv"
	"strings"
)

// initList - List display details. Because of the way alternative completions
// are handled, MaxLength cannot be set when there are alternative completions.
func (g *CompletionGroup) initList(rl *Instance) {

	// We may only ever have two different
	// columns: (suggestions, and alternatives)
	g.tcMaxX = 2

	// We make the list anyway, especially if we need to use it later
	if g.Descriptions == nil {
		g.Descriptions = make(map[string]string)
	}
	if g.SuggestionsAlt == nil {
		g.SuggestionsAlt = make(map[string]string)
	}

	// Compute size of each completion item box. Group independent
	g.tcMaxLength = rl.getListPad()

	// Same for suggestions alt
	g.tcMaxLengthAlt = 0
	for i := range g.Suggestions {
		if len(g.Suggestions[i]) > g.tcMaxLength {
			g.tcMaxLength = len([]rune(g.Suggestions[i]))
		}
	}

	// Max values depend on if we have alternative suggestions
	if len(g.SuggestionsAlt) == 0 {
		g.tcMaxX = 1
	} else {
		g.tcMaxX = 2
	}

	if len(g.Suggestions) > g.MaxLength {
		g.tcMaxY = g.MaxLength
	} else {
		g.tcMaxY = len(g.Suggestions)
	}

	g.tcPosX = 0
	g.tcPosY = 0
	g.tcOffset = 0
}

// moveTabListHighlight - Moves the highlighting for currently selected completion item (list display)
// We don't care about the x, because only can have 2 columns of selectable choices (--long and -s)
func (g *CompletionGroup) moveTabListHighlight(rl *Instance, x, y int) (done bool, next bool) {

	// We dont' pass to x, because not managed by callers
	g.tcPosY += x

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

	// Once we get to the end of choices: check which column we were selecting.
	if g.tcOffset+g.tcPosY > len(g.Suggestions) {
		// If we have alternative options and that we are not yet
		// completing them, start on top of their column
		if g.tcPosX == 0 && len(g.SuggestionsAlt) > 0 {
			g.tcPosX++
			g.tcPosY = 1
			g.tcOffset = 0
			return false, false
		}

		// Else no alternatives, return for next group.
		// Reset all values, in case we pass on them again.
		g.tcPosX = 0 // First column
		g.tcPosY = 1 // first row
		g.tcOffset = 0
		return true, true
	}

	// Here we must check, in x == 1, that the current choice
	// is not empty. Handle for both reverse and forward movements.
	sugg := g.Suggestions[g.tcPosY-1]
	_, ok := g.SuggestionsAlt[sugg]
	if !ok && g.tcPosX == 1 {
		if rl.tabCompletionReverse {
			for i := len(g.Suggestions[:g.tcPosY-1]); i > 0; i-- {
				su := g.Suggestions[i]
				if _, ok := g.SuggestionsAlt[su]; ok {
					g.tcPosY -= (len(g.Suggestions[:g.tcPosY-1])) - i
					return false, false
				}
			}
			g.tcPosX = 0
			g.tcPosY = g.tcMaxY

		} else {
			for i, su := range g.Suggestions[g.tcPosY-1:] {
				if _, ok := g.SuggestionsAlt[su]; ok {
					g.tcPosY += i
					return false, false
				}
			}
		}
	}

	// Setup offset if needs to be.
	// TODO: should be rewrited to conditionally process rolling menus with alternatives
	if g.tcOffset+g.tcPosY < 1 && len(g.Suggestions) > 0 {
		g.tcPosY = g.tcMaxY
		g.tcOffset = len(g.Suggestions) - g.tcMaxY
	}
	if g.tcOffset < 0 {
		g.tcOffset = 0
	}

	// MIGHT BE NEEDED IF PROBLEMS WIHT ROLLING COMPLETIONS
	// ------------------------------------------------------------------------------
	// Once we get to the end of choices: check which column we were selecting.
	// We use +1 because we may have a single suggestion, and we just want "a ratio"
	// if g.tcOffset+g.tcPosY > len(g.Suggestions) {
	//
	//         // If we have alternative options and that we are not yet
	//         // completing them, start on top of their column
	//         if g.tcPosX == 1 && len(g.SuggestionsAlt) > 0 {
	//                 g.tcPosX++
	//                 g.tcPosY = 1
	//                 g.tcOffset = 0
	//                 return false
	//         }
	//
	//         // Else no alternatives, return for next group.
	//         g.tcPosY = 1
	//         return true
	// }
	return false, false
}

// writeList - A list completion string
func (g *CompletionGroup) writeList(rl *Instance) (comp string) {

	// Print group title and adjust offset if there is one.
	if g.Name != "" {
		comp += fmt.Sprintf("\n %s%s%s %s", BOLD, YELLOW, g.Name, RESET)
		rl.tcUsedY++
	}

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
	maxLengthAlt := g.tcMaxLengthAlt + 2
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
		sugg := fmt.Sprintf("\r\n%s%-"+cellWidth+"s", highlight(y, 0), item)

		// Alt suggestion
		alt, ok := g.SuggestionsAlt[item]
		if ok {
			alt = fmt.Sprintf(" %s%"+cellWidthAlt+"s", highlight(y, 1), alt)
		} else {
			// Else, make an empty cell
			alt = strings.Repeat(" ", maxLengthAlt+2) // + 2 to keep account of spaces
		}

		// Description
		description := g.Descriptions[g.Suggestions[i]]
		if len(description) > maxDescWidth {
			description = description[:maxDescWidth-3] + "..." + RESET
		}

		// Total completion line
		comp += sugg + seqReset + alt + " " + seqReset + description
	}

	// Add the equivalent of this group's size to final screen clearing
	// Can be set and used only if no alterative completions have been given.
	if len(g.SuggestionsAlt) == 0 {
		if len(g.Suggestions) > g.MaxLength {
			rl.tcUsedY += g.MaxLength
		} else {
			rl.tcUsedY += len(g.Suggestions)
		}
	} else {
		rl.tcUsedY += len(g.Suggestions)
	}

	// Special case: history search handles titles differently.
	if rl.modeAutoFind && rl.modeTabFind && rl.searchMode == HistoryFind {
		rl.tcUsedY--
	}

	return
}

func (rl *Instance) getListPad() (pad int) {
	for _, group := range rl.tcGroups {
		if group.DisplayType == TabDisplayList {
			for i := range group.Suggestions {
				if len(group.Suggestions[i]) > pad {
					pad = len([]rune(group.Suggestions[i]))
				}
			}
		}
	}

	return
}
