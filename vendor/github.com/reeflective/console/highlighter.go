package console

import (
	"strings"

	"github.com/spf13/cobra"
)

var (
	seqFgGreen  = "\x1b[32m"
	seqFgYellow = "\x1b[33m"
	seqFgReset  = "\x1b[39m"

	seqBrightWigth = "\x1b[38;05;244m"
)

// Base text effects.
var (
	reset      = "\x1b[0m"
	bold       = "\x1b[1m"
	dim        = "\x1b[2m"
	underscore = "\x1b[4m"
	blink      = "\x1b[5m"
	reverse    = "\x1b[7m"

	// Effects reset.
	boldReset       = "\x1b[22m" // 21 actually causes underline instead
	dimReset        = "\x1b[22m"
	underscoreReset = "\x1b[24m"
	blinkReset      = "\x1b[25m"
	reverseReset    = "\x1b[27m"
)

// highlightSyntax - Entrypoint to all input syntax highlighting in the Wiregost console.
func (c *Console) highlightSyntax(input []rune) (line string) {
	// Split the line as shellwords
	args, unprocessed, err := split(string(input), true)
	if err != nil {
		args = append(args, unprocessed)
	}

	highlighted := make([]string, 0)   // List of processed words, append to
	remain := args                     // List of words to process, draw from
	trimmed := trimSpacesMatch(remain) // Match stuff against trimmed words

	// Highlight the root command when found.
	cmd, _, _ := c.activeMenu().Find(trimmed)
	if cmd != nil {
		highlighted, remain = c.highlightCommand(highlighted, args, cmd)
	}

	// Highlight command flags
	highlighted, remain = c.highlightCommandFlags(highlighted, remain, cmd)

	// Done with everything, add remainind, non-processed words
	highlighted = append(highlighted, remain...)

	// Join all words.
	line = strings.Join(highlighted, "")

	return line
}

func (c *Console) highlightCommand(done, args []string, _ *cobra.Command) ([]string, []string) {
	highlighted := make([]string, 0)
	var rest []string

	if len(args) == 0 {
		return done, args
	}

	// Highlight the root command when found, or any of its aliases.
	for _, cmd := range c.activeMenu().Commands() {
		// Change 1: Highlight based on first arg in usage rather than the entire usage itself
		cmdFound := strings.Split(cmd.Use, " ")[0] == strings.TrimSpace(args[0])

		for _, alias := range cmd.Aliases {
			if alias == strings.TrimSpace(args[0]) {
				cmdFound = true
				break
			}
		}

		if cmdFound {
			highlighted = append(highlighted, bold+seqFgGreen+args[0]+seqFgReset+boldReset)
			rest = args[1:]

			return append(done, highlighted...), rest
		}
	}

	return append(done, highlighted...), args
}

func (c *Console) highlightCommandFlags(done, args []string, _ *cobra.Command) ([]string, []string) {
	highlighted := make([]string, 0)
	var rest []string

	if len(args) == 0 {
		return done, args
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			highlighted = append(highlighted, bold+seqBrightWigth+arg+seqFgReset+boldReset)
		} else {
			highlighted = append(highlighted, arg)
		}
	}

	return append(done, highlighted...), rest
}
