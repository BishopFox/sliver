package aka

import (
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

func AkaCreateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	aliasName := args[0]
	command := args[1]
	defaultArgs := args[2:] // anything else is going to be the default args to the alias

	if _, exists := akaAliases[aliasName]; exists {
		con.PrintErrorf("Alias '%s' already exists\n", aliasName)
		return
	}

	var desc string
	if len(defaultArgs) == 0 {
		desc = fmt.Sprintf("%s", command)
	} else {
		desc = fmt.Sprintf("%s %s", command, strings.Join(defaultArgs, " "))
	}

	alias := &AkaAlias{
		Name:        aliasName,
		Command:     command,
		DefaultArgs: defaultArgs,
		Description: desc,
	}

	akaAliases[aliasName] = alias

	// save this new alias to disk
	err := SaveAkaAliases()
	if err != nil {
		con.PrintErrorf("Failed to save alias '%s' to disk: %s\n", aliasName, err)
		// we still technically added it to the map, so the alias did get created in
		// memory so we should still print the Info message below
	}

	con.PrintInfof("Create alias for '%s' -> '%v'\n", aliasName, alias.Description)
}
