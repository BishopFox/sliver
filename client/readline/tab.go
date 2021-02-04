package readline

import (
	"fmt"
)

// TabDisplayType defines how the autocomplete suggestions display
type TabDisplayType int

const (
	// TabDisplayGrid is the default. It's where the screen below the prompt is
	// divided into a grid with each suggestion occupying an individual cell.
	TabDisplayGrid = iota

	// TabDisplayList is where suggestions are displayed as a list with a
	// description. The suggestion gets highlighted but both are searchable (ctrl+f)
	TabDisplayList

	// TabDisplayMap is where suggestions are displayed as a list with a
	// description however the description is what gets highlighted and only
	// that is searchable (ctrl+f). The benefit of TabDisplayMap is when your
	// autocomplete suggestions are IDs rather than human terms.
	TabDisplayMap
)

// getTabCompletion - This root function sets up all completion items and engines,
// dealing with all search and completion modes. But it does not perform printing.
func (rl *Instance) getTabCompletion() {
	rl.tcOffset = 0

	if rl.TabCompleter == nil {
		return
	}

	// Populate for completion search if in this mode
	if rl.modeAutoFind && rl.searchMode == CompletionFind {
		rl.getTabSearchCompletion()
		return
	}

	// Populate for History search if in this mode
	if rl.modeAutoFind && rl.searchMode == HistoryFind {
		rl.getHistorySearchCompletion()
		return
	}

	// Else, yield normal completions
	rl.getNormalCompletion()
}

// We pass a special subset of the current input line, so that
// completions are available no matter where the cursor is.
func (rl *Instance) getCompletionLine() (line []rune, pos int) {
	switch {
	case rl.pos == len(rl.line):
		pos = rl.pos - len(rl.currentComp)
		return rl.line, pos

	case rl.pos < len(rl.line):
		pos = rl.pos - len(rl.currentComp)
		line = rl.line[:pos]
		return

	default:
		pos = rl.pos - len(rl.currentComp)
		return rl.line, pos
	}
}

// getNormalCompletion - Populates and sets up completion for normal comp mode
func (rl *Instance) getNormalCompletion() {
	rl.tcPrefix, rl.tcGroups = rl.TabCompleter(rl.getCompletionLine())
	if len(rl.tcGroups) == 0 {
		return
	}
	rl.tcGroups = checkNilItems(rl.tcGroups) // Avoid nil maps in groups

	for i, group := range rl.tcGroups {
		group.init(rl)
		if i != 0 {
			group.tcPosY = 1
		}
	}
}

// moveTabCompletionHighlight - This function is in charge of highlighting the current completion item.
func (rl *Instance) moveTabCompletionHighlight(x, y int) {

	g := rl.getCurrentGroup()

	// If there is no current group, we leave any current completion mode.
	if g == nil || g.Suggestions == nil {
		rl.modeTabCompletion = false
		return
	}

	// Get the next group that has available suggestions
	if (x > 0 || y > 0) && (len(g.Suggestions) == 0) {
		rl.cycleNextGroup()
		g = rl.getCurrentGroup()
	}
	// Or get the previous group, when going reverse
	if (x < 0 || y < 0) && (len(g.Suggestions) == 0) {
		rl.cyclePreviousGroup()
		g = rl.getCurrentGroup()
	}

	// done means we need to find the next/previous group.
	// next determines if we need to get the next OR previous group.
	var done, next bool

	// Depending on the display, we only keep track of x or (x and y)
	switch g.DisplayType {
	case TabDisplayGrid:
		done, next = g.moveTabGridHighlight(rl, x, y)
	case TabDisplayList:
		done, next = g.moveTabListHighlight(rl, x, y)

	case TabDisplayMap:
		done, next = g.moveTabMapHighlight(rl, x, y)
	}

	// Cycle to next/previous group, if done with current one.
	if done {
		if next {
			rl.cycleNextGroup()
			nextGroup := rl.getCurrentGroup()
			nextGroup.goFirstCell()
		} else {
			rl.cyclePreviousGroup()
			prevGroup := rl.getCurrentGroup()
			prevGroup.goLastCell()
		}
	}
}

// writeTabCompletion - Prints all completion groups and their items
func (rl *Instance) writeTabCompletion() {

	// The final completions string to print.
	var completions string

	// This stablizes the completion printing just beyond the input line
	rl.tcUsedY = 0

	// Safecheck
	if !rl.modeTabCompletion {
		return
	}

	// If we are not yet in tab completion mode, this means we just want
	// to print all suggestions, without selecting a candidate yet.
	if !rl.tabCompletionSelect {
		for i, group := range rl.tcGroups {
			if i > 0 {
				switch group.DisplayType {
				case TabDisplayGrid:
					group.tcPosX = 1
				case TabDisplayList, TabDisplayMap:
					group.tcPosY = 1
				}
			}
			completions += group.writeCompletion(rl)
		}
	}

	// Else, we already have some completions printed, and we just want to update.
	// Each group produces its own string, added to the main one.
	if rl.tabCompletionSelect {
		for _, group := range rl.tcGroups {
			completions += group.writeCompletion(rl)
		}
	}

	// Because some completion groups might have more suggestions
	// than what their MaxLength allows them to, cycling sometimes occur,
	// but does not fully clears itself: some descriptions are messed up with.
	// We always clear the screen as a result, between writings.
	print(seqClearScreenBelow)

	// Then we print all of them.
	fmt.Printf(completions)
}

// getTabSearchCompletion - Populates and sets up completion for completion search
func (rl *Instance) getTabSearchCompletion() {
	rl.tcPrefix, rl.tcGroups = rl.TabCompleter(rl.line, rl.pos)
	if len(rl.tcGroups) == 0 {
		return
	}

	rl.tcGroups = checkNilItems(rl.tcGroups) // Avoid nil maps in groups
	rl.getCurrentGroup()                     // Make sure there is a current group

	for _, g := range rl.tcGroups {
		g.updateTabFind(rl)
	}

	// If total number of matches is zero, we directly change the hint, and return
	if comps, _ := rl.getCompletionCount(); comps == 0 {
		rl.hintText = append(rl.hintText, []rune(": no matches (Ctrl-G to cancel)")...)
	}
}

// getHistorySearchCompletion - Populates and sets up completion for command history search
func (rl *Instance) getHistorySearchCompletion() {
	rl.tcGroups = rl.completeHistory() // Refresh full list each time
	if len(rl.tcGroups) == 0 {
		return
	}
	rl.tcGroups = checkNilItems(rl.tcGroups) // Avoid nil maps in groups
	rl.getCurrentGroup()                     // Make sure there is a current group

	if len(rl.tcGroups[0].Suggestions) == 0 {
		rl.histHint = []rune(fmt.Sprintf("%s%s%s %s", DIM, RED, "No command history source, or empty", RESET))
		rl.hintText = rl.histHint
		return
	}
	rl.histHint = []rune(rl.tcGroups[0].Name)

	// Refresh filtered candidates
	rl.tcGroups[0].updateTabFind(rl)

	// If no items matched history, add hint text that we failed to search
	if len(rl.tcGroups[0].Suggestions) == 0 {
		rl.histHint = []rune(rl.tcGroups[0].Name)
		rl.hintText = append(rl.histHint, []rune(": no matches")...)
	}
}

func (rl *Instance) getCurrentGroup() (group *CompletionGroup) {
	for _, g := range rl.tcGroups {
		if g.isCurrent && len(g.Suggestions) > 0 {
			return g
		}
	}
	// We might, for whatever reason, not find one.
	// If there are groups but no current, make first one the king.
	if len(rl.tcGroups) > 0 {
		// Find first group that has list > 0, as another checkup
		for _, g := range rl.tcGroups {
			if len(g.Suggestions) > 0 {
				g.isCurrent = true
				return g
			}
		}
	}
	return
}

// cycleNextGroup - Finds either the first non-empty group,
// or the next non-empty group after the current one.
func (rl *Instance) cycleNextGroup() {
	for i, g := range rl.tcGroups {
		if g.isCurrent {
			g.isCurrent = false
			if i == len(rl.tcGroups)-1 {
				rl.tcGroups[0].isCurrent = true
			} else {
				rl.tcGroups[i+1].isCurrent = true
				// Here, we check if the cycled group is not empty.
				// If yes, cycle to next one now.
				new := rl.getCurrentGroup()
				if len(new.Suggestions) == 0 {
					rl.cycleNextGroup()
				}
			}
			break
		}
	}
}

// cyclePreviousGroup - Same as cycleNextGroup but reverse
func (rl *Instance) cyclePreviousGroup() {
	for i, g := range rl.tcGroups {
		if g.isCurrent {
			g.isCurrent = false
			if i == 0 {
				rl.tcGroups[len(rl.tcGroups)-1].isCurrent = true
			} else {
				rl.tcGroups[i-1].isCurrent = true
				new := rl.getCurrentGroup()
				if len(new.Suggestions) == 0 {
					rl.cyclePreviousGroup()
				}
			}
			break
		}
	}
}

// Check if we have a single completion candidate
func (rl *Instance) hasOneCandidate() bool {
	if len(rl.tcGroups) == 0 {
		return false
	}

	// If one group and one option, obvious
	if len(rl.tcGroups) == 1 {
		cur := rl.getCurrentGroup()
		if cur == nil {
			return false
		}
		if len(cur.Suggestions) == 1 {
			return true
		}
		return false
	}

	// If many groups but only one option overall
	if len(rl.tcGroups) > 1 {
		var count int
		for _, group := range rl.tcGroups {
			for range group.Suggestions {
				count++
			}
		}
		if count == 1 {
			return true
		}
		return false
	}

	return false
}

func (rl *Instance) getCompletionCount() (comps int, lines int) {
	for _, group := range rl.tcGroups {
		comps += len(group.Suggestions)
		if group.tcMaxY > len(group.Suggestions) {
			lines += len(group.Suggestions)
		} else {
			lines += group.tcMaxY
		}
	}
	return
}

// Insert the current completion candidate into the input line.
// This candidate might either be the currently selected one (white frame),
// or the only candidate available, if the total number of candidates is 1.
func (rl *Instance) insertCandidate() {

	cur := rl.getCurrentGroup()

	if cur != nil {
		completion := cur.getCurrentCell(rl)
		prefix := len(rl.tcPrefix)

		// Ensure no indexing error happens with prefix
		if len(completion) >= prefix {
			rl.insert([]rune(completion[prefix:]))
			if !cur.TrimSlash {
				rl.insert([]rune(" "))
			}
		}
	}
}

func (rl *Instance) resetTabCompletion() {
	rl.modeTabCompletion = false
	rl.tabCompletionSelect = false
	rl.compConfirmWait = false
	rl.tcOffset = 0
	rl.tcUsedY = 0
	rl.modeTabFind = false
	rl.modeAutoFind = false
	rl.tfLine = []rune{}

	// Reset tab highlighting
	if len(rl.tcGroups) > 0 {
		for _, g := range rl.tcGroups {
			g.isCurrent = false
		}
		rl.tcGroups[0].isCurrent = true
	}
}
