package aka

import (
	"strings"
)

func Cmdhook(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	if aliasArgs, exists := akaAliases[args[0]]; exists {
		var expandedArgs []string

		expandedArgs = append(expandedArgs, aliasArgs.Command)
		expandedArgs = append(expandedArgs, aliasArgs.DefaultArgs...)
		// join the user passed args so the shell processes it without flags if
		// available
		if len(args[1:]) > 0 {
			expandedArgs = append(expandedArgs, strings.Join(args[1:], " "))
		}

		return expandedArgs, nil
	}

	return args, nil
}
