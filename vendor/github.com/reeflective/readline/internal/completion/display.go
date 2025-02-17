package completion

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/term"
)

// Display prints the current completion list to the screen,
// respecting the current display and completion settings.
func Display(eng *Engine, maxRows int) {
	eng.usedY = 0

	defer fmt.Print(term.ClearScreenBelow)

	// The completion engine might be inactive but still having
	// a non-empty list of completions. This is on purpose, as
	// sometimes it's better to keep completions printed for a
	// little more time. The engine itself is responsible for
	// deleting those lists when it deems them useless.
	if eng.Matches() == 0 || eng.skipDisplay {
		fmt.Print(term.ClearLineAfter)
		return
	}

	// The final completions string to print.
	completions := term.ClearLineAfter

	for _, group := range eng.groups {
		completions += eng.renderCompletions(group)
	}

	// Crop the completions so that it fits within our terminal
	completions, eng.usedY = eng.cropCompletions(completions, maxRows)

	if completions != "" {
		fmt.Print(completions)
	}
}

// Coordinates returns the number of terminal rows used
// when displaying the completions with Display().
func Coordinates(e *Engine) int {
	return e.usedY
}

// renderCompletions renders all completions in a given list (with aliases or not).
// The descriptions list argument is optional.
func (e *Engine) renderCompletions(grp *group) string {
	var builder strings.Builder

	if len(grp.rows) == 0 {
		return ""
	}

	if grp.tag != "" {
		tag := fmt.Sprintf("%s%s%s %s", color.Bold, color.FgYellow, grp.tag, color.Reset)
		builder.WriteString(tag + term.ClearLineAfter + term.NewlineReturn)
	}

	for rowIndex, row := range grp.rows {
		for columnIndex := range grp.columnsWidth {
			var value Candidate

			// If there are aliases, we might have no completions at the current
			// coordinates, so just print the corresponding padding and return.
			if len(row) > columnIndex {
				value = row[columnIndex]
			}

			// Apply all highlightings to the displayed value:
			// selection, prefixes, styles and other things,
			padding := grp.getPad(value, columnIndex, false)
			isSelected := rowIndex == grp.posY && columnIndex == grp.posX && grp.isCurrent
			display := e.highlightDisplay(grp, value, padding, columnIndex, isSelected)

			builder.WriteString(display)

			// Add description if no aliases, or if done with them.
			onLast := columnIndex == len(grp.columnsWidth)-1
			if grp.aliased && onLast && value.Description == "" {
				value = row[0]
			}

			if !grp.aliased || onLast {
				grp.maxDescAllowed = grp.setMaximumSizes(columnIndex)

				descPad := grp.getPad(value, columnIndex, true)
				desc := e.highlightDesc(grp, value, descPad, rowIndex, columnIndex, isSelected)
				builder.WriteString(desc)
			}
		}

		// We're done for this line.
		builder.WriteString(term.ClearLineAfter + term.NewlineReturn)
	}

	return builder.String()
}

func (e *Engine) highlightDisplay(grp *group, val Candidate, pad, col int, selected bool) (candidate string) {
	// An empty display value means padding.
	if val.Display == "" {
		return padSpace(pad)
	}

	reset := color.Fmt(val.Style)
	candidate, padded := grp.trimDisplay(val, pad, col)

	if e.IsearchRegex != nil && e.isearchBuf.Len() > 0 && !selected {
		match := e.IsearchRegex.FindString(candidate)
		match = color.Fmt(color.Bg+"244") + match + color.Reset + reset
		candidate = e.IsearchRegex.ReplaceAllLiteralString(candidate, match)
	}

	if selected {
		// If the comp is currently selected, overwrite any highlighting already applied.
		userStyle := color.UnquoteRC(e.config.GetString("completion-selection-style"))
		selectionHighlightStyle := color.Fmt(color.Bg+"255") + userStyle
		candidate = selectionHighlightStyle + candidate

		if grp.aliased {
			candidate += color.Reset
		}
	} else {
		// Highlight the prefix if any and configured for it.
		if e.config.GetBool("colored-completion-prefix") && e.prefix != "" {
			if prefixMatch, err := regexp.Compile("^" + e.prefix); err == nil {
				prefixColored := color.Bold + color.FgBlue + e.prefix + color.BoldReset + color.FgDefault + reset
				candidate = prefixMatch.ReplaceAllString(candidate, prefixColored)
			}
		}

		candidate = reset + candidate + color.Reset
	}

	return candidate + padded
}

func (e *Engine) highlightDesc(grp *group, val Candidate, pad, row, col int, selected bool) (desc string) {
	if val.Description == "" {
		return color.Reset
	}

	desc, padded := grp.trimDesc(val, pad)

	// If the next row has the same completions, replace the description with our hint.
	if len(grp.rows) > row+1 && grp.rows[row+1][0].Description == val.Description {
		desc = "|"
	} else if e.IsearchRegex != nil && e.isearchBuf.Len() > 0 && !selected {
		match := e.IsearchRegex.FindString(desc)
		match = color.Fmt(color.Bg+"244") + match + color.Reset + color.Dim
		desc = e.IsearchRegex.ReplaceAllLiteralString(desc, match)
	}

	// If the comp is currently selected, overwrite any highlighting already applied.
	// Replace all background reset escape sequences in it, to ensure correct display.
	if row == grp.posY && col == grp.posX && grp.isCurrent && !grp.aliased {
		userDescStyle := color.UnquoteRC(e.config.GetString("completion-selection-style"))
		selectionHighlightStyle := color.Fmt(color.Bg+"255") + userDescStyle
		desc = strings.ReplaceAll(desc, color.BgDefault, userDescStyle)
		desc = selectionHighlightStyle + desc
	}

	compDescStyle := color.UnquoteRC(e.config.GetString("completion-description-style"))

	return compDescStyle + desc + color.Reset + padded
}

// cropCompletions - When the user cycles through a completion list longer
// than the console MaxTabCompleterRows value, we crop the completions string
// so that "global" cycling (across all groups) is printed correctly.
func (e *Engine) cropCompletions(comps string, maxRows int) (cropped string, usedY int) {
	// Get the current absolute candidate position
	absPos := e.getAbsPos()

	// Scan the completions for cutting them at newlines
	scanner := bufio.NewScanner(strings.NewReader(comps))

	// If absPos < MaxTabCompleterRows, cut below MaxTabCompleterRows and return
	if absPos < maxRows-1 {
		return e.cutCompletionsBelow(scanner, maxRows)
	}

	// If absolute > MaxTabCompleterRows, cut above and below and return
	//      -> This includes de facto when we tabCompletionReverse
	if absPos >= maxRows-1 {
		return e.cutCompletionsAboveBelow(scanner, maxRows, absPos)
	}

	return
}

func (e *Engine) cutCompletionsBelow(scanner *bufio.Scanner, maxRows int) (string, int) {
	var count int
	var cropped string

	for scanner.Scan() {
		line := scanner.Text()
		if count < maxRows-1 {
			cropped += line + term.NewlineReturn
			count++
		} else {
			break
		}
	}

	cropped = strings.TrimSuffix(cropped, term.NewlineReturn)

	// Add hint for remaining completions, if any.
	_, used := e.completionCount()
	remain := used - count

	if remain <= 0 {
		return cropped, count - 1
	}

	cropped += fmt.Sprintf(term.NewlineReturn+color.Dim+color.FgYellow+" %d more completion rows... (scroll down to show)"+color.Reset, remain)

	return cropped, count
}

func (e *Engine) cutCompletionsAboveBelow(scanner *bufio.Scanner, maxRows, absPos int) (string, int) {
	cutAbove := absPos - maxRows + 1

	var cropped string
	var count int

	for scanner.Scan() {
		line := scanner.Text()

		if count <= cutAbove {
			count++

			continue
		}

		if count > cutAbove && count <= absPos {
			cropped += line + term.NewlineReturn
			count++
		} else {
			break
		}
	}

	cropped = strings.TrimSuffix(cropped, term.NewlineReturn)
	count -= cutAbove + 1

	// Add hint for remaining completions, if any.
	_, used := e.completionCount()
	remain := used - (maxRows + cutAbove)

	if remain <= 0 {
		return cropped, count - 1
	}

	cropped += fmt.Sprintf(term.NewlineReturn+color.Dim+color.FgYellow+" %d more completion rows... (scroll down to show)"+color.Reset, remain)

	return cropped, count
}
