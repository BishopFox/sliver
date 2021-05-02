package gonsole

import (
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"
)

// AddHelpCommand - The console will automatically add a command named "help", which accepts any
// (optional) command and/or any of its subcommands, and prints the corresponding help. If no
// argument is passed, prints the list of available of commands for the current menu.
// The name of the group is left to the user's discretion, for putting the command in a given group/topic.
// Command names and their subcommands will be automatically completed.
func (c *Console) AddHelpCommand(group string) {
	for _, cc := range c.menus {
		help := cc.AddCommand("help",
			"print menu, command or subcommand help for the current menu (menu)",
			"",
			group,
			[]string{""},
			func() interface{} { return &commandHelp{console: c} })
		help.AddArgumentCompletion("Command", c.Completer.menuCommands)
	}
}

// commandHelp - Print help for the current menu (lists all commands)
type commandHelp struct {
	Positional struct {
		Command []string `description:"(optional) command / subcommand (at any level) to print help for"`
	} `positional-args:"true"`

	// Needed to access commands
	console *Console
}

// Execute - Print help for the current menu (lists all commands)
func (h *commandHelp) Execute(args []string) (err error) {

	// If no component argument is asked for
	if len(h.Positional.Command) == 0 {
		h.console.printMenuHelp(h.console.CurrentMenu().Name)
		return
	}

	parser := h.console.CommandParser()
	command := h.console.findHelpCommand(h.Positional.Command, parser)

	if command == nil {
		fmt.Printf(errorStr+"Invalid command: %s%s%s\n",
			readline.BOLD, h.Positional.Command, readline.RESET)
		return
	}
	h.console.printCommandHelp(command)

	return
}

func getChildCommand(args []string, root *flags.Command) (child *flags.Command) {
	child = root
	var temp = root
	for _, arg := range args {
		if cmd := temp.Find(arg); cmd != nil {
			child = cmd
			temp = cmd
		}
	}
	return
}

func (c *CommandCompleter) menuCommands() (completions []*readline.CompletionGroup) {
	args := c.args[1:]

	var candidateCommands = []*flags.Command{}
	if len(args) < 2 {
		candidateCommands = c.console.CommandParser().Commands()
	} else {
		cmd := c.console.CommandParser().Find(args[0])
		if cmd != nil {
			candidateCommands = getChildCommand(args[1:], cmd).Commands()
		}
	}

	for _, cmd := range candidateCommands {
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
				Descriptions: map[string]string{
					cmd.Name: readline.Dim(cmd.ShortDescription),
				},
				DisplayType: readline.TabDisplayList,
			}
			completions = append(completions, grp)
		}
	}
	return
}

func (c *CommandCompleter) subCommands() (completions []*readline.CompletionGroup) {

	// First argument is the 'help' command, second is 'command'
	return
}
