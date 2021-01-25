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
	}

	// Not in either search mode, just yield completions
	if !rl.modeAutoFind {
		rl.getNormalCompletion()
	}

	// Populate for History search if in this mode
	if rl.modeAutoFind && rl.searchMode == HistoryFind {
		rl.getHistorySearchCompletion()
	}

	// If no completions available, return
	if len(rl.tcGroups) == 0 {
		return
	}
	rl.tcGroups = checkNilItems(rl.tcGroups) // Avoid nil maps in groups

	// Init/Setup all groups with their priting details
	for _, group := range rl.tcGroups {
		group.init(rl)
	}
}

// writeTabCompletion - Prints all completion groups and their items
func (rl *Instance) writeTabCompletion() {

	// This stablizes the completion printing just beyond the input line
	rl.tcUsedY = 0

	if !rl.modeTabCompletion {
		return
	}

	// Each group produces its own string, added to the main one
	var completions string
	for _, group := range rl.tcGroups {
		completions += group.writeCompletion(rl)
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

// getNormalCompletion - Populates and sets up completion for normal comp mode
func (rl *Instance) getNormalCompletion() {
	rl.tcPrefix, rl.tcGroups = rl.TabCompleter(rl.line, rl.pos)
	if len(rl.tcGroups) == 0 {
		return
	}
	rl.tcGroups = checkNilItems(rl.tcGroups) // Avoid nil maps in groups
	rl.getCurrentGroup()                     // Make sure there is a current group
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
				g.isCurrent = true // Might be used by code not calling here.
				return g
			}
		}
	}
	return
}

func (rl *Instance) resetTabCompletion() {
	rl.modeTabCompletion = false
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
