package help

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

	---
	This file contains all of the long-form help templates, all commands should have a long form help,
	even if the command is pretty simple. Try to include example commands in your template.
*/

import (
	"fmt"
	"strconv"

	"github.com/jessevdk/go-flags"
	"github.com/maxlandon/readline"

	cctx "github.com/bishopfox/sliver/client/context"
)

// PrintMenuHelp - Prints all commands (per category)
// and a brief description when help is asked from the menu.
func PrintMenuHelp(context string) {

	// Commands and an ordered list of the groups
	var groups []string
	var cmds map[string][]*flags.Command

	// The user can specify the menu help he wants. If none is
	// given or recognized, we default on the current console context.
	switch context {
	case cctx.Server:
		groups, cmds = cctx.Commands.GetServerGroups()
		fmt.Println(readline.Bold(readline.Blue(" Main Menu Commands\n")))
	case cctx.Sliver:
		groups, cmds = cctx.Commands.GetSliverGroups()
		fmt.Println(readline.Bold(readline.Blue(" Sliver Menu Commands \n")))

	default:
		// As default use the current context
		switch cctx.Context.Menu {
		case cctx.Server:
			groups, cmds = cctx.Commands.GetServerGroups()
			fmt.Println(readline.Bold(readline.Blue(" Main Menu Commands\n")))
		case cctx.Sliver:
			groups, cmds = cctx.Commands.GetSliverGroups()
			fmt.Println(readline.Bold(readline.Blue(" Sliver Menu Commands \n")))
		}
	}

	// Print help for each command group
	for _, group := range groups {
		fmt.Println(readline.Yellow(" " + group)) // Title category

		maxLen := 0
		for _, cmd := range cmds[group] {
			cmdLen := len(cmd.Name)
			if cmdLen > maxLen {
				maxLen = cmdLen
			}
		}

		for _, cmd := range cmds[group] {
			pad := fmt.Sprintf("%-"+strconv.Itoa(maxLen)+"s", cmd.Name)
			fmt.Printf("    "+pad+"  %s\n", readline.Dim(cmd.ShortDescription))
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
		printOptionGroup(grp)
	}

	// Then additional descriptions
	if additional := GetHelpFor(cmd.Name); additional != "" {
		fmt.Println("\n" + GetHelpFor(cmd.Name))
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
