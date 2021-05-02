package gonsole

import (
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// syntaxHighlighter - Entrypoint to all input syntax highlighting in the Wiregost console
func (c *CommandCompleter) syntaxHighlighter(input []rune) (line string) {

	// Format and sanitize input
	args, last, lastWord := formatInputHighlighter(input)

	var highlighted = []string{}
	var remain = args

	// Detect base command automatically
	var command = c.detectedCommand(args)

	// Return input as is
	if noCommandOrEmpty(remain, last, command) {
		return string(input)
	}

	// Recursively analyse commands and their options, etc
	if commandFound(command) {
		highlighted, remain = c.handleSubCommandSyntax(lastWord, highlighted, remain, command, command)
	}

	// Process any expanded variables found
	processed := c.processEnvVars(highlighted, remain)

	// Finally, join all elements with a space: even spaces themselves
	// are in the highlighted list, but as "" each.
	line = strings.Join(processed, " ")

	return
}

func (c *CommandCompleter) handleSubCommandSyntax(lastWord string, processed, args []string, parent, command *flags.Command) (highlighted, remain []string) {

	highlighted, remain = c.highlightCommand(processed, args, command)

	// SubCommand
	if sub, ok := subCommandFound(lastWord, args, command); ok {
		if gCommand := c.console.FindCommand(command.Name); gCommand != nil {
			highlighted, remain = c.handleSubCommandSyntax(lastWord, highlighted, remain, command, sub)
		}
	}

	return
}

func (c *CommandCompleter) highlightCommand(processed, args []string, command *flags.Command) (highlighted, remain []string) {
	var color = c.getTokenHighlighting("{command}")

	for i, arg := range args {
		if arg == command.Name {
			highlighted = append(highlighted, color+arg+readline.RESET)
			remain = args[i+1:]
			break
		}
		highlighted = append(highlighted, arg)
	}

	highlighted = append(processed, highlighted...)

	return
}

// evaluateExpansion - Given a single "word" argument, resolve any embedded expansion variables
func (c *CommandCompleter) evaluateExpansion(arg string) (expanded string) {
	// For each available per-menu expansion variable, evaluate and replace. Any group
	// successfully replacing the token will break the loop, and the remaining expanders will
	// not be evaluated.
	var evaluated = false
	for exp := range c.console.current.expansionComps {
		var color = c.getTokenHighlighting(string(exp))

		if strings.HasPrefix(arg, string(exp)) { // It is an env var.
			if args := strings.Split(arg, "/"); len(args) > 1 {
				var processed = []string{}
				for _, a := range args {
					processed = append(processed, c.evaluateExpansion(a))
					// if strings.HasPrefix(a, string(exp)) && a != " " { // It is an env var.
					//         processed = append(processed, color+a+readline.RESET)
					//         evaluated = true
					//         break
					// }
				}
				expanded = strings.Join(processed, "/")
				evaluated = true
				break
			}
			expanded = color + arg + readline.RESET
			evaluated = true
			break
		}
	}
	if !evaluated {
		expanded = arg
	}
	return
}

// processEnvVars - Highlights environment variables. NOTE: Rewrite with logic from console/env.go
func (c *CommandCompleter) processEnvVars(highlighted []string, remain []string) (processed []string) {

	// Check already processed input
	for _, arg := range highlighted {
		if arg == "" || arg == " " {
			processed = append(processed, arg)
			continue
		}
		processed = append(processed, c.evaluateExpansion(arg))
	}

	// Check remaining args (non-processed)
	for _, arg := range remain {
		if arg == "" || arg == " " {
			processed = append(processed, arg)
			continue
		}

		processed = append(processed, c.evaluateExpansion(arg))
	}

	return
}

func (c *CommandCompleter) getTokenHighlighting(token string) (highlight string) {
	// Get the effect from the config and load it
	if effect, found := c.console.config.Highlighting[token]; found {
		highlight = effect

		if defColor, exists := defaultColorCallbacks[effect]; exists {
			highlight = defColor
		}
	}

	return
}
