package gonsole

import (
	"fmt"
	"regexp"
	"strings"
)

// Run - Start the console application (readline loop). Blocking.
// The error returned will always be an error that the console
// application does not understand or cannot handle.
func (c *Console) Run() (err error) {

	for {
		// Recompute the prompt for the current menu
		// First check and pull from configuration.
		c.current.Prompt.refreshPromptSettings()

		// Set the shell history sources with menu ones
		c.shell.SetHistoryCtrlR(c.current.historyCtrlRName, c.current.historyCtrlR)
		c.shell.SetHistoryAltR(c.current.historyAltRName, c.current.historyAltR)

		// Instantiate and bind all commands for the current
		// menu, respecting any filter used to hide some of them.
		c.bindCommands()

		// Run user-provided pre-loop hooks
		c.runPreLoopHooks()

		// Leave a newline before redrawing the prompt
		if c.LeaveNewline {
			fmt.Println()
		}

		// Block and read user input. Provides completion, syntax, hints, etc.
		// Various types of errors might arise from here. We handle them
		// in a special function, where we can specify behavior for certain errors.
		line, err := c.shell.Readline()
		if err != nil {
			// Handle readline errors in a specialized function
		}

		// If the menu prompt is asked to leave a newline
		// between prompt and output, we print it now.
		if c.PreOutputNewline {
			fmt.Println()
		}

		// The line might need some sanitization, like removing empty/redundant spaces,
		// but also in case where there are weird slashes and other kung-fu bombs.
		args, empty := c.sanitizeInput(line)
		if empty {
			continue
		}

		// Run user-provided pre-run line hooks, which may modify the input line
		args = c.runLineHooks(args)

		// Parse any special rune tokens, before trying to evaluate any expanded variable.
		tokenParsed, err := c.parseTokens(args)
		if err != nil {
			tokenParsed = args
		}

		// Parse the input line for any expanded variables, and evaluate them.
		args, err = c.parseAllExpansionVariables(tokenParsed)
		if err != nil {
			fmt.Println(warn+"Failed to evaluate expanded variables: %s", err.Error())
		}

		// Run user-provided pre-run hooks
		c.runPreRunHooks()

		// We then pass the processed command line to the command parser,
		// where any error arising from parsing or execution will be handled.
		// Thus we don't need to handle any error here.
		c.execute(args)
	}
}

func (c *Console) runPreLoopHooks() {
	for _, hook := range c.PreLoopHooks {
		hook()
	}
}

func (c *Console) runPreRunHooks() {
	for _, hook := range c.PreRunHooks {
		hook()
	}
}

func (c *Console) runLineHooks(args []string) (processed []string) {
	// By default, pass args as they are
	processed = args

	// Or modify them again
	for _, hook := range c.PreRunLineHooks {
		processed, _ = hook(processed)
	}
	return
}

// sanitizeInput - Trims spaces and other unwished elements from the input line.
func (c *Console) sanitizeInput(line string) (sanitized []string, empty bool) {

	// Assume the input is not empty
	empty = false

	// Trim border spaces
	trimmed := strings.TrimSpace(line)
	if len(line) < 1 {
		empty = true
		return
	}

	// Parse arguments for quotes, and split according to these quotes first:
	// they might influence heavily on the go-flags argument parsing done further
	// Split all strings with '' and ""
	r := regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)'`)
	unfiltered := r.FindAllString(trimmed, -1)

	var test []string
	for _, arg := range unfiltered {
		if strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'") {
			trim := strings.TrimPrefix(arg, "'")
			trim = strings.TrimSuffix(trim, "'")
			test = append(test, trim)
			continue
		}
		test = append(test, arg)
	}

	// Catch any eventual empty items
	for _, arg := range test {
		// for _, arg := range unfiltered {
		if arg != "" {
			sanitized = append(sanitized, arg)
		}
	}

	return
}
