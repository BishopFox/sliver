package aka

import (
	"strings"
  "fmt"
)

func Cmdhook(args []string) ([]string, error) {
  fmt.Printf("ARGS: %v\n", args)
  if len(args) == 0 {
    return args, nil
  }

  if aliasArgs, exists := akaAliases[args[0]]; exists {
    var expandedArgs []string

    for _, aliasArg := range aliasArgs {
      parts := strings.Fields(aliasArg)
      expandedArgs = append(expandedArgs, parts...)
    }

    expandedArgs = append(expandedArgs, args[1:]...)

    return expandedArgs, nil
  }

  return args, nil
}
