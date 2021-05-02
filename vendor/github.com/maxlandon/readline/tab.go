package readline

import (
	"bufio"
	"context"
	"fmt"
	"strings"
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

	// Populate registers if requested.
	if rl.modeAutoFind && rl.searchMode == RegisterFind {
		rl.getRegisterCompletion()
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

// getRegisterCompletion - Populates and sets up completion for Vim registers.
func (rl *Instance) getRegisterCompletion() {

	rl.tcGroups = rl.completeRegisters()
	if len(rl.tcGroups) == 0 {
		return
	}
	rl.tcGroups = checkNilItems(rl.tcGroups) // Avoid nil maps in groups

	// Adjust the index for each group after the first:
	// this ensures no latency when we will move around them.
	for i, group := range rl.tcGroups {
		group.init(rl)
		if i != 0 {
			group.tcPosY = 1
		}
	}

	// If there aren't ANY completion candidates, we
	// escape the completion mode from here directly.
	var items bool
	for _, group := range rl.tcGroups {
		if len(group.Suggestions) > 0 {
			items = true
		}
	}
	if !items {
		rl.modeTabCompletion = false
	}
}

// getTabSearchCompletion - Populates and sets up completion for completion search.
func (rl *Instance) getTabSearchCompletion() {

	// Get completions from the engine, and make sure there is a current group.
	rl.getCompletions()
	if len(rl.tcGroups) == 0 {
		return
	}
	rl.getCurrentGroup()

	// Set the hint for this completion mode
	rl.hintText = append([]rune("Completion search: "), rl.tfLine...)

	// Set the hint for this completion mode
	rl.hintText = append([]rune("Completion search: "), rl.tfLine...)

	for _, g := range rl.tcGroups {
		g.updateTabFind(rl)
	}

	// If total number of matches is zero, we directly change the hint, and return
	if comps, _, _ := rl.getCompletionCount(); comps == 0 {
		rl.hintText = append(rl.hintText, []rune(DIM+RED+" ! no matches (Ctrl-G/Esc to cancel)"+RESET)...)
	}
}

// getHistorySearchCompletion - Populates and sets up completion for command history search
func (rl *Instance) getHistorySearchCompletion() {

	// Refresh full list each time
	rl.tcGroups = rl.completeHistory()
	if len(rl.tcGroups) == 0 {
		return
	}
	rl.tcGroups = checkNilItems(rl.tcGroups) // Avoid nil maps in groups
	rl.getCurrentGroup()                     // Make sure there is a current group

	// The history hint is already set, but overwrite it if we don't have completions
	if len(rl.tcGroups[0].Suggestions) == 0 {
		rl.histHint = []rune(fmt.Sprintf("%s%s%s %s", DIM, RED,
			"No command history source, or empty (Ctrl-G/Esc to cancel)", RESET))
		rl.hintText = rl.histHint
		return
	}

	// Set the hint line with everything
	rl.histHint = append([]rune("\033[38;5;183m"+string(rl.histHint)+RESET), rl.tfLine...)
	rl.histHint = append(rl.histHint, []rune(RESET)...)
	rl.hintText = rl.histHint

	// Refresh filtered candidates
	rl.tcGroups[0].updateTabFind(rl)

	// If no items matched history, add hint text that we failed to search
	if len(rl.tcGroups[0].Suggestions) == 0 {
		rl.hintText = append(rl.histHint, []rune(DIM+RED+" ! no matches (Ctrl-G/Esc to cancel)"+RESET)...)
		return
	}
}

// getNormalCompletion - Populates and sets up completion for normal comp mode.
// Will automatically cancel the completion mode if there are no candidates.
func (rl *Instance) getNormalCompletion() {

	// Get completions groups, pass delayedTabContext and check nils
	rl.getCompletions()

	// Adjust the index for each group after the first:
	// this ensures no latency when we will move around them.
	for i, group := range rl.tcGroups {
		group.init(rl)
		if i != 0 {
			group.tcPosY = 1
		}
	}

	// If there aren't ANY completion candidates, we
	// escape the completion mode from here directly.
	var items bool
	for _, group := range rl.tcGroups {
		if len(group.Suggestions) > 0 {
			items = true
		}
	}
	if !items {
		rl.modeTabCompletion = false
	}
}

// getCompletions - Calls the completion engine/function to yield a list of 0 or more completion groups,
// sets up a delayed tab context and passes it on to the tab completion engine function, and ensure no
// nil groups/items will pass through. This function is called by different comp search/nav modes.
func (rl *Instance) getCompletions() {

	// If there is no wired tab completion engine, nothing we can do.
	if rl.TabCompleter == nil {
		return
	}

	// Cancel any existing tab context first.
	if rl.delayedTabContext.cancel != nil {
		rl.delayedTabContext.cancel()
	}

	// Recreate a new context
	rl.delayedTabContext = DelayedTabContext{rl: rl}
	rl.delayedTabContext.Context, rl.delayedTabContext.cancel = context.WithCancel(context.Background())

	// Get the correct line to be completed, and the current cursor position
	compLine, compPos := rl.getCompletionLine()

	// Call up the completion engine/function to yield completion groups
	rl.tcPrefix, rl.tcGroups = rl.TabCompleter(compLine, compPos, rl.delayedTabContext)

	// Avoid nil maps in groups. Maybe we could also pop any empty group.
	rl.tcGroups = checkNilItems(rl.tcGroups)

	// We have been loading fresh completion sin this function,
	// so adjust the positions for each group, so that cycling
	// correctly occurs in both directions (tab/shift+tab)
	for i, group := range rl.tcGroups {
		if i > 0 {
			switch group.DisplayType {
			case TabDisplayGrid:
				group.tcPosX = 1
			case TabDisplayList, TabDisplayMap:
				group.tcPosY = 1
			}
		}
	}
}

// moveTabCompletionHighlight - This function is in charge of
// computing the new position in the current completions liste.
func (rl *Instance) moveTabCompletionHighlight(x, y int) {

	g := rl.getCurrentGroup()

	// If there is no current group, we leave any current completion mode.
	if g == nil || g.Suggestions == nil {
		rl.modeTabCompletion = false
		return
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

	// In any case, we write the completions strings, trimmed for redundant
	// newline occurences that have been put at the end of each group.
	for _, group := range rl.tcGroups {
		completions += group.writeCompletion(rl)
	}

	// Because some completion groups might have more suggestions
	// than what their MaxLength allows them to, cycling sometimes occur,
	// but does not fully clears itself: some descriptions are messed up with.
	// We always clear the screen as a result, between writings.
	print(seqClearScreenBelow)

	// Crop the completions so that it fits within our MaxTabCompleterRows
	completions, rl.tcUsedY = rl.cropCompletions(completions)

	// Then we print all of them.
	fmt.Printf(completions)
}

// cropCompletions - When the user cycles through a completion list longer
// than the console MaxTabCompleterRows value, we crop the completions string
// so that "global" cycling (across all groups) is printed correctly.
func (rl *Instance) cropCompletions(comps string) (cropped string, usedY int) {

	// If we actually fit into the MaxTabCompleterRows, return the comps
	if rl.tcUsedY < rl.MaxTabCompleterRows {
		return comps, rl.tcUsedY
	}

	// Else we go on, but we have more comps than what allowed:
	// we will add a line to the end of the comps, giving the actualized
	// number of completions remaining and not printed
	var moreComps = func(cropped string, offset int) (hinted string, noHint bool) {
		_, _, adjusted := rl.getCompletionCount()
		remain := adjusted - offset
		if remain == 0 {
			return cropped, true
		}
		hint := fmt.Sprintf(DIM+YELLOW+" %d more completions... (scroll down to show)"+RESET+"\n", remain)
		hinted = cropped + hint
		return hinted, false
	}

	// Get the current absolute candidate position (prev groups x suggestions + curGroup.tcPosY)
	var absPos = rl.getAbsPos()

	// Get absPos - MaxTabCompleterRows for having the number of lines to cut at the top
	// If the number is negative, that means we don't need to cut anything at the top yet.
	var maxLines = absPos - rl.MaxTabCompleterRows
	if maxLines < 0 {
		maxLines = 0
	}

	// Scan the completions for cutting them at newlines
	scanner := bufio.NewScanner(strings.NewReader(comps))

	// If absPos < MaxTabCompleterRows, cut below MaxTabCompleterRows and return
	if absPos <= rl.MaxTabCompleterRows {
		var count int
		for scanner.Scan() {
			line := scanner.Text()
			if count < rl.MaxTabCompleterRows {
				cropped += line + "\n"
				count++
			} else {
				count++
				break
			}
		}
		cropped, _ = moreComps(cropped, count)
		return cropped, count
	}

	// If absolute > MaxTabCompleterRows, cut above and below and return
	//      -> This includes de facto when we tabCompletionReverse
	if absPos > rl.MaxTabCompleterRows {
		cutAbove := absPos - rl.MaxTabCompleterRows
		var count int
		for scanner.Scan() {
			line := scanner.Text()
			if count < cutAbove {
				count++
				continue
			}
			if count >= cutAbove && count < absPos {
				cropped += line + "\n"
				count++
			} else {
				count++
				break
			}
		}
		cropped, _ := moreComps(cropped, rl.MaxTabCompleterRows+cutAbove)
		return cropped, count - cutAbove
	}

	return
}

func (rl *Instance) getAbsPos() int {
	var prev int
	var foundCurrent bool
	for _, grp := range rl.tcGroups {
		if grp.isCurrent {
			prev += grp.tcPosY + 1 // + 1 for title
			foundCurrent = true
			break
		} else {
			prev += grp.tcMaxY + 1 // + 1 for title
		}
	}

	// If there was no current group, it means
	// we showed completions but there is no
	// candidate selected yet, return 0
	if !foundCurrent {
		return 0
	}
	return prev
}

// We pass a special subset of the current input line, so that
// completions are available no matter where the cursor is.
func (rl *Instance) getCompletionLine() (line []rune, pos int) {

	pos = rl.pos - len(rl.currentComp)
	if pos < 0 {
		pos = 0
	}

	switch {
	case rl.pos == len(rl.line):
		line = rl.line
	case rl.pos < len(rl.line):
		line = rl.line[:pos]
	default:
		line = rl.line
	}

	return
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

// When the completions are either longer than:
// - The user-specified max completion length
// - The terminal lengh
// we use this function to prompt for confirmation before printing comps.
func (rl *Instance) promptCompletionConfirm(sentence string) {
	rl.hintText = []rune(sentence)

	rl.compConfirmWait = true
	rl.viUndoSkipAppend = true

	rl.renderHelpers()
}

func (rl *Instance) getCompletionCount() (comps int, lines int, adjusted int) {
	for _, group := range rl.tcGroups {
		comps += len(group.Suggestions)
		// if group.Name != "" {
		adjusted++ // Title
		// }
		if group.tcMaxY > len(group.Suggestions) {
			lines += len(group.Suggestions)
			adjusted += len(group.Suggestions)
		} else {
			lines += group.tcMaxY
			adjusted += group.tcMaxY
		}
	}
	return
}

func (rl *Instance) resetTabCompletion() {
	rl.modeTabCompletion = false
	rl.tabCompletionSelect = false
	rl.compConfirmWait = false

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
