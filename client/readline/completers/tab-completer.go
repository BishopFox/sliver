package completers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// CommandCompleter - A completer using a github.com/jessevdk/go-flags Command Parser, in order
// to build completions for commands, arguments, options and their arguments as well.
// This completer needs to be instantiated with its constructor, in order to ensure the parser is not nil.
type CommandCompleter struct {
	parser *flags.Parser
}

// NewCommandCompleter - Instantiate a new tab completer using a github.com/jessevdk/go-flags Command Parser.
func NewCommandCompleter(parser *flags.Parser) (completer *CommandCompleter, err error) {
	if parser == nil {
		return nil, errors.New("command completer was instantiated with a nil parser")
	}
	return &CommandCompleter{parser: parser}, nil
}

// TabCompleter - A default tab completer working with a github.com/jessevdk/go-flags parser.
func (c *CommandCompleter) TabCompleter(line []rune, pos int) (lastWord string, completions []*readline.CompletionGroup) {

	// Format and sanitize input
	// @args     => All items of the input line
	// @last     => The last word detected in input line as []rune
	// @lastWord => The last word detected in input as string
	args, last, lastWord := formatInput(line)

	// Detect base command automatically
	var command = c.detectedCommand(args)

	// Propose commands
	if noCommandOrEmpty(args, last, command) {
		return c.completeMenuCommands(lastWord, pos)
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

		// Then propose subcommands. We don't return from here, otherwise it always skips the next steps.
		if hasSubCommands(command, args) {
			completions = completeSubCommands(args, lastWord, command)
		}

		// Handle subcommand if found (maybe we should rewrite this function and use it also for base command)
		if sub, ok := subCommandFound(lastWord, args, command); ok {
			return handleSubCommand(line, pos, sub)
		}

		// If user asks for completions with "-" / "--", show command options.
		// We ask this here, after having ensured there is no subcommand invoked.
		// This prevails over command arguments, even if they are required.
		if commandOptionsAsked(args, lastWord, command) {
			return completeCommandOptions(args, lastWord, command)
		}

		// Propose argument completion before anything, and if needed
		if arg, yes := commandArgumentRequired(lastWord, args, command); yes {
			return completeCommandArguments(command, arg, lastWord)
		}

	}

	return
}

// [ Main Completion Functions ] -----------------------------------------------------------------------------------------------------------------

// completeMenuCommands - Selects all commands available in a given context and returns them as suggestions
// Many categories, all from command parsers.
func (c *CommandCompleter) completeMenuCommands(lastWord string, pos int) (prefix string, completions []*readline.CompletionGroup) {

	prefix = lastWord // We only return the PREFIX for readline to correctly show suggestions.

	// Check their namespace (which should be their "group" (like utils, core, Jobs, etc))
	for _, cmd := range c.parser.Commands() {
		// If command matches readline input
		if strings.HasPrefix(cmd.Name, lastWord) {
			// Check command group: add to existing group if found
			var found bool
			for _, grp := range completions {
				if grp.Name == cmd.Aliases[0] {
					found = true
					grp.Suggestions = append(grp.Suggestions, cmd.Name)
					grp.Descriptions[cmd.Name] = readline.Dim(cmd.ShortDescription)
				}
			}
			// Add a new group if not found
			if !found {
				grp := &readline.CompletionGroup{
					Name:        cmd.Aliases[0],
					Suggestions: []string{cmd.Name},
					Descriptions: map[string]string{
						cmd.Name: readline.Dim(cmd.ShortDescription),
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

// completeSubCommands - Takes subcommands and gives them as suggestions
// One category, from one source (a parent command).
func completeSubCommands(args []string, lastWord string, command *flags.Command) (completions []*readline.CompletionGroup) {

	group := &readline.CompletionGroup{
		Name:         command.Name,
		Suggestions:  []string{},
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	for _, sub := range command.Commands() {
		if strings.HasPrefix(sub.Name, lastWord) {
			group.Suggestions = append(group.Suggestions, sub.Name)
			group.Descriptions[sub.Name] = readline.DIM + sub.ShortDescription + readline.RESET
		}
	}

	completions = append(completions, group)

	return
}

// handleSubCommand - Handles completion for subcommand options and arguments, + any option value related completion
// Many categories, from many sources: this function calls the same functions as the ones previously called for completing its parent command.
func handleSubCommand(line []rune, pos int, command *flags.Command) (lastWord string, completions []*readline.CompletionGroup) {

	args, last, lastWord := formatInput(line)

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
		return completeCommandOptions(args, lastWord, command)
	}

	// If command has non-filled arguments, propose them first
	if arg, yes := commandArgumentRequired(lastWord, args, command); yes {
		return completeCommandArguments(command, arg, lastWord)
	}

	return
}

// completeCommandOptions - Yields completion for options of a command, with various decorators
// Many categories, from one source (a command)
func completeCommandOptions(args []string, lastWord string, cmd *flags.Command) (prefix string, completions []*readline.CompletionGroup) {

	prefix = lastWord // We only return the PREFIX for readline to correctly show suggestions.

	// Get all (root) option groups.
	groups := cmd.Groups()

	// Append command options not gathered in groups
	groups = append(groups, cmd.Group)

	// For each group, build completions
	for _, grp := range groups {

		_, comp := completeOptionGroup(lastWord, grp, "")

		// No need to add empty groups, will screw the completion system.
		if len(comp.Suggestions) > 0 {
			completions = append(completions, comp)
		}
	}

	// Do the same for global options, which are not part of any group "per-se"
	_, gcomp := completeOptionGroup(lastWord, cmd.Group, "global options")
	if len(gcomp.Suggestions) > 0 {
		completions = append(completions, gcomp)
	}

	return
}

// completeOptionGroup - make completions for a single group of options. Title is optional, not used if empty.
func completeOptionGroup(lastWord string, grp *flags.Group, title string) (prefix string, compGrp *readline.CompletionGroup) {

	compGrp = &readline.CompletionGroup{
		Name:           grp.ShortDescription,
		Descriptions:   map[string]string{},
		DisplayType:    readline.TabDisplayList,
		Aliases: map[string]string{},
	}

	// An optional title for this comp group.
	// Used by global flag options, added to all commands.
	if title != "" {
		compGrp.Name = title
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
			compGrp.Suggestions = append(compGrp.Suggestions, optName)

			// Add short if there is, and that the prefix is only one dash
			if strings.HasPrefix("-", lastWord) {
				if opt.ShortName != 0 {
					compGrp.Aliases[optName] = "-" + string(opt.ShortName)
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

			desc := fmt.Sprintf(" -- %s%s%s", opt.Description, def, readline.RESET)
			compGrp.Descriptions[optName] = desc
		}
	}
	return
}

// RecursiveGroupCompletion - Handles recursive completion for nested option groups
// Many categories, one source (a command's root option group). Called by the function just above.
func RecursiveGroupCompletion(args []string, last []rune, group *flags.Group) (lastWord string, completions []*readline.CompletionGroup) {
	return
}
