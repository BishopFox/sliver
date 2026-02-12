package alias

/*
	Sliver Implant Framework
	Sliver implant 框架
	Copyright (C) 2021  Bishop Fox
	版权所有 (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	本程序是自由软件：你可以再发布和/或修改它
	it under the terms of the GNU General Public License as published by
	在自由软件基金会发布的 GNU General Public License 条款下，
	the Free Software Foundation, either version 3 of the License, or
	可以使用许可证第 3 版，或
	(at your option) any later version.
	（由你选择）任何更高版本。

	This program is distributed in the hope that it will be useful,
	发布本程序是希望它能发挥作用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但不提供任何担保；甚至不包括
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	对适销性或特定用途适用性的默示担保。请参阅
	GNU General Public License for more details.
	GNU General Public License 以获取更多细节。

	You should have received a copy of the GNU General Public License
	你应当已随本程序收到一份 GNU General Public License 副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	如果没有，请参见 <https://www.gnu.org/licenses/>。
*/

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/spf13/cobra"
)

// AliasesRemoveCmd - Locally load a alias into the Sliver shell.
// AliasesRemoveCmd - 在本地 Sliver shell 中移除 alias。
func AliasesRemoveCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name := args[0]
	// name := ctx.Args.String("name")
	// 从 ctx 读取 name 参数
	if name == "" {
		con.PrintErrorf("Extension name is required\n")
		return
	}
	confirm := false
	_ = forms.Confirm(fmt.Sprintf("Remove '%s' alias?", name), &confirm)
	if !confirm {
		return
	}
	err := RemoveAliasByCommandName(name, con)
	if err != nil {
		con.PrintErrorf("Error removing alias: %s\n", err)
		return
	} else {
		con.PrintInfof("Alias '%s' removed\n", name)
	}
}

// RemoveAliasByCommandName - Remove an alias by command name.
// RemoveAliasByCommandName - 按命令名移除 alias。
func RemoveAliasByCommandName(commandName string, con *console.SliverClient) error {
	if commandName == "" {
		return errors.New("command name is required")
	}
	if _, ok := loadedAliases[commandName]; !ok {
		return errors.New("alias not loaded")
	}
	delete(loadedAliases, commandName)
	// con.App.Commands().Remove(commandName)
	// 从 con.App.Commands() 中移除 commandName
	extPath := filepath.Join(assets.GetAliasesDir(), filepath.Base(commandName))
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		return nil
	}
	err := os.RemoveAll(extPath)
	if err != nil {
		return err
	}

	return nil
}
