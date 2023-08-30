package completion

import (
	"bufio"
	"fmt"
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
		completions += group.writeComps(eng)
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
