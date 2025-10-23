package aka

import (
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

func AkaCreateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
  aliasName := args[0]
  rest := args[1:] // anything else; guaranteed to at least have one arg with Min args == 2

  if _, exists := akaAliases[aliasName]; exists {
    con.PrintErrorf("Alias '%s' already exists\n", aliasName)
    return
  }

  akaAliases[aliasName] = rest

  con.PrintInfof("Create alias for '%s' -> '%v'\n", aliasName, strings.Join(rest, " "))
}
