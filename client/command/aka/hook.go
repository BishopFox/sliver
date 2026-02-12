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
		// 拼接用户传入参数，以便 shell 在可能情况下将其按无 flags 方式处理
		// available
		// （如果可行）
		if len(args[1:]) > 0 {
			expandedArgs = append(expandedArgs, strings.Join(args[1:], " "))
		}

		return expandedArgs, nil
	}

	return args, nil
}
