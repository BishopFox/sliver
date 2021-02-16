package readline

// CompletionGroup - A group/category of items offered to completion, with its own
// name, descriptions and completion display format/type.
// The output, if there are multiple groups available for a given completion input,
// will look like ZSH's completion system.
type CompletionGroup struct {
	Name        string
	Description string

	// Candidates & related
	Suggestions  []string
	Aliases      map[string]string // A candidate has an alternative name (ex: --long, -l option flags)
	Descriptions map[string]string // Items descriptions
	DisplayType  TabDisplayType    // Map, list or normal
	MaxLength    int               // Each group can be limited in the number of comps offered

	// When this is true, the completion is inserted really (not virtually) without
	// the trailing slash, if any. This is used when we want to complete paths.
	TrimSlash bool
	// When this is true, we don't add a space after entering the candidate.
	// Can be used for multi-stage completions, like URLS (scheme:// + host)
	NoSpace bool

	// Values used by the shell
	tcPosX         int
	tcPosY         int
	tcMaxX         int
	tcMaxY         int
	tcOffset       int
	tcMaxLength    int // Used when display is map/list, for determining message width
	tcMaxLengthAlt int // Same as tcMaxLength but for SuggestionsAlt.

	// true if we want to cycle through suggestions because they overflow MaxLength
	allowCycle bool

	// This is to say we are currently cycling through this group, for highlighting choice
	isCurrent bool
}

// init - The completion group computes and sets all its values, and is then ready to work.
func (g *CompletionGroup) init(rl *Instance) {

	// Details common to all displays
	g.checkCycle(rl) // Based on the number of groups given to the shell, allows cycling or not
	g.checkMaxLength(rl)

	// Details specific to tab display modes
	switch g.DisplayType {

	case TabDisplayGrid:
		g.initGrid(rl)
	case TabDisplayMap:
		g.initMap(rl)
	case TabDisplayList:
		g.initList(rl)
	}
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

		// Lists that have a alternative completions are not allowed to have
		// MaxLength set, because rolling does not work yet.
		if g.DisplayType == TabDisplayList {
			g.MaxLength = 1000 // Should be enough not to trigger anything related.
		}
	}

}

// checkNilItems - For each completion group we avoid nil maps and possibly other items
func checkNilItems(groups []*CompletionGroup) (checked []*CompletionGroup) {

	for _, grp := range groups {
		if grp.Descriptions == nil || len(grp.Descriptions) == 0 {
			grp.Descriptions = make(map[string]string)
		}
		if grp.Aliases == nil || len(grp.Aliases) == 0 {
			grp.Aliases = make(map[string]string)
		}
		checked = append(checked, grp)
	}

	return
}

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
	return
}

// getCurrentCell - The completion groups computes the current cell value,
// depending on its display type and its different parameters
func (g *CompletionGroup) getCurrentCell(rl *Instance) string {

	switch g.DisplayType {
	case TabDisplayGrid:
		// x & y coodinates + safety check
		cell := (g.tcMaxX * (g.tcPosY - 1)) + g.tcOffset + g.tcPosX - 1
		if cell < 0 {
			cell = 0
		}

		if cell < len(g.Suggestions) {
			return g.Suggestions[cell]
		}
		return ""

	case TabDisplayMap:
		// x & y coodinates + safety check
		cell := g.tcOffset + g.tcPosY - 1
		if cell < 0 {
			cell = 0
		}

		sugg := g.Suggestions[cell]
		return sugg

	case TabDisplayList:
		// x & y coodinates + safety check
		cell := g.tcOffset + g.tcPosY - 1
		if cell < 0 {
			cell = 0
		}

		sugg := g.Suggestions[cell]

		// If we are in the alt suggestions column, check key and return
		if g.tcPosX == 1 {
			if alt, ok := g.Aliases[sugg]; ok {
				return alt
			}
			return sugg
		}
		return sugg
	}

	// We should never get here
	return ""
}

func (g *CompletionGroup) goFirstCell() {
	switch g.DisplayType {
	case TabDisplayGrid:
		g.tcPosX = 1
		g.tcPosY = 1

	case TabDisplayList:
		g.tcPosX = 0
		g.tcPosY = 1
		g.tcOffset = 0

	case TabDisplayMap:
		g.tcPosX = 0
		g.tcPosY = 1
		g.tcOffset = 0
	}

}

func (g *CompletionGroup) goLastCell() {
	switch g.DisplayType {
	case TabDisplayGrid:
		g.tcPosY = g.tcMaxY

		restX := len(g.Suggestions) % g.tcMaxX
		if restX != 0 {
			g.tcPosX = restX
		} else {
			g.tcPosX = g.tcMaxX
		}

		// We need to adjust the X position depending
		// on the interpretation of the remainder with
		// respect to the group's MaxLength.
		restY := len(g.Suggestions) % g.tcMaxY
		maxY := len(g.Suggestions) / g.tcMaxX
		if restY == 0 && maxY > g.MaxLength {
			g.tcPosX = g.tcMaxX
		}
		if restY != 0 && maxY > g.MaxLength-1 {
			g.tcPosX = g.tcMaxX
		}

	case TabDisplayList:
		// By default, the last item is at maxY
		g.tcPosY = g.tcMaxY

		// If the max length is smaller than the number
		// of suggestions, we need to adjust the offset.
		if len(g.Suggestions) > g.MaxLength {
			g.tcOffset = len(g.Suggestions) - g.tcMaxY
		}

		// We do not take into account the alternative suggestions
		g.tcPosX = 0

	case TabDisplayMap:
		// By default, the last item is at maxY
		g.tcPosY = g.tcMaxY

		// If the max length is smaller than the number
		// of suggestions, we need to adjust the offset.
		if len(g.Suggestions) > g.MaxLength {
			g.tcOffset = len(g.Suggestions) - g.tcMaxY
		}

		// We do not take into account the alternative suggestions
		g.tcPosX = 0
	}
}
