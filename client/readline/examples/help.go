package main

import (
	"fmt"
	"strconv"

	"github.com/evilsocket/islazy/tui"
	"github.com/jessevdk/go-flags"
)

// PrintMenuHelp - Prints all commands (per category)
// and a brief description when help is asked from the menu.
func PrintMenuHelp(parser *flags.Parser) {

	// First print menu title and summary
	switch parser.Name {
	case "server":
		fmt.Println(tui.Bold(tui.Blue(" Main Menu Commands\n")))
	case "sliver":
		fmt.Println(tui.Bold(tui.Blue(" Sliver Menu Commands \n")))
	}

	// Get all command and push themm in their categories
	var cmdCats = map[string][]*flags.Command{}
	var cats []string
	for _, cmd := range parser.Commands() {
		if len(cmd.Aliases) > 0 {
			if !stringInSlice(cmd.Aliases[0], &cats) {
				cats = append(cats, cmd.Aliases[0])
			}
			cmdCats[cmd.Aliases[0]] = append(cmdCats[cmd.Aliases[0]], cmd)
		}
	}

	for _, cat := range cats {

		fmt.Println(tui.Yellow(" " + cat)) // Title category

		maxLen := 0
		for _, sub := range cmdCats[cat] {
			cmdLen := len(sub.Name)
			if cmdLen > maxLen {
				maxLen = cmdLen
			}
		}

		for _, sub := range cmdCats[cat] {
			pad := fmt.Sprintf("%-"+strconv.Itoa(maxLen)+"s", sub.Name)
			fmt.Printf("    "+pad+"  %s\n", tui.Dim(sub.ShortDescription))
		}

		// Space before next category
		fmt.Println()
	}
}

func stringInSlice(a string, list *[]string) bool {
	for _, b := range *list {
		if b == a {
			return true
		}
	}
	return false
}

// PrintCommandHelp - This function is called by all command structs, either because
// there are no optional arguments, or because flags are passed.
func PrintCommandHelp(cmd *flags.Command) {

	// We first print a short description
	var subs string
	if len(cmd.Commands()) > 0 {
		subs = " ["
		for i, sub := range cmd.Commands() {
			subs += " " + tui.Bold(sub.Name)
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
	fmt.Println(tui.Yellow("Usage") + ": " + tui.Bold(cmd.Name) + options + subs + args)
	fmt.Println(tui.Yellow("Description") + ": " + cmd.ShortDescription)

	// Sub Commands
	if len(cmd.Commands()) > 0 {
		fmt.Println()
		fmt.Println(tui.Bold(tui.Blue("Sub Commands")))
	}
	maxLen := 0
	for _, sub := range cmd.Commands() {
		cmdLen := len(sub.Name)
		if cmdLen > maxLen {
			maxLen = cmdLen
		}
	}
	for _, sub := range cmd.Commands() {
		pad := fmt.Sprintf(tui.Bold("%-"+strconv.Itoa(maxLen)+"s"), sub.Name)
		fmt.Printf(" "+pad+" : %s\n", sub.ShortDescription)
	}

	// Grouped flag options
	for _, grp := range cmd.Groups() {
		printOptionGroup(grp)
	}

	// Then additional descriptions
	if additional := cmd.LongDescription; additional != "" {
		fmt.Println("\n" + cmd.LongDescription)
	}
	return
}

func printOptionGroup(grp *flags.Group) {
	fmt.Println("\n    " + tui.Bold(tui.Green(grp.ShortDescription)))

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
			defaults = tui.DIM + " (default: "
			for i, def := range opt.Default {
				defaults += def
				if i < (len(opt.Default) - 1) {
					defaults += " ,"
				}
			}
			defaults += ")" + tui.RESET
		}
		fmt.Printf("     %s  %s  %s %s\n", nameDesc, tui.Dim(optType), opt.Description, defaults)
	}
}

// findHelpCommand - A -h, --help flag was invoked in the output.
// Find the root or any subcommand.
func (c *console) findHelpCommand(args []string, parser *flags.Parser) *flags.Command {

	var root *flags.Command
	for _, cmd := range parser.Commands() {
		if cmd.Name == args[0] {
			root = cmd
		}
	}
	if root == nil {
		return nil
	}
	if len(args) == 1 || len(root.Commands()) == 0 {
		return root
	}

	var sub *flags.Command
	if len(args) > 1 {
		for _, s := range root.Commands() {
			if s.Name == args[1] {
				sub = s
			}
		}
	}
	if sub == nil {
		return root
	}
	if len(args) == 2 || len(sub.Commands()) == 0 {
		return sub
	}

	return nil
}
