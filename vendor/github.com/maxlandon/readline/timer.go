package readline

import (
	"context"
	"sync/atomic"
)

// DelayedTabContext is a custom context interface for async updates to the tab completions
type DelayedTabContext struct {
	rl      *Instance
	Context context.Context
	cancel  context.CancelFunc
}

func delayedSyntaxTimer(rl *Instance, i int64) {
	if rl.PasswordMask != 0 || rl.DelayedSyntaxWorker == nil {
		return
	}

	// if len(rl.line)+rl.promptLen > GetTermWidth() {
	//         // line wraps, which is hard to do with random ANSI escape sequences
	//         // so better we don't bother trying.
	//         return
	// }

	// We pass either the current line or the one with the current completion.
	newLine := rl.DelayedSyntaxWorker(rl.getLine())
	var sLine string
	count := atomic.LoadInt64(&rl.delayedSyntaxCount)
	if count != i {
		return
	}

	// Highlight the line again
	if rl.SyntaxHighlighter != nil {
		sLine = rl.SyntaxHighlighter(newLine)
	} else {
		sLine = string(newLine)
	}

	// Save the line as current, and refresh
	rl.updateLine([]rune(sLine))
}

// AppendGroupSuggestions - Given a group name, append a list of completion candidates.
// Any item in the list already existing in the current group's will be ignored.
// If the group is not found with the given name, all suggestions will be dropped, no new group is created.
func (dtc DelayedTabContext) AppendGroupSuggestions(groupName string, suggestions []string) {
	dtc.rl.mutex.Lock()
	defer dtc.rl.mutex.Unlock()

	// Get the group, and return if not found
	var grp *CompletionGroup
	for _, g := range dtc.rl.tcGroups {
		if g.Name == groupName {
			grp = g
		}
	}
	if grp == nil {
		return
	}

	// Add candidate items
	for i := range suggestions {
		select {
		case <-dtc.Context.Done():
			return

		default:
			// Drop duplicates
			for _, actual := range grp.Suggestions {
				if actual == suggestions[i] {
					continue
				}
			}

			// Descriptions might be used by tabdisplay maps or grids, but not lists.
			if grp.DisplayType != TabDisplayList {
				grp.Descriptions[suggestions[i]] = suggestions[i]
			}

			// Suggestions are used by all groups no matter their display type
			grp.Suggestions = append(grp.Suggestions, suggestions[i])
		}
	}

	dtc.rl.clearHelpers()
	dtc.rl.renderHelpers()

}

// AppendGroupAliases - Given a group name, append a list of completion candidates' ALIASES, that is, a second candidate item.
// If any candidate (map index) already exists, its corresponding alias will be overwritten with the new one.
// If any candidate (map index) does not exists yet, it will be added along with its alias.
// If the group is not found with the given name, all suggestions will be dropped, no new group is created.
func (dtc DelayedTabContext) AppendGroupAliases(groupName string, aliases map[string]string) {
	dtc.rl.mutex.Lock()
	defer dtc.rl.mutex.Unlock()

	// Get the group, and return if not found
	var grp *CompletionGroup
	for _, g := range dtc.rl.tcGroups {
		if g.Name == groupName {
			grp = g
		}
	}
	if grp == nil {
		return
	}

	// Add candidate aliases
	for sugg, alias := range aliases {
		select {
		case <-dtc.Context.Done():
			return

		default:
			// Add to suggestions list if not existing yet
			var found bool
			for _, actual := range grp.Suggestions {
				if actual == sugg {
					found = true
				}
			}
			if !found {
				grp.Suggestions = append(grp.Suggestions, sugg)
			}

			// Map the new description anyway
			grp.Aliases[sugg] = alias
		}
	}

	// Reinit all completion groups (recomputes sizes)
	for _, grp := range dtc.rl.tcGroups {
		grp.init(dtc.rl)
	}

	// dtc.rl.clearHelpers()
	// dtc.rl.renderHelpers()
}

// AppendGroupDescriptions - Given a group name, append a list of descriptions to a list of candidates.
// If any candidate (map index) already exists, its corresponding description will be overwritten with the new one.
// If any candidate (map index) does not exists yet, it will be added along with its description.
func (dtc DelayedTabContext) AppendGroupDescriptions(groupName string, descriptions map[string]string) {
	dtc.rl.mutex.Lock()
	defer dtc.rl.mutex.Unlock()

	// Get the group, and return if not found
	var grp *CompletionGroup
	for _, g := range dtc.rl.tcGroups {
		if g.Name == groupName {
			grp = g
		}
	}
	if grp == nil {
		return
	}

	// Add candidate descriptions
	for sugg, desc := range descriptions {
		select {
		case <-dtc.Context.Done():
			return

		default:
			// Add to suggestions list if not existing yet
			var found bool
			for _, actual := range grp.Suggestions {
				if actual == sugg {
					found = true
				}
			}
			if !found {
				grp.Suggestions = append(grp.Suggestions, sugg)
			}

			// Map the new description anyway
			grp.Descriptions[sugg] = desc
		}
	}

	// Reinit all completion groups (recomputes sizes)
	for _, grp := range dtc.rl.tcGroups {
		grp.init(dtc.rl)
	}
	dtc.rl.clearHelpers()
	dtc.rl.renderHelpers()
}

// AppendGroup - Asynchronously add an entire group of completions to the current list
func (dtc DelayedTabContext) AppendGroup(group *CompletionGroup) {
	dtc.rl.mutex.Lock()
	defer dtc.rl.mutex.Unlock()

	// Simply append group to the list
	dtc.rl.tcGroups = append(dtc.rl.tcGroups, group)

	dtc.rl.clearHelpers()
	dtc.rl.renderHelpers()
}
