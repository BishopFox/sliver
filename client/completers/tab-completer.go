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
func TabCompleter(line []rune, pos int) (lastWord string, completions []*readline.CompletionGroup) {

	// Format and sanitize input
	// @args     => All items of the input line
	// @last     => The last word detected in input line as []rune
	// @lastWord => The last word detected in input as string
	args, last, lastWord := FormatInput(line)

	// Detect base command automatically
	var command = detectedCommand(args)

	// Propose commands
	if noCommandOrEmpty(args, last, command) {
		return CompleteMenuCommands(lastWord, pos)
	}

	// Check environment variables
	if envVarAsked(args, lastWord) {
		completeEnvironmentVariables(lastWord)
	}

	// Base command has been identified
	if commandFound(command) {
		// Check environment variables again
		if envVarAsked(args, lastWord) {
			return completeEnvironmentVariables(lastWord)
		}

		// If options are asked for root command, return commpletions.
		if len(command.Groups()) > 0 {
			for _, grp := range command.Groups() {
				if opt, yes := optionArgRequired(args, last, grp); yes {
					return completeOptionArguments(command, opt, lastWord)
				}
			}
		}

		// We must check both for options in this command and any of its subcommands options.
		if sub, ok := subCommandFound(lastWord, args, command); ok {
			if len(sub.Groups()) > 0 {
				for _, grp := range sub.Groups() {
					if opt, yes := optionArgRequired(args, last, grp); yes {
						return completeOptionArguments(sub, opt, lastWord)
					}
				}
			}
		}

		// Then propose subcommands. We don't return from here, otherwise it always skips the next steps.
		if hasSubCommands(command, args) {
			completions = CompleteSubCommands(args, lastWord, command)
		}

		// Handle subcommand if found (maybe we should rewrite this function and use it also for base command)
		if sub, ok := subCommandFound(lastWord, args, command); ok {
			return HandleSubCommand(line, pos, sub)
		}

		// If user asks for completions with "-" / "--", show command options.
		// We ask this here, after having ensured there is no subcommand invoked.
		// This prevails over command arguments, even if they are required.
		if commandOptionsAsked(args, lastWord, command) {
			return CompleteCommandOptions(args, lastWord, command)
		}

		// Propose argument completion before anything, and if needed
		if arg, yes := commandArgumentRequired(lastWord, args, command); yes {
			return completeCommandArguments(command, arg, lastWord)
		}

	}

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
				if grp.Name == cmd.Aliases[0] {
					found = true
					grp.Suggestions = append(grp.Suggestions, cmd.Name+" ")
					grp.Descriptions[cmd.Name+" "] = tui.Dim(cmd.ShortDescription)
				}
			}
			// Add a new group if not found
			if !found {
				grp := &readline.CompletionGroup{
					Name:        cmd.Aliases[0],
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

// CompleteSubCommands - Takes subcommands and gives them as suggestions
// One category, from one source (a parent command).
func CompleteSubCommands(args []string, lastWord string, command *flags.Command) (completions []*readline.CompletionGroup) {

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

	args, last, lastWord := FormatInput(line)

	// Check environment variables
	if envVarAsked(args, lastWord) {
		completeEnvironmentVariables(lastWord)
	}

	// Check argument options
	if len(command.Groups()) > 0 {
		for _, grp := range command.Groups() {
			if opt, yes := optionArgRequired(args, last, grp); yes {
				return completeOptionArguments(command, opt, lastWord)
			}
		}
	}

	// If user asks for completions with "-" or "--". This must take precedence on arguments.
	if subCommandOptionsAsked(args, lastWord, command) {
		return CompleteCommandOptions(args, lastWord, command)
	}

	// If command has non-filled arguments, propose them first
	if arg, yes := commandArgumentRequired(lastWord, args, command); yes {
		return completeCommandArguments(command, arg, lastWord)
	}

	return
}

// CompleteCommandOptions - Yields completion for options of a command, with various decorators
// Many categories, from one source (a command)
func CompleteCommandOptions(args []string, lastWord string, cmd *flags.Command) (prefix string, completions []*readline.CompletionGroup) {

	prefix = lastWord // We only return the PREFIX for readline to correctly show suggestions.

	// Get all (root) option groups.
	groups := cmd.Groups()

	// Append command options not gathered in groups
	groups = append(groups, cmd.Group)

	// For each group, build completions
	for _, grp := range groups {

		compGrp := &readline.CompletionGroup{
			Name:           grp.ShortDescription,
			Descriptions:   map[string]string{},
			DisplayType:    readline.TabDisplayList,
			SuggestionsAlt: map[string]string{},
		}

		// Add each option to completion group
		for _, opt := range grp.Options() {

			// Check if option is already set, next option if yes
			// if optionNotRepeatable(opt) && optionIsAlreadySet(args, lastWord, opt) {
			//         continue
			// }

			// Depending on the current last word, either build a group with option longs only, or with shorts
			if strings.HasPrefix("--"+opt.LongName, lastWord) {
				optName := "--" + opt.LongName
				compGrp.Suggestions = append(compGrp.Suggestions, optName+" ")

				// Add short if there is, and that the prefix is only one dash
				if strings.HasPrefix("-", lastWord) {
					if opt.ShortName != 0 {
						compGrp.SuggestionsAlt[optName+" "] = "-" + string(opt.ShortName) + " "
					}
				}

				// Option default value if any
				var def string
				if len(opt.Default) > 0 {
					def = " (default:"
					for _, d := range opt.Default {
						def += " " + d + ","
					}
					def = strings.TrimSuffix(def, ",")
					def += ")"
				}
				var desc string
				if opt.Required {
					desc = fmt.Sprintf("%s%s R%s %s%s%s%s", tui.RED, tui.DIM, tui.RESET, tui.DIM, opt.Description, def, tui.RESET)
				} else {
					desc = fmt.Sprintf("%s%s O%s %s%s%s%s", tui.GREEN, tui.DIM, tui.RESET, tui.DIM, opt.Description, def, tui.RESET)
				}

				compGrp.Descriptions[optName+" "] = desc
			}
		}

		// No need to add empty groups, will screw the completion system.
		if len(compGrp.Suggestions) > 0 {
			completions = append(completions, compGrp)
		}
	}

	return
}

// RecursiveGroupCompletion - Handles recursive completion for nested option groups
// Many categories, one source (a command's root option group). Called by the function just above.
func RecursiveGroupCompletion(args []string, last []rune, group *flags.Group) (lastWord string, completions []*readline.CompletionGroup) {

	return
}
