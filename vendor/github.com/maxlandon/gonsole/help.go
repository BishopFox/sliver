package gonsole

import (
	"fmt"
	"strconv"

	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"
)

// The user has passed a -h, --help option flag in the input line,
// so we handle the error raised by the command parser and print the corresponding help.
func (c *Console) handleHelpFlag(args []string) {
	cmd := c.findHelpCommand(args, c.current.parser)

	// If command is nil, it means the help was requested as
	// the menu help: print all commands for the menu.
	if cmd == nil {
		c.printMenuHelp(c.current.Name)
		return
	}

	// Else print the help for a specific command
	c.printCommandHelp(cmd)
}

// printMenuHelp - Prints all commands (per category)
// and a brief description when help is asked from the menu.
func (c *Console) printMenuHelp(menu string) {

	// The user can specify the menu help he wants. If none is
	// given or recognized, we default on the current console menu.
	cmds, groups := c.GetCommands()

	// Menu title
	fmt.Printf(readline.BOLD+readline.BLUE+" %s Menu Commands\n\n", c.current.Name)

	// Print help for each command group
	for _, group := range groups {
		if _, found := cmds[group]; !found {
			continue
		} else if len(cmds[group]) == 0 {
			continue
		}
		// Commands might have been "ungenerated" due to active command filters
		allNils := true
		for _, cmd := range cmds[group] {
			if cmd != nil {
				allNils = false
			}
		}
		if allNils {
			continue
		}

		fmt.Println(readline.Yellow(" " + group)) // Title category

		maxLen := 0
		for _, cmd := range cmds[group] {
			if cmd == nil {
				continue
			}
			cmdLen := len(cmd.Name)
			if cmdLen > maxLen {
				maxLen = cmdLen
			}
		}

		for _, cmd := range cmds[group] {
			if cmd == nil {
				continue
			}
			pad := fmt.Sprintf("%-"+strconv.Itoa(maxLen)+"s", cmd.Name)
			fmt.Printf("    "+pad+"  %s\n", readline.Dim(cmd.ShortDescription))
		}

		// Space before next category
		fmt.Println()
	}
}

// findHelpCommand - A -h, --help flag was invoked in the output.
// Find the root or any subcommand.
func (c *Console) findHelpCommand(args []string, parser *flags.Parser) *flags.Command {
	for _, cmd := range parser.Commands() {
		if cmd.Name == args[0] {
			return c.findLastCommand(args[1:], cmd)
		}
	}

	return nil
}

// findHelpCommand - A -h, --help flag was invoked in the output.
// Find the root or any subcommand.
func (c *Console) findLastCommand(args []string, command *flags.Command) (sub *flags.Command) {
	for i, arg := range args {
		for _, cmd := range command.Commands() {
			if cmd.Name == arg {
				return c.findLastCommand(args[i:], cmd)
			}
		}
	}
	return command
}

func stringInSlice(a string, list *[]string) bool {
	for _, b := range *list {
		if b == a {
			return true
		}
	}
	return false
}

// printCommandHelp - This function is called by all command structs, either because
// there are no optional arguments, or because flags are passed.
func (c *Console) printCommandHelp(cmd *flags.Command) {

	// We first print a short description
	var subs string
	if len(cmd.Commands()) > 0 {
		subs = " ["
		for i, sub := range cmd.Commands() {
			subs += " " + readline.Bold(sub.Name)
			if i < (len(cmd.Commands()) - 1) {
				subs += " |"
			}
		}
		subs += " ]"
	}
	var options string
	if len(cmd.Options()) > 0 || len(cmd.Groups()) > 0 {
		options = " --options"
	}

	// Command arguments
	var args string
	if len(cmd.Args()) > 0 {
		for _, arg := range cmd.Args() {
			if arg.Required == 1 && arg.RequiredMaximum == 1 {
				args += " " + arg.Name
			}
			if arg.Required > 0 && arg.RequiredMaximum == -1 {
				args += " " + arg.Name + "1" + " [" + arg.Name + "2]" + " [" + arg.Name + "3]"
			}
			if arg.Required == -1 {
				args += fmt.Sprintf(" [%s]", arg.Name)
			}
		}
	}
	fmt.Println(readline.Yellow("Usage") + ": " + readline.Bold(cmd.Name) + options + subs + args)
	fmt.Println(readline.Yellow("Description") + ": " + cmd.ShortDescription)

	// Sub Commands
	if len(cmd.Commands()) > 0 {
		fmt.Println()
		fmt.Println(readline.Bold(readline.Blue("Sub Commands")))
	}
	maxLen := 0
	for _, sub := range cmd.Commands() {
		cmdLen := len(sub.Name)
		if cmdLen > maxLen {
			maxLen = cmdLen
		}
	}
	for _, sub := range cmd.Commands() {
		pad := fmt.Sprintf(readline.Bold("%-"+strconv.Itoa(maxLen)+"s"), sub.Name)
		fmt.Printf(" "+pad+" : %s\n", sub.ShortDescription)
	}

	// Grouped flag options
	for _, grp := range cmd.Groups() {
		if grp.ShortDescription != "Help Options" {
			printOptionGroup(grp)
		}
	}

	// Global options (the parser has options that apply to all commands)
	// We don't show the help options (showing the -h, --help flag)
	for _, grp := range c.current.parser.Groups() {
		if grp.ShortDescription != "Help Options" {
			printOptionGroup(grp)
		}
	}

	// Then additional descriptions
	if cmd.LongDescription != "" {
		fmt.Println("\n" + cmd.LongDescription)
	}

	return
}

func printOptionGroup(grp *flags.Group) {
	fmt.Println("\n    " + readline.Bold(readline.Green(grp.ShortDescription)))

	grpOptLen := 0
	for _, opt := range grp.Options() {
		len := len("--" + opt.LongName)
		if len > grpOptLen {
			grpOptLen = len
		}
	}

	typeLen := 0
	for _, opt := range grp.Options() {
		var optName string
		if opt.Field().Type.Name() != "" {
			optName = opt.Field().Type.Name()
		} else {
			optName = fmt.Sprintf("%s", opt.Field().Type)
		}

		len := len("--" + optName)
		if len > typeLen {
			typeLen = len
		}
	}

	// Print lign for each option
	for _, opt := range grp.Options() {
		// --flag
		optForm := "--" + opt.LongName
		nameDesc := fmt.Sprintf("%-"+strconv.Itoa(grpOptLen)+"s", optForm)

		// type
		var optName string
		if opt.Field().Type.Name() != "" {
			optName = opt.Field().Type.Name()
		} else {
			optName = fmt.Sprintf("%s", opt.Field().Type)
		}
		optType := fmt.Sprintf("%-"+strconv.Itoa(typeLen)+"s", optName)

		// Description & defaults
		var defaults string
		if len(opt.Default) > 0 {
			defaults = readline.DIM + " (default: "
			for i, def := range opt.Default {
				defaults += def
				if i < (len(opt.Default) - 1) {
					defaults += " ,"
				}
			}
			defaults += ")" + readline.RESET
		}
		fmt.Printf("     %s  %s  %s %s\n", nameDesc, readline.Dim(optType), opt.Description, defaults)
	}
}
