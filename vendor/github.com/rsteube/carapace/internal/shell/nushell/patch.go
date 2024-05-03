package nushell

import (
	"strings"

	shlex "github.com/rsteube/carapace-shlex"
)

// Patch uses the lexer to parse and patch given arguments which
// are currently passed unprocessed to the completion function.
//
// see https://www.nushell.sh/book/working_with_strings.html
func Patch(args []string) []string {
	// TODO
	for index, arg := range args {
		if len(arg) == 0 {
			continue
		}

		switch arg[0] {
		case '"', "'"[0]:
			if tokens, err := shlex.Split(arg); err == nil {
				args[index] = tokens[0].Value
			}
		case '`':
			args[index] = strings.Trim(arg, "`")
		}
	}
	return args
}
