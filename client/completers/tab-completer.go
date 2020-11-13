package completers

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"strings"

	"github.com/evilsocket/islazy/tui"
	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/context"
)

// TabCompleter - Entrypoint to all tab completions in the Wiregost console.
func TabCompleter(line []rune, pos int) (prefix string, completions []*readline.CompletionGroup) {

	// Format and sanitize input
	// @args     => All items of the input line
	// @last     => The last word detected in input line as []rune
	// @lastWord => The last word detected in input as string
	args, last, lastWord := FormatInput(line)
	prefix = lastWord

	// Detect base command automatically
	var command = detectedCommand(args)

	// Propose commands
	if noCommandOrEmpty(args, last, command) {
		return CompleteMenuCommands(lastWord, pos)
	}

	// Check environment variables
	if envVarAsked(args, lastWord) {

	}

	// Base command has been identified
	if commandFound(command) {

		// If user asks for completions with "-" / "--", show command options
		if optionsAsked(args, lastWord, command) {
			return CompleteCommandOptions(args, lastWord, command)
		}

		// Check environment variables again
		if envVarAsked(args, lastWord) {

		}

		// Propose argument completion before anything, and if needed
		if _, yes := argumentRequired(lastWord, args, command, false); yes { // add *commands.Context.Menu in the string here

		}

		// Then propose subcommands. We don't return from here, otherwise it always skips the next steps.
		if hasSubCommands(command, args) {
			prefix, completions = CompleteSubCommands(args, lastWord, command)
		}

		// Handle subcommand if found (maybe we should rewrite this function and use it also for base command)
		if sub, ok := subCommandFound(lastWord, args, command); ok {
			return HandleSubCommand(line, pos, sub)
		}
	}

	// -------------------- IMPORTANT ------------------------
	// WE NEED TO PASS A DEEP COPY OF THE OBJECTS: OTHERWISE THE COMPLETION SEARCH FUNCTION WILL MESS UP WITH THEM.

	return
}

// [ Main Completion Functions ] -----------------------------------------------------------------------------------------------------------------

// CompleteMenuCommands - Selects all commands available in a given context and returns them as suggestions
// Many categories, all from command parsers.
func CompleteMenuCommands(lastWord string, pos int) (prefix string, completions []*readline.CompletionGroup) {

	prefix = lastWord        // We only return the PREFIX for readline to correctly show suggestions.
	var parser *flags.Parser // Current context parser

	// Gather all root commands bound to current menu context
	switch context.Context.Menu {
	case context.Server:
		parser = commands.Server
	case context.Sliver:
		parser = commands.Sliver
	}

	// Check their namespace (which should be their "group" (like utils, core, Jobs, etc))
	for _, cmd := range parser.Commands() {
		// If command matches readline input
		if strings.HasPrefix(cmd.Name, lastWord) {
			// Check command group: add to existing group if found
			var found bool
			for _, grp := range completions {
				if grp.Name == cmd.Namespace {
					found = true
					grp.Suggestions = append(grp.Suggestions, cmd.Name+" ")
					grp.Descriptions[cmd.Name+" "] = tui.Dim(cmd.ShortDescription)
				}
			}
			// Add a new group if not found
			if !found {
				grp := &readline.CompletionGroup{
					Name:        cmd.Namespace,
					Suggestions: []string{cmd.Name + " "},
					Descriptions: map[string]string{
						cmd.Name + " ": tui.Dim(cmd.ShortDescription),
					},
				}
				completions = append(completions, grp)
			}
		}
	}

	// Make adjustments to the CompletionGroup list: set maxlength depending on items, check descriptions, etc.
	for _, grp := range completions {
		// If the length of suggestions is too long and we have
		// many groups, use grid display.
		if len(completions) >= 10 && len(grp.Suggestions) >= 10 {
			grp.DisplayType = readline.TabDisplayGrid
		} else {
			// By default, we use a map of command to descriptions
			grp.DisplayType = readline.TabDisplayList
		}
	}

	return
}

// CompleteCommandArguments - Completes all values for arguments to a command. Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple contexts
func CompleteCommandArguments(cmd *flags.Command, arg string, line []rune, pos int) (lastWord string, completions []*readline.CompletionGroup) {

	_, _, lastWord = FormatInput(line)

	return
}

// CompleteSubCommands - Takes subcommands and gives them as suggestions
// One category, from one source (a parent command).
func CompleteSubCommands(args []string, lastWord string, command *flags.Command) (prefix string, completions []*readline.CompletionGroup) {

	prefix = lastWord // We only return the PREFIX for readline to correctly show suggestions.

	group := &readline.CompletionGroup{
		Name:         command.Name,
		Suggestions:  []string{},
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	for _, sub := range command.Commands() {
		if strings.HasPrefix(sub.Name, lastWord) {
			group.Suggestions = append(group.Suggestions, sub.Name+" ")
			group.Descriptions[sub.Name+" "] = tui.DIM + sub.ShortDescription + tui.RESET
		}
	}

	completions = append(completions, group)

	return
}

// HandleSubCommand - Handles completion for subcommand options and arguments, + any option value related completion
// Many categories, from many sources: this function calls the same functions as the ones previously called for completing its parent command.
func HandleSubCommand(line []rune, pos int, command *flags.Command) (lastWord string, completions []*readline.CompletionGroup) {

	args, _, lastWord := FormatInput(line)

	// Check environment variables
	// if envVarAsked(args, last) {
	//         return CompleteEnvironmentVariables(line, pos)
	// }

	// If command has non-filled arguments, propose them first
	if _, yes := argumentRequired(lastWord, args, command, true); yes {
		// if arg, yes := argumentRequired(lastWord, args, context.Context.Menu, command, true); yes {
		// _, suggestions, listSuggestions, tabType = CompleteCommandArguments(command, arg, line, pos)
	}

	// If user asks for completions with "-" or "--". (Note: This takes precedence on arguments, as it is evaluated after arguments)
	if optionsAsked(args, lastWord, command) {
		return CompleteCommandOptions(args, lastWord, command)
	}

	return
}

// CompleteCommandOptions - Yields completion for options of a command, with various decorators
// Many categories, from one source (a command)
func CompleteCommandOptions(args []string, lastWord string, cmd *flags.Command) (prefix string, completions []*readline.CompletionGroup) {

	prefix = lastWord // We only return the PREFIX for readline to correctly show suggestions.

	// Get all option groups
	groups := cmd.Groups()

	// For each group, build completions
	for _, grp := range groups {

		compGrp := &readline.CompletionGroup{
			Name:         grp.LongDescription,
			Descriptions: map[string]string{},
			DisplayType:  readline.TabDisplayList,
		}

		// Add each option to completion group
		for _, opt := range grp.Options() {

			// Check if option is already set, next option if yes
			if optionNotRepeatable(opt) && optionIsAlreadySet(args, lastWord, opt) {
				continue
			}

			if strings.HasPrefix("--"+opt.LongName, lastWord) {
				optName := "--" + opt.LongName
				compGrp.Suggestions = append(compGrp.Suggestions, optName+" ")

				var desc string
				if opt.Required {
					desc = fmt.Sprintf("%s%sR%s %s%s%s", tui.RED, tui.DIM, tui.RESET, tui.DIM, opt.Description, tui.RESET)
				} else {
					desc = fmt.Sprintf("%s%sO%s %s%s%s", tui.GREEN, tui.DIM, tui.RESET, tui.DIM, opt.Description, tui.RESET)
				}

				compGrp.Descriptions[optName+" "] = desc
			}
		}

		completions = append(completions, compGrp)
	}

	return
}

// RecursiveGroupCompletion - Handles recursive completion for nested option groups
// Many categories, one source (a command's root option group). Called by the function just above.
func RecursiveGroupCompletion(args []string, last []rune, group *flags.Group) (lastWord string, completions []*readline.CompletionGroup) {

	return
}
