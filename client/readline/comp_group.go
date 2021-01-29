package readline

// CompletionGroup - A group/category of items offered to completion, with its own
// name, descriptions and completion display format/type.
// The output, if there are multiple groups available for a given completion input,
// will look like ZSH's completion system.
type CompletionGroup struct {
	Name        string
	Description string

	// Same as readline old system
	Suggestions  []string
	Descriptions map[string]string // Items descriptions
	DisplayType  TabDisplayType    // Map, list or normal
	MaxLength    int               // Each group can be limited in the number of comps offered

	// Alternative suggestions: used when a candidate has an alternative name
	// this applies to options, when both short and long flags are used.
	// Index is Suggestion
	SuggestionsAlt map[string]string

	// Values used by the shell
	tcPosX         int
	tcPosY         int
	tcMaxX         int
	tcMaxY         int
	tcOffset       int
	tcMaxLength    int // Used when display is map/list, for determining message width
	tcMaxLengthAlt int // Same as tcMaxLength but for SuggestionsAlt.

	// allowCycle - is true if we want to cycle through suggestions because they overflow MaxLength
	// This is set by the shell when it has detected this group is alone in the suggestions.
	// Might be the case of things like remote processes .
	allowCycle bool
	isCurrent  bool // This is to say we are currently cycling through this group, for highlighting choice
}

// updateTabFind - When searching through all completion groups (whether it be command history or not),
// we ask each of them to filter its own items and return the results to the shell for aggregating them.
// The rx parameter is passed, as the shell already checked that the search pattern is valid.
func (g *CompletionGroup) updateTabFind(rl *Instance) {

	suggs := make([]string, 0)

	// We perform filter right here, so we create a new completion group, and populate it with our results.
	for i := range g.Suggestions {
		if rl.regexSearch.MatchString(g.Suggestions[i]) {
			suggs = append(suggs, g.Suggestions[i])
		} else if g.DisplayType == TabDisplayList && rl.regexSearch.MatchString(g.Descriptions[g.Suggestions[i]]) {
			// this is a list so lets also check the descriptions
			suggs = append(suggs, g.Suggestions[i])
		}
	}

	// We overwrite the group's items, (will be refreshed as soon as something is typed in the search)
	g.Suggestions = suggs

	// Finally, the group computes its new printing settings
	g.init(rl)
}

// checkCycle - Based on the number of groups given to the shell, allows cycling or not
func (g *CompletionGroup) checkCycle(rl *Instance) {

	if len(rl.tcGroups) == 1 {
		g.allowCycle = true
	}

	if len(rl.tcGroups) >= 10 {
		g.allowCycle = false
	}

}

// checkMaxLength - Based on the number of groups given to the shell, check/set MaxLength defaults
func (g *CompletionGroup) checkMaxLength(rl *Instance) {

	// This means the user forgot to set it
	if g.MaxLength == 0 {
		if len(rl.tcGroups) < 5 {
			g.MaxLength = 20
		}

		if len(rl.tcGroups) >= 5 {
			g.MaxLength = 20
		}
	}
}

// getCurrentCell - The completion groups computes the current cell value,
// depending on its display type and its different parameters
func (g *CompletionGroup) getCurrentCell() string {

	switch g.DisplayType {
	case TabDisplayGrid, TabDisplayMap:
		// x & y coodinates
		cell := (g.tcMaxX * (g.tcPosY - 1)) + g.tcOffset + g.tcPosX - 1
		if cell < len(g.Suggestions) {
			return g.Suggestions[cell]
		}
		return ""

	case TabDisplayList:
		// The current y gives us the correct key, at least
		sugg := g.Suggestions[g.tcOffset+g.tcPosY-1]

		// If we are in the alt suggestions column, check key and return
		if g.tcPosX == 2 {
			if alt, ok := g.SuggestionsAlt[sugg]; ok {
				return alt
			}
			return sugg // return key in case of failure
		}
		return sugg // Else return the suggestion itself
	}

	// We should NEVER get here
	return ""
}
