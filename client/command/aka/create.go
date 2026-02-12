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
	defaultArgs := args[2:] // 其他任何内容都将成为别名的默认参数
	// 其余参数将作为该 alias 的默认参数

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
	// 将这个新 alias 保存到磁盘
	err := SaveAkaAliases()
	if err != nil {
		con.PrintErrorf("Failed to save alias '%s' to disk: %s\n", aliasName, err)
		// we still technically added it to the map, so the alias did get created in
		// 从技术上看我们仍将其加入了 map，因此该 alias 实际已在
		// memory so we should still print the Info message below
		// 内存中创建，所以仍应打印下方的 Info 消息
	}

	con.PrintInfof("Create alias for '%s' -> '%v'\n", aliasName, alias.Description)
}
