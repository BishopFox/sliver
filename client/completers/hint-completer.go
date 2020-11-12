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
	"github.com/evilsocket/islazy/tui"
	"github.com/jessevdk/go-flags"
)

// HintCompleter - Entrypoint to all hints in the Wiregost console
func HintCompleter(line []rune, pos int) (hint []rune) {

	// Format and sanitize input
	args, last, lastWord := FormatInput(line)

	// Detect base command automatically
	var command = detectedCommand(args, "") // add *commands.Context.Menu in the string here

	// Menu hints (command line is empty, or nothing recognized)
	if noCommandOrEmpty(args, last, command) {
	}

	// Check environment variables
	if envVarAsked(args, lastWord) {
	}

	// Command Hint
	if commandFound(command) {

		// Command hint by default (no space between cursor and last command character)
		hint = CommandHint(command)

		// Check environment variables
		if envVarAsked(args, lastWord) {
		}

		// If command has args, hint for args
		if _, yes := argumentRequired(lastWord, args, "", command, false); yes { // add *commands.Context.Menu in the string here
		}

		// Brief subcommand hint
		if lastIsSubCommand(lastWord, command) {
		}

		// Handle subcommand if found
		if _, ok := subCommandFound(lastWord, args, command); ok {
		}

	}

	// Handle special Wiregost commands
	if isSpecialCommand(args, command) {
	}

	// Handle system binaries, shell commands, etc...
	if commandFoundInPath(args[0]) {
	}

	return
}

// CommandHint - Yields the hint of a Wiregost command
func CommandHint(command *flags.Command) (hint []rune) {
	return []rune(commandHint + command.ShortDescription)
}

// HandleSubcommandHints - Handles hints for a subcommand and its arguments, options, etc.
func HandleSubcommandHints(args []string, last []rune, command *flags.Command) (hint []rune) {
	return
}

// CommandArgumentHints - Yields hints for arguments to commands if they have some
func CommandArgumentHints(args []string, last []rune, command *flags.Command, arg string) (hint []rune) {
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
	return
}

// MenuHint - Returns the Hint for a given menu context
func MenuHint(args []string, current []rune) (hint []rune) {
	return current
}

// SpecialCommandHint - Shows hints for Wiregost special commands
func SpecialCommandHint(args []string, current []rune) (hint []rune) {
	return current
}

// envVarHint - Yields hints for environment variables
func envVarHint(args []string, last []rune) (hint []rune) {
	return
}

var (
	// Hint signs
	menuHint    = tui.RESET + tui.DIM + tui.BOLD + " Menu  " + tui.RESET                              // Dim
	envHint     = tui.RESET + tui.GREEN + tui.BOLD + " Env  " + tui.RESET + tui.DIM + tui.GREEN       // Green
	commandHint = "\033[38;5;223m" + tui.BOLD + " Command  " + tui.RESET + tui.DIM + "\033[38;5;223m" // Cream
	exeHint     = tui.RESET + tui.DIM + tui.BOLD + " Shell " + tui.RESET + tui.DIM                    // Dim
	optionHint  = "\033[38;5;222m" + tui.BOLD + " Options  " + tui.RESET + tui.DIM + "\033[38;5;222m" // Cream-Yellow
	valueHint   = "\033[38;5;217m" + tui.BOLD + " Value  " + tui.RESET + tui.DIM + "\033[38;5;217m"   // Pink-Cream
	argHint     = "\033[38;5;217m" + tui.BOLD + " Arg  " + tui.RESET + tui.DIM + "\033[38;5;217m"     // Pink-Cream
)
