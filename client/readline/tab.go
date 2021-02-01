package readline

import (
	"fmt"

	"github.com/evilsocket/islazy/tui"
)

// This file gathers all alterative tab completion functions, therefore is not separated in files like
// tabgrid.go, tabmap.go, etc., because in this new setup such a structure and distinction is now irrelevant.

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
// dealing with all search and completion modes.
func (rl *Instance) getTabCompletion() {
	rl.tcOffset = 0

	if rl.TabCompleter == nil {
		return // No completions to offer
	}

	// Populate for completion search if in this mode
	if rl.searchMode == CompletionFind {
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
	// rl.tcPrefix, rl.tcGroups = rl.TabCompleter(rl.line, rl.pos)
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
			if i != 0 {
				group.tcPosY = 1
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
		rl.histHint = []rune(fmt.Sprintf("%s%s%s %s", tui.DIM, tui.RED, "No command history source, or empty", tui.RESET))
		rl.hintText = rl.histHint
		return
	}
	rl.histHint = []rune(rl.tcGroups[0].Name)

	// if rl.regexSearch.String() != "(?i)" {
	rl.tcGroups[0].updateTabFind(rl) // Refresh filtered candidates
	// }

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

func (rl *Instance) resetTabCompletion() {
	rl.modeTabCompletion = false
	rl.tabCompletionSelect = false
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

// checkNilItems - For each completion group we avoid nil maps and possibly other items
func checkNilItems(groups []*CompletionGroup) (checked []*CompletionGroup) {

	for _, grp := range groups {
		if grp.Descriptions == nil || len(grp.Descriptions) == 0 {
			grp.Descriptions = make(map[string]string)
		}
		if grp.SuggestionsAlt == nil || len(grp.SuggestionsAlt) == 0 {
			grp.SuggestionsAlt = make(map[string]string)
		}
		checked = append(checked, grp)
	}

	return
}

func (rl *Instance) hasOneCandidate() bool {
	cur := rl.getCurrentGroup()

	// If one group and one option, obvious
	if len(rl.tcGroups) == 1 && len(cur.Suggestions) == 1 {
		return true
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
