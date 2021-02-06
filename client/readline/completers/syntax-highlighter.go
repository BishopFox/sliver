package completers

import (
	"fmt"
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// SyntaxHighlighter - Entrypoint to all input syntax highlighting in the Wiregost console
func (c *CommandCompleter) SyntaxHighlighter(input []rune) (line string) {

	// Format and sanitize input
	args, last, lastWord := formatInput(input)

	// Remain is all arguments that have not been highlighted, we need it for completing long commands
	var remain = args

	// Detect base command automatically
	var command = c.detectedCommand(args)

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

	return
}

func highlightCommand(args []string, command *flags.Command) (line string, remain []string) {
	line = readline.BOLD + command.Name + readline.RESET + " "
	remain = args[1:]
	return
}

func highlightSubCommand(input string, args []string, command *flags.Command) (line string, remain []string) {
	line = input
	line += readline.BOLD + command.Name + readline.RESET + " "
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
						processed = append(processed, "\033[38;5;108m"+readline.DIM+a+readline.RESET)
						continue
					}
				}
			}
			processed = append(processed, "\033[38;5;108m"+readline.DIM+arg+readline.RESET)
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
					full += "\033[38;5;108m" + readline.DIM + args[0] + readline.RESET
					continue
				}
			}
			if len(args) > 1 {
				var counter int
				for _, arg := range args {
					// If var is an env var
					if strings.HasPrefix(arg, "$") && arg != "" && arg != "$" {
						if counter < len(args)-1 {
							full += "\033[38;5;108m" + readline.DIM + args[0] + readline.RESET + "/"
							counter++
							continue
						}
						if counter == len(args)-1 {
							full += "\033[38;5;108m" + readline.DIM + args[0] + readline.RESET
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
