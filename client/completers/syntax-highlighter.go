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
)

// SyntaxHighlighter - Entrypoint to all input syntax highlighting in the Wiregost console
func SyntaxHighlighter(input []rune) (line string) {

	// Format and sanitize input
	args, last, lastWord := FormatInput(input)

	// Remain is all arguments that have not been highlighted, we need it for completing long commands
	var remain = args

	// Detect base command automatically
	var command = detectedCommand(args, "contexthere")

	// Return input as is
	if noCommandOrEmpty(remain, last, command) {
		return string(input)
	}

	// Base command
	if commandFound(command) {
		line, remain = highlightCommand(remain, command)

		// SubCommand
		if sub, ok := subCommandFound(lastWord, args, command); ok {
			line, remain = highlightSubCommand(line, remain, sub)
		}

	}

	line = processRemain(line, remain)
	// line = processEnvVars(line, remain)

	return
}

func highlightCommand(args []string, command *flags.Command) (line string, remain []string) {
	line = tui.BOLD + command.Name + tui.RESET + " "
	remain = args[1:]
	return
}

func highlightSubCommand(input string, args []string, command *flags.Command) (line string, remain []string) {
	line = input
	line += tui.BOLD + command.Name + tui.RESET + " "
	remain = args[1:]
	return
}

func processRemain(input string, remain []string) (line string) {

	// Check the last is not the last space in input
	if len(remain) == 1 && remain[0] == " " {
		return input
	}

	line = input + strings.Join(remain, " ")
	// line = processEnvVars(input, remain)
	return
}

// processEnvVars - Highlights environment variables. NOTE: Rewrite with logic from console/env.go
func processEnvVars(input string, remain []string) (line string) {

	var processed []string

	inputSlice := strings.Split(input, " ")

	// Check already processed input
	for _, arg := range inputSlice {
		if arg == "" || arg == " " {
			continue
		}
		if strings.HasPrefix(arg, "$") { // It is an env var.
			if args := strings.Split(arg, "/"); len(args) > 1 {
				for _, a := range args {
					fmt.Println(a)
					if strings.HasPrefix(a, "$") && a != " " { // It is an env var.
						processed = append(processed, "\033[38;5;108m"+tui.DIM+a+tui.RESET)
						continue
					}
				}
			}
			processed = append(processed, "\033[38;5;108m"+tui.DIM+arg+tui.RESET)
			continue
		}
		processed = append(processed, arg)
	}

	// Check remaining args (non-processed)
	for _, arg := range remain {
		if arg == "" {
			continue
		}
		if strings.HasPrefix(arg, "$") && arg != "$" { // It is an env var.
			var full string
			args := strings.Split(arg, "/")
			if len(args) == 1 {
				if strings.HasPrefix(args[0], "$") && args[0] != "" && args[0] != "$" { // It is an env var.
					full += "\033[38;5;108m" + tui.DIM + args[0] + tui.RESET
					continue
				}
			}
			if len(args) > 1 {
				var counter int
				for _, arg := range args {
					// If var is an env var
					if strings.HasPrefix(arg, "$") && arg != "" && arg != "$" {
						if counter < len(args)-1 {
							full += "\033[38;5;108m" + tui.DIM + args[0] + tui.RESET + "/"
							counter++
							continue
						}
						if counter == len(args)-1 {
							full += "\033[38;5;108m" + tui.DIM + args[0] + tui.RESET
							counter++
							continue
						}
					}

					// Else, if we are not at the end of array
					if counter < len(args)-1 && arg != "" {
						full += arg + "/"
						counter++
					}
					if counter == len(args)-1 {
						full += arg
						counter++
					}
				}
			}
			// Else add first var
			processed = append(processed, full)
		}
	}

	line = strings.Join(processed, " ")

	// Very important, keeps the line clear when erasing
	// line += " "

	return
}
