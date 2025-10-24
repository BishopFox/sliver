package line

import (
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var (
    // Base text effects.
	Reset      = "\x1b[0m"
	Bold       = "\x1b[1m"
	Dim        = "\x1b[2m"
	Underscore = "\x1b[4m"
	Blink      = "\x1b[5m"
	Reverse    = "\x1b[7m"

	// Effects reset.
	BoldReset       = "\x1b[22m" // 21 actually causes underline instead
	DimReset        = "\x1b[22m"
	UnderscoreReset = "\x1b[24m"
	BlinkReset      = "\x1b[25m"
	ReverseReset    = "\x1b[27m"

    // Colors
	GreenFG  = "\x1b[32m"
	YellowFG = "\x1b[33m"
	ResetFG  = "\x1b[39m"
	BrightWhiteFG = "\x1b[38;05;244m"
)

// HighlightCommand applies highlighting to commands in an input line.
func HighlightCommand(done, args []string, root *cobra.Command, cmdColor string) ([]string, []string) {
	highlighted := make([]string, 0)
	var rest []string

	if len(args) == 0 {
		return done, args
	}

	// Highlight the root command when found, or any of its aliases.
	for _, cmd := range root.Commands() {
		// Change 1: Highlight based on first arg in usage rather than the entire usage itself
		cmdFound := strings.Split(cmd.Use, " ")[0] == strings.TrimSpace(args[0])

        if slices.Contains(cmd.Aliases, strings.TrimSpace(args[0])) {
				cmdFound = true
				break
		}

		if cmdFound {
			highlighted = append(highlighted, Bold+cmdColor+args[0]+ResetFG+BoldReset)
			rest = args[1:]

			return append(done, highlighted...), rest
		}
	}

	return append(done, highlighted...), args
}

// HighlightCommand applies highlighting to command flags in an input line.
func HighlightCommandFlags(done, args []string, flagColor string) ([]string, []string) {
	highlighted := make([]string, 0)
	var rest []string

	if len(args) == 0 {
		return done, args
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			highlighted = append(highlighted, Bold+flagColor+arg+ResetFG+BoldReset)
		} else {
			highlighted = append(highlighted, arg)
		}
	}

	return append(done, highlighted...), rest
}
