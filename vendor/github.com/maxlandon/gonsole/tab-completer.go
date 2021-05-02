package gonsole

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// CommandCompleter - A completer using a github.com/jessevdk/go-flags Command Parser, in order
// to build completions for commands, arguments, options and their arguments as well.
// This completer needs to be instantiated with its constructor, in order to ensure the parser is not nil.
type CommandCompleter struct {
	console  *Console
	args     []string
	last     []rune
	lastWord string
	mutex    *sync.Mutex
}

// newCommandCompleter - Instantiate a new tab completer using a github.com/jessevdk/go-flags Command Parser.
func newCommandCompleter(c *Console) (completer *CommandCompleter) {
	return &CommandCompleter{console: c, mutex: &sync.Mutex{}}
}

// tabCompleter - A default tab completer working with a github.com/jessevdk/go-flags parser.
func (c *CommandCompleter) tabCompleter(line []rune, pos int, dtc readline.DelayedTabContext) (lastWord string, comps []*readline.CompletionGroup) {

	// Format and sanitize input
	// @args     => All items of the input line
	// @last     => The last word detected in input line as []rune
	// @lastWord => The last word detected in input as string
	c.mutex.Lock()
	c.args, c.last, c.lastWord = formatInput(line)
	c.mutex.Unlock()

	// The prefix is always what has been given by the shell, by default.
	lastWord = c.lastWord

	// Detect base command automatically
	var command = c.detectedCommand(c.args)

	// Propose commands
	if noCommandOrEmpty(c.args, c.last, command) {
		return c.completeMenuCommands(lastWord, pos)
	}

	// Check environment variables
	if yes, exp, envComps := c.envVarAsked(c.args, lastWord); yes {
		lastWord, comps = c.completeExpansionVariables(lastWord, exp, envComps())
	}

	// Base command has been identified
	if commandFound(command) {

		// Get the corresponding *Command from the console, containing user completion functions
		gCommand := c.console.FindCommand(command.Name)
		if gCommand == nil {
			return
		}

		// Check environment variables again
		if yes, exp, envComps := c.envVarAsked(c.args, lastWord); yes {
			return c.completeExpansionVariables(lastWord, exp, envComps())
		}

		// If options are asked for root command, return commpletions.
		if len(command.Groups()) > 0 {
			for _, grp := range command.Groups() {
				if opt, yes := optionArgRequired(c.args, c.last, grp); yes {
					return c.completeOptionArguments(gCommand, command, opt, lastWord)
				}
			}
		}

		// Then propose subcommands. We don't return from here, otherwise it always skips the next steps.
		if hasSubCommands(command, c.args) {
			comps = completeSubCommands(c.args, lastWord, command, gCommand)
		}

		// Handle subcommand if found (maybe we should rewrite this function and use it also for base command)
		if sub, ok := subCommandFound(lastWord, c.args, command); ok {
			subg := gCommand.FindCommand(sub.Name)
			return c.handleSubCommand(c.args[1:], command, sub, subg)
		}

		// If user asks for completions with "-" / "--", show command options.
		// We ask this here, after having ensured there is no subcommand invoked.
		// This prevails over command arguments, even if they are required.
		if commandOptionsAsked(c.args, lastWord, command) {
			globalOpts := c.console.CommandParser().Groups()
			return completeCommandOptions(c.args, lastWord, command, globalOpts)
		}

		// Propose argument completion before anything, and if needed
		if arg, yes := commandArgumentRequired(lastWord, c.args, command); yes {
			return c.completeCommandArguments(gCommand, command, arg, lastWord)
		}

	}

	return
}

// [ Main Completion Functions ] -----------------------------------------------------------------------------------------------------------------

// completeMenuCommands - Selects all commands available in a given menu and returns them as suggestions
// Many categories, all from command parsers.
func (c *CommandCompleter) completeMenuCommands(lastWord string, pos int) (prefix string, completions []*readline.CompletionGroup) {

	prefix = lastWord // We only return the PREFIX for readline to correctly show suggestions.

	// Check their namespace (which should be their "group" (like utils, core, Jobs, etc))
	for _, cmd := range c.console.CommandParser().Commands() {
		// If command matches readline input, and the command is available
		if strings.HasPrefix(cmd.Name, lastWord) && !cmd.Hidden {
			// Check command group: add to existing group if found
			var found bool
			for _, grp := range completions {
				if grp.Name == c.console.GetCommandGroup(cmd) {
					found = true
					grp.Suggestions = append(grp.Suggestions, cmd.Name)
					grp.Descriptions[cmd.Name] = readline.Dim(cmd.ShortDescription)
				}
			}
			// Add a new group if not found
			if !found {
				grp := &readline.CompletionGroup{
					Name:        c.console.GetCommandGroup(cmd),
					Suggestions: []string{cmd.Name},
					DisplayType: readline.TabDisplayList,
					Descriptions: map[string]string{
						cmd.Name: readline.Dim(cmd.ShortDescription),
					},
				}
				completions = append(completions, grp)
			}
		}
	}

	return
}

// completeSubCommands - Takes subcommands and gives them as suggestions
// One category, from one source (a parent command).
func completeSubCommands(args []string, lastWord string, command *flags.Command, gCommand *Command) (completions []*readline.CompletionGroup) {

	// Check their namespace (which should be their "group" (like utils, core, Jobs, etc))
	for _, cmd := range command.Commands() {
		// If command matches readline input, and the command is available
		if strings.HasPrefix(cmd.Name, lastWord) && !cmd.Hidden {
			// Check command group: add to existing group if found
			var found bool
			for _, grp := range completions {
				if grp.Name == gCommand.getCommandGroup(cmd) {
					found = true
					grp.Suggestions = append(grp.Suggestions, cmd.Name)
					grp.Descriptions[cmd.Name] = readline.Dim(cmd.ShortDescription)
					break
				}
			}
			// Add a new group if not found
			if !found && gCommand.getCommandGroup(cmd) != "" {
				grp := &readline.CompletionGroup{
					Name:        gCommand.getCommandGroup(cmd),
					Suggestions: []string{cmd.Name},
					DisplayType: readline.TabDisplayList,
					Descriptions: map[string]string{
						cmd.Name: readline.Dim(cmd.ShortDescription),
					},
				}
				completions = append(completions, grp)
			} else if !found && gCommand.getCommandGroup(cmd) == "" {
				var cmdGrpfound bool
				for _, grp := range completions {
					if grp.Name == command.Name {
						cmdGrpfound = true
						grp.Suggestions = append(grp.Suggestions, cmd.Name)
						grp.Descriptions[cmd.Name] = readline.Dim(cmd.ShortDescription)
						break
					}
				}
				if !cmdGrpfound {
					grp := &readline.CompletionGroup{
						Name:        command.Name,
						Suggestions: []string{cmd.Name},
						DisplayType: readline.TabDisplayList,
						Descriptions: map[string]string{
							cmd.Name: readline.Dim(cmd.ShortDescription),
						},
					}
					completions = append(completions, grp)
				}
			}

		}
	}

	return
}

// handleSubCommand - Handles completion for subcommand options and arguments, + any option value related completion
// Many categories, from many sources: this function calls the same functions as the ones previously called for completing its parent command.
func (c *CommandCompleter) handleSubCommand(args []string, parent, command *flags.Command, gCommand *Command) (lastWord string, comps []*readline.CompletionGroup) {

	lastWord = c.lastWord

	// Check environment variables
	if yes, exp, envComps := c.envVarAsked(args, lastWord); yes {
		c.completeExpansionVariables(lastWord, exp, envComps())
	}

	// Check argument options
	if len(command.Groups()) > 0 {
		for _, grp := range command.Groups() {
			if opt, yes := optionArgRequired(args, c.last, grp); yes {
				return c.completeOptionArguments(gCommand, command, opt, lastWord)
			}
		}
	}

	// Then propose subcommands. We don't return from here, otherwise it always skips the next steps.
	if hasSubCommands(command, args) {
		comps = completeSubCommands(args, lastWord, command, gCommand)
	}

	// Handle subcommand if found (maybe we should rewrite this function and use it also for base command)
	if sub, ok := subCommandFound(lastWord, args, command); ok {
		subgCommand := gCommand.FindCommand(sub.Name)
		if gCommand == nil {
			return
		}
		return c.handleSubCommand(args[1:], command, sub, subgCommand)
	}

	// If user asks for completions with "-" or "--". This must take precedence on arguments.
	if commandOptionsAsked(args, lastWord, command) {
		globalOpts := c.console.CommandParser().Groups()
		globalOpts = append(globalOpts, parent.Groups()...)
		return completeCommandOptions(args, lastWord, command, globalOpts)
	}

	// If command has non-filled arguments, propose them first
	if arg, yes := commandArgumentRequired(lastWord, args, command); yes {
		return c.completeCommandArguments(gCommand, command, arg, lastWord)
	}

	return
}

// completeCommandOptions - Yields completion for options of a command, with various decorators
// Many categories, from one source (a command)
func completeCommandOptions(args []string, lastWord string, cmd *flags.Command, globalOpts []*flags.Group) (prefix string, comps []*readline.CompletionGroup) {

	prefix = lastWord // We only return the PREFIX for readline to correctly show suggestions.

	// Get all (root) option groups.
	groups := cmd.Groups()

	// Append command options not gathered in groups
	groups = append(groups, cmd.Group)

	// For each group, build completions
	for _, grp := range groups {

		_, comp := completeOptionGroup(args, lastWord, grp, "")

		// No need to add empty groups, will screw the completion system.
		if len(comp.Suggestions) > 0 {
			comps = append(comps, comp)
		}
	}

	// Append global options (persistent options from root commands/parser)
	// For each group, build completions
	for _, grp := range globalOpts {

		_, comp := completeOptionGroup(args, lastWord, grp, "")

		// No need to add empty groups, will screw the completion system.
		if len(comp.Suggestions) > 0 {
			comps = append(comps, comp)
		}
	}

	// Do the same for global options, which are not part of any group "per-se"
	_, gcomp := completeOptionGroup(args, lastWord, cmd.Group, "global options")
	if len(gcomp.Suggestions) > 0 {
		comps = append(comps, gcomp)
	}

	return
}

// completeOptionGroup - make completions for a single group of options. Title is optional, not used if empty.
func completeOptionGroup(args []string, lastWord string, grp *flags.Group, title string) (prefix string, compGrp *readline.CompletionGroup) {

	compGrp = &readline.CompletionGroup{
		Name:         strings.ToLower(grp.ShortDescription),
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
		Aliases:      map[string]string{},
	}

	// An optional title for this comp group.
	// Used by global flag options, added to all commands.
	if title != "" {
		compGrp.Name = strings.ToLower(title)
	}

	// Add each option to completion group
	for _, opt := range grp.Options() {

		set, _ := optionIsAlreadySet(args, lastWord, opt)

		// Check that we must indeed show the option completion, given that:
		// - it might be a non-array type and already be set in the input
		if optionNotRepeatable(opt) && set {
			continue
		}
		// - have a min/max number of required occurences, because the type is an array/map, or unknown.
		// NOTE: to implement: hide when maximum required is hit.

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

			var desc string
			if opt.Required {
				desc = fmt.Sprintf(" %s%s%s%s",
					readline.DIM, opt.Description, def, readline.RESET)
			} else {
				desc = fmt.Sprintf(" %s%s%s%s",
					readline.DIM, opt.Description, def, readline.RESET)
			}
			compGrp.Descriptions[optName] = desc
		}
	}
	return
}

// recursiveGroupCompletion - Handles recursive completion for nested option groups
// Many categories, one source (a command's root option group). Called by the function just above.
func recursiveGroupCompletion(args []string, last []rune, group *flags.Group) (lastWord string, comps []*readline.CompletionGroup) {
	return
}
