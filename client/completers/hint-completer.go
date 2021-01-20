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
	"strings"

	"github.com/bishopfox/sliver/client/commands"
	"github.com/bishopfox/sliver/client/util"
	"github.com/evilsocket/islazy/tui"
	"github.com/jessevdk/go-flags"
)

// HintCompleter - Entrypoint to all hints in the Wiregost console
func HintCompleter(line []rune, pos int) (hint []rune) {

	// Format and sanitize input
	args, last, lastWord := FormatInput(line)

	// Detect base command automatically
	var command = detectedCommand(args)

	// Menu hints (command line is empty, or nothing recognized)
	if noCommandOrEmpty(args, last, command) {
		hint = MenuHint(args, last)
	}

	// Check environment variables
	if envVarAsked(args, lastWord) {
		return envVarHint(args, last)
	}

	// Command Hint
	if commandFound(command) {

		// Command hint by default (no space between cursor and last command character)
		hint = CommandHint(command)

		// Check environment variables
		if envVarAsked(args, lastWord) {
			return envVarHint(args, last)
		}

		if len(command.Groups()) != 0 {
			if opt, yes := optionArgRequired(args, last, command.Groups()[0]); yes {
				hint = OptionArgumentHint(args, last, opt)
			}
		}

		// If command has args, hint for args
		if arg, yes := commandArgumentRequired(lastWord, args, command); yes { // add *commands.Context.Menu in the string here
			hint = []rune(CommandArgumentHints(args, last, command, arg))
		}

		// Brief subcommand hint
		if lastIsSubCommand(lastWord, command) {
			hint = []rune(commandHint + command.Find(string(last)).ShortDescription)
		}

		// Handle subcommand if found
		if sub, ok := subCommandFound(lastWord, args, command); ok {
			return HandleSubcommandHints(args, last, sub)
		}

	}

	// Handle system binaries, shell commands, etc...
	if commandFoundInPath(args[0]) {
		// hint = []rune(exeHint + util.ParseSummary(util.GetManPages(args[0])))
	}

	return
}

// CommandHint - Yields the hint of a Wiregost command
func CommandHint(command *flags.Command) (hint []rune) {
	return []rune(commandHint + command.ShortDescription)
}

// HandleSubcommandHints - Handles hints for a subcommand and its arguments, options, etc.
func HandleSubcommandHints(args []string, last []rune, command *flags.Command) (hint []rune) {

	// If command has args, hint for args
	if arg, yes := commandArgumentRequired(string(last), args, command); yes {
		hint = []rune(CommandArgumentHints(args, last, command, arg))
		return
	}

	// Environment variables
	if envVarAsked(args, string(last)) {
		hint = envVarHint(args, last)
	}

	// If the last word in input is an option --name, yield argument hint if needed
	if len(command.Groups()) != 0 {
		if opt, yes := optionArgRequired(args, last, command.Groups()[0]); yes {
			hint = OptionArgumentHint(args, last, opt)
		}
	}

	// If user asks for completions with "-" or "--". (Note: This takes precedence on any argument hints, as it is evaluated after them)
	if commandOptionsAsked(args, string(last), command) {
		return OptionHints(args, last, command)
	}

	return
}

// CommandArgumentHints - Yields hints for arguments to commands if they have some
func CommandArgumentHints(args []string, last []rune, command *flags.Command, arg string) (hint []rune) {

	found := commands.ArgumentByName(command, arg)
	// Base Hint is just a description of the command argument
	hint = []rune(argHint + found.Description)

	return
}

// ModuleOptionHints - If the option being set has a description, show it
func ModuleOptionHints(opt string) (hint []rune) {
	return
}

// OptionHints - Yields hints for proposed options lists/groups
func OptionHints(args []string, last []rune, command *flags.Command) (hint []rune) {
	return
}

// OptionArgumentHint - Yields hints for arguments to an option (generally the last word in input)
func OptionArgumentHint(args []string, last []rune, opt *flags.Option) (hint []rune) {
	return []rune(valueHint + opt.Description)
	return
}

// MenuHint - Returns the Hint for a given menu context
func MenuHint(args []string, current []rune) (hint []rune) {
	return
}

// SpecialCommandHint - Shows hints for Wiregost special commands
func SpecialCommandHint(args []string, current []rune) (hint []rune) {
	return current
}

// envVarHint - Yields hints for environment variables
func envVarHint(args []string, last []rune) (hint []rune) {
	// Trim last in case its a path with multiple vars
	allVars := strings.Split(string(last), "/")
	lastVar := allVars[len(allVars)-1]

	// Base hint
	hint = []rune(envHint + lastVar)

	envVar := strings.TrimPrefix(lastVar, "$")

	if v, ok := util.ClientEnv[envVar]; ok {
		if v != "" {
			hintStr := string(hint) + " => " + util.ClientEnv[envVar]
			hint = []rune(hintStr)
		}
	}
	return
}

var (
	// Hint signs
	menuHint    = tui.RESET + tui.DIM + tui.BOLD + " Menu  " + tui.RESET                                    // Dim
	envHint     = tui.RESET + tui.GREEN + tui.BOLD + " Env  " + tui.RESET + tui.DIM + tui.GREEN             // Green
	commandHint = tui.RESET + tui.DIM + tui.BOLD + " Command  " + tui.RESET + tui.DIM + "\033[38;5;244m"    // Cream
	exeHint     = tui.RESET + tui.DIM + tui.BOLD + " Shell " + tui.RESET + tui.DIM                          // Dim
	optionHint  = "\033[38;5;222m" + tui.BOLD + " Options  " + tui.RESET + tui.DIM + "\033[38;5;222m"       // Cream-Yellow
	valueHint   = "\033[38;5;217m" + tui.BOLD + " Value  " + tui.RESET + tui.DIM + "\033[38;5;244m"         // Pink-Cream
	argHint     = tui.DIM + "\033[38;5;217m" + tui.BOLD + " Arg  " + tui.RESET + tui.DIM + "\033[38;5;244m" // Pink-Cream
)
