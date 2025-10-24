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

    for _, aliasArg := range aliasArgs {
      parts := strings.Fields(aliasArg)
      expandedArgs = append(expandedArgs, parts...)
    }

    // join the user passed args so the shell processes it without flags
    expandedArgs = append(expandedArgs, strings.Join(args[1:], " "))

    return expandedArgs, nil
  }

  return args, nil
}
