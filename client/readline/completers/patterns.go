package completers

import (
	"os/exec"
	"reflect"
	"strings"
	"unicode"

	"github.com/jessevdk/go-flags"
)

// These functions are just shorthands for checking various conditions on the input line.
// They make the main function more readable, which might be useful, should a logic error pop somewhere.

// [ Parser Commands & Options ] --------------------------------------------------------------------------
// ArgumentByName Get the name of a detected command's argument
func argumentByName(command *flags.Command, name string) *flags.Arg {
	args := command.Args()
	for _, arg := range args {
		if arg.Name == name {
			return arg
		}
	}
	return nil
}

// optionByName - Returns an option for a command or a subcommand, identified by name
func optionByName(cmd *flags.Command, option string) *flags.Option {

	if cmd == nil {
		return nil
	}
	// Get all (root) option groups.
	groups := cmd.Groups()

	// For each group, build completions
	for _, grp := range groups {
		// Add each option to completion group
		for _, opt := range grp.Options() {
			if opt.LongName == option {
				return opt
			}
		}
	}
	return nil
}

// [ Menus ] --------------------------------------------------------------------------------------------
// Is the input line is either empty, or without any detected command ?
func noCommandOrEmpty(args []string, last []rune, command *flags.Command) bool {
	if len(args) == 0 || len(args) == 1 && command == nil {
		return true
	}
	return false
}

// [ Commands ] -------------------------------------------------------------------------------------
// detectedCommand - Returns the base command from parser if detected, depending on context
func (c *CommandCompleter) detectedCommand(args []string) (command *flags.Command) {
	command = c.parser.Find(args[0])
	return
}

// is the command a special command, usually not handled by parser ?
func isSpecialCommand(args []string, command *flags.Command) bool {

	// If command is not nil, return
	if command == nil {
		// Shell
		if args[0] == "!" {
			return true
		}
		// Exit
		if args[0] == "exit" {
			return true
		}
		return false
	}
	return false
}

// The commmand has been found
func commandFound(command *flags.Command) bool {
	if command != nil {
		return true
	}
	return false
}

// Search for input in $PATH
func commandFoundInPath(input string) bool {
	_, err := exec.LookPath(input)
	if err != nil {
		return false
	}
	return true
}

// [ SubCommands ]-------------------------------------------------------------------------------------
// Does the command have subcommands ?
func hasSubCommands(command *flags.Command, args []string) bool {
	if len(args) < 2 || command == nil {
		return false
	}

	if len(command.Commands()) != 0 {
		return true
	}

	return false
}

// Does the input has a subcommand in it ?
func subCommandFound(lastWord string, args []string, command *flags.Command) (sub *flags.Command, ok bool) {
	if len(args) <= 1 || command == nil {
		return nil, false
	}

	sub = command.Find(args[1])
	if sub != nil {
		return sub, true
	}

	return nil, false
}

// Is the last input PRECISELY a subcommand. This is used as a brief hint for the subcommand
func lastIsSubCommand(lastWord string, command *flags.Command) bool {
	if sub := command.Find(lastWord); sub != nil {
		return true
	}
	return false
}

// [ Arguments ]-------------------------------------------------------------------------------------
// Does the command have arguments ?
func hasArgs(command *flags.Command) bool {
	if len(command.Args()) != 0 {
		return true
	}
	return false
}

// commandArgumentRequired - Analyses input and sends back the next argument name to provide completion for
func commandArgumentRequired(lastWord string, args []string, command *flags.Command) (name string, yes bool) {

	// Trim command and subcommand args
	var remain []string
	if args[0] == command.Name {
		remain = args[1:]
	}
	if len(args) > 1 && args[1] == command.Name {
		remain = args[2:]
	}

	// The remain may include a "" as a last element,
	// which we don't consider as a real remain, so we move it away
	if lastWord == "" {
		if len(remain) > 1 {
			remain = remain[:]
		}
		if len(remain) == 1 { // Avoid index error
			remain = []string{}
		}
	}

	// Trim all --option flags and their arguments if they have
	remain = filterOptions(remain, command)

	// For each argument, check if needs completion. If not continue, if yes return.
	// The arguments remainder is popped according to the number of values expected.
	for i, arg := range command.Args() {

		// If it's required and has one argument, check filled.
		if arg.Required == 1 && arg.RequiredMaximum == 1 {

			// If last word is the argument, and we are
			// last arg in: line keep completing.
			if len(remain) <= 1 {
				return arg.Name, true
			}
			// If last word is the argument, and we are
			// last arg in line keep completing.
			// if len(remain) <= 1 && i == (len(command.Args())-1) {
			//         return arg.Name, true
			// }

			// If filed and we are not last arg, continue
			if len(remain) > 1 && i < (len(command.Args())-1) {
				remain = remain[1:]
				continue
			}
		}

		// If we need more than one value and we knwo the maximum,
		// either return or pop the remain.
		if arg.Required > 0 && arg.RequiredMaximum > 1 {
			// Pop the corresponding amount of arguments.
			var found int
			for i := 0; i < len(remain) && i < arg.RequiredMaximum; i++ {
				remain = remain[1:]
				found++
			}

			// If we still need values:
			if len(remain) == 0 && found <= arg.RequiredMaximum {
				if lastWord == "" { // We are done, no more completions.
					break
				} else {
					return arg.Name, true
				}
			}
			// Else go on with the next argument
			continue
		}

		// If have required arguments, with no limit of needs, return true
		if arg.Required > 0 && arg.RequiredMaximum == -1 {
			return arg.Name, true
		}

		// Else, if no requirements and the command has subcommands,
		// return so that we complete subcommands
		if arg.Required == -1 && len(command.Commands()) > 0 {
			continue
		}

		// Else, return this argument
		// NOTE: This block is after because we always use []type arguments
		// AFTER individual argument fields. Thus blocks any args that have
		// not been processed.
		if arg.Required == -1 {
			return arg.Name, true
		}
	}

	// Once we exited the loop, it means that none of the arguments require completion:
	// They are all either optional, or fullfiled according to their required numbers.
	// Thus we return none
	return "", false
}

// getRemainingArgs - Filters the input slice from commands and detected option:value pairs, and returns args
func getRemainingArgs(args []string, last []rune, command *flags.Command) (remain []string) {

	var input []string
	// Clean subcommand name
	if args[0] == command.Name && len(args) >= 2 {
		input = args[1:]
	} else if len(args) == 1 {
		input = args
	}

	// For each each argument
	for i := 0; i < len(input); i++ {
		// Check option prefix
		if strings.HasPrefix(input[i], "-") || strings.HasPrefix(input[i], "--") {
			// Clean it
			cur := strings.TrimPrefix(input[i], "--")
			cur = strings.TrimPrefix(cur, "-")

			// Check if option matches any command option
			if opt := command.FindOptionByLongName(cur); opt != nil {
				boolean := true
				if opt.Field().Type == reflect.TypeOf(boolean) {
					continue // If option is boolean, don't skip an argument
				}
				i++ // Else skip next arg in input
				continue
			}
		}

		// Safety check
		if input[i] == "" || input[i] == " " {
			continue
		}

		remain = append(remain, input[i])
	}

	return
}

// [ Options ]-------------------------------------------------------------------------------------
// commandOptionsAsked - Does the user asks for options in a root command ?
func commandOptionsAsked(args []string, lastWord string, command *flags.Command) bool {
	if len(args) >= 2 && (strings.HasPrefix(lastWord, "-") || strings.HasPrefix(lastWord, "--")) {
		return true
	}
	return false
}

// commandOptionsAsked - Does the user asks for options in a subcommand ?
func subCommandOptionsAsked(args []string, lastWord string, command *flags.Command) bool {
	if len(args) > 2 && (strings.HasPrefix(lastWord, "-") || strings.HasPrefix(lastWord, "--")) {
		return true
	}
	return false
}

// Is the last input argument is a dash ?
func isOptionDash(args []string, last []rune) bool {
	if len(args) > 2 && (strings.HasPrefix(string(last), "-") || strings.HasPrefix(string(last), "--")) {
		return true
	}
	return false
}

// optionIsAlreadySet - Detects in input if an option is already set
func optionIsAlreadySet(args []string, lastWord string, opt *flags.Option) bool {
	return false
}

// Check if option type allows for repetition
func optionNotRepeatable(opt *flags.Option) bool {
	return true
}

// [ Option Values ]-------------------------------------------------------------------------------------
// Is the last input word an option name (--option) ?
func optionArgRequired(args []string, last []rune, group *flags.Group) (opt *flags.Option, yes bool) {

	var lastItem string
	var lastOption string
	var option *flags.Option

	// If there is argument required we must have 1) command 2) --option inputs at least.
	if len(args) <= 2 {
		return nil, false
	}

	// Check for last two arguments in input
	if strings.HasPrefix(args[len(args)-2], "-") || strings.HasPrefix(args[len(args)-2], "--") {

		// Long opts
		if strings.HasPrefix(args[len(args)-2], "--") {
			lastOption = strings.TrimPrefix(args[len(args)-2], "--")
			if opt := group.FindOptionByLongName(lastOption); opt != nil {
				option = opt
			}

			// Short opts
		} else if strings.HasPrefix(args[len(args)-2], "-") {
			lastOption = strings.TrimPrefix(args[len(args)-2], "-")
			if len(lastOption) > 0 {
				if opt := group.FindOptionByShortName(rune(lastOption[0])); opt != nil {
					option = opt
				}
			}
		}

	}

	// If option is found, and we still are in writing the argument
	if (lastItem == "" && option != nil) || option != nil {
		// Check if option is a boolean, if yes return false
		boolean := true
		if option.Field().Type == reflect.TypeOf(boolean) {
			return nil, false
		}

		return option, true
	}

	// Check for previous argument
	if lastItem != "" && option == nil {
		if strings.HasPrefix(args[len(args)-2], "-") || strings.HasPrefix(args[len(args)-2], "--") {

			// Long opts
			if strings.HasPrefix(args[len(args)-2], "--") {
				lastOption = strings.TrimPrefix(args[len(args)-2], "--")
				if opt := group.FindOptionByLongName(lastOption); opt != nil {
					option = opt
					return option, true
				}

				// Short opts
			} else if strings.HasPrefix(args[len(args)-2], "-") {
				lastOption = strings.TrimPrefix(args[len(args)-2], "-")
				if opt := group.FindOptionByShortName(rune(lastOption[0])); opt != nil {
					option = opt
					return option, true
				}
			}
		}
	}

	return nil, false
}

// [ Other ]-------------------------------------------------------------------------------------
// Does the user asks for Environment variables ?
func envVarAsked(args []string, lastWord string) bool {

	// Check if the current word is an environment variable, or if the last part of it is a variable
	if len(lastWord) > 1 && strings.HasPrefix(lastWord, "$") {
		if strings.LastIndex(lastWord, "/") < strings.LastIndex(lastWord, "$") {
			return true
		}
		return false
	}

	// Check if env var is asked in a path or something
	if len(lastWord) > 1 {
		// If last is a path, it cannot be an env var anymore
		if lastWord[len(lastWord)-1] == '/' {
			return false
		}

		if lastWord[len(lastWord)-1] == '$' {
			return true
		}
	}

	// If we are at the beginning of an env var
	if len(lastWord) > 0 && lastWord[len(lastWord)-1] == '$' {
		return true
	}

	return false
}

// filterOptions - Check various elements of an option and return a list
func filterOptions(args []string, command *flags.Command) (processed []string) {

	for i := 0; i < len(args); i++ {
		arg := args[i]
		// --long-name options
		if strings.HasPrefix(arg, "--") {
			name := strings.TrimPrefix(arg, "--")
			if opt := optionByName(command, name); opt != nil {
				var boolean = true
				if opt.Field().Type == reflect.TypeOf(boolean) {
					continue
				}
				// Else skip the option argument (next item)
				i++
			}
			continue
		}
		// -s short options
		if strings.HasPrefix(arg, "-") {
			name := strings.TrimPrefix(arg, "-")
			if opt := optionByName(command, name); opt != nil {
				var boolean = true
				if opt.Field().Type == reflect.TypeOf(boolean) {
					continue
				}
				// Else skip the option argument (next item)
				i++
			}
			continue
		}
		processed = append(processed, arg)
	}

	return
}

// Other Functions -------------------------------------------------------------------------------------------------------------//

// formatInput - Formats & sanitize the command line input
func formatInput(line []rune) (args []string, last []rune, lastWord string) {
	args = strings.Split(string(line), " ")         // The readline input as a []string
	last = trimSpaceLeft([]rune(args[len(args)-1])) // The last char in input
	lastWord = string(last)
	return
}

func trimSpaceLeft(in []rune) []rune {
	firstIndex := len(in)
	for i, r := range in {
		if unicode.IsSpace(r) == false {
			firstIndex = i
			break
		}
	}
	return in[firstIndex:]
}

func equal(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasPrefix(r, prefix []rune) bool {
	if len(r) < len(prefix) {
		return false
	}
	return equal(r[:len(prefix)], prefix)
}
