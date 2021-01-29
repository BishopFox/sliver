package readline

// moveTabCompletionHighlight - This function is in charge of highlighting the current completion item.
func (rl *Instance) moveTabCompletionHighlight(x, y int) {

	g := rl.getCurrentGroup()

	// If nil, nothing matched input so it amounts to no suggestions.
	// We return right now to avoid dereference.
	if g == nil || g.Suggestions == nil {
		return
	}

	if len(g.Suggestions) == 0 {
		rl.cycleNextGroup()
		g = rl.getCurrentGroup()
	}

	// This is triggered when we need to cycle through the next group
	var done bool

	// Depending on the display, we only keep track of x or (x and y)
	switch g.DisplayType {
	case TabDisplayGrid:
		done = g.moveTabGridHighlight(rl, x, y)

	case TabDisplayList:
		done = g.moveTabListHighlight(x, y)

	case TabDisplayMap:
		done = g.moveTabMapHighlight(x, y)
	}

	// Cycle to next group: we tell them who is the next one to handle highlighting
	if done {
		rl.cycleNextGroup()
	}
}

// moveTabGridHighlight - Moves the highlighting for currently selected completion item (grid display)
func (g *CompletionGroup) moveTabGridHighlight(rl *Instance, x, y int) (done bool) {

	g.tcPosX += x
	g.tcPosY += y

	// Columns
	if g.tcPosX < 1 {
		g.tcPosX = g.tcMaxX
		g.tcPosY--
	}
	if g.tcPosX > g.tcMaxX {
		g.tcPosX = 1
		g.tcPosY++
	}

	// Lines
	if g.tcPosY < 1 {
		g.tcPosY = rl.tcUsedY
	}
	if g.tcPosY > rl.tcUsedY {
		g.tcPosY = 1
		return true
	}

	if (g.tcMaxX*(g.tcPosY-1))+g.tcPosX > len(g.Suggestions) {
		if x < 0 {
			g.tcPosX = len(g.Suggestions) - (g.tcMaxX * (g.tcPosY - 1))
			// return true
		}

		if x > 0 {
			g.tcPosX = 1
			g.tcPosY = 1
			// return true
		}

		if y < 0 {
			g.tcPosY--
			// return true
		}

		if y > 0 {
			g.tcPosY = 1
			// return true
		}

		return true
	}

	return false
}

// moveTabListHighlight - Moves the highlighting for currently selected completion item (list display)
// We don't care about the x, because only can have 2 columns of selectable choices (--long and -s)
func (g *CompletionGroup) moveTabListHighlight(x, y int) (done bool) {

	// We dont' pass to x, because not managed by callers
	g.tcPosY += x

	// Columns (alternative suggestions)
	if g.tcPosX < 1 {
		g.tcPosX = g.tcMaxX
		g.tcPosY--
	}
	if g.tcPosX > g.tcMaxX {
		g.tcPosX = 1
		g.tcPosY++
	}

	// Lines
	if g.tcPosY < 1 {
		g.tcPosY = 1 // We had suppressed it for some time, don't know why.
		g.tcOffset--
	}
	if g.tcPosY > g.tcMaxY {
		g.tcPosY--
		g.tcOffset++
	}

	// Here we must check, in x == 2, that the current choice
	// is not empty. If it is, directly return after setting y value.
	sugg := g.Suggestions[g.tcPosY-1]
	_, ok := g.SuggestionsAlt[sugg]
	if !ok && g.tcPosX == 2 {
		for i, su := range g.Suggestions[g.tcPosY-1:] {
			if _, ok := g.SuggestionsAlt[su]; ok {
				g.tcPosY += i
				return false
			}
		}
	}

	// Setup offset if needs to be.
	if g.tcOffset+g.tcPosY < 1 && len(g.Suggestions) > 0 {
		g.tcPosY = g.tcMaxY
		g.tcOffset = len(g.Suggestions) - g.tcMaxY
	}
	if g.tcOffset < 0 {
		g.tcOffset = 0
	}

	// Once we get to the end of choices: check which column we were selecting.
	if g.tcOffset+g.tcPosY > len(g.Suggestions) {

		// If we have alternative options and that we are not yet
		// completing them, start on top of their column
		if g.tcPosX == 1 && len(g.SuggestionsAlt) > 0 {
			g.tcPosX++
			g.tcPosY = 1
			g.tcOffset = 0
			return false
		}

		// Else no alternatives, return for next group.
		// Reset all values, in case we pass on them again.
		g.tcPosX = 1 // First column
		g.tcPosY = 1 // first row
		g.tcOffset = 0
		return true
	}
	return false
}

// moveTabMapHighlight - Moves the highlighting for currently selected completion item (map display)
func (g *CompletionGroup) moveTabMapHighlight(x, y int) (done bool) {

	g.tcPosY += x
	g.tcPosY += y

	// Lines
	if g.tcPosY < 1 {
		g.tcPosY = 1 // We had suppressed it for some time, don't know why.
		g.tcOffset--
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
		g.tcPosY = 1
		g.tcOffset = 0
		return true
	}
	return false
}
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
