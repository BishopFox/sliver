package command

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox
	Copyright (C) 2023 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"strings"

	client "github.com/bishopfox/sliver/client/console"
	"github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Bind is a convenience function to bind flags to a given command.
// Bind 是一个方便的函数，用于将标志绑定到给定的 command.
// name - The name of the flag set (can be empty).
// name - The 标志集的名称（可以为空）。
// cmd  - The command to which the flags should be bound.
// cmd - The 命令，其标志应为 bound.
// flags - A function exposing the flag set through which flags are declared.
// flags - A 函数公开标志集，通过该标志集标志是 declared.
func Bind(name string, persistent bool, cmd *cobra.Command, flags func(f *pflag.FlagSet)) {
	flagSet := pflag.NewFlagSet(name, pflag.ContinueOnError) // Create the flag set.
	flagSet := pflag.NewFlagSet(name, pflag.ContinueOnError) // Create 标志 set.
	flags(flagSet)                                           // Let the user bind any number of flags to it.
	flags(flagSet)                                           // Let 用户将任意数量的标志绑定到 it.

	if persistent {
		cmd.PersistentFlags().AddFlagSet(flagSet)
	} else {
		cmd.Flags().AddFlagSet(flagSet)
	}
}

// BindFlagCompletions is a convenience function for adding completions to a command's flags.
// BindFlagCompletions 是一个方便的函数，用于将完成添加到命令的 flags.
// cmd - The command owning the flags to complete.
// cmd - The 命令拥有 complete. 标志
// bind - A function exposing a map["flag-name"]carapace.Action.
// 绑定 - A 函数公开地图 [__PH0__]carapace.Action.
func BindFlagCompletions(cmd *cobra.Command, bind func(comp *carapace.ActionMap)) {
	comps := make(carapace.ActionMap)
	bind(&comps)

	carapace.Gen(cmd).FlagCompletion(comps)
}

// RestrictTargets generates a cobra annotation map with a single console.CommandHiddenFilter key
// RestrictTargets 使用单个 console.CommandHiddenFilter 键生成眼镜蛇注释图
// to a comma-separated list of filters to use in order to expose/hide commands based on requirements.
// 到要使用的 comma__PH0__ 过滤器列表，以便基于 requirements. 执行 expose/hide 命令
// Ex: cmd.Annotations = RestrictTargets("windows") will only show the command if the target is Windows.
// Ex: cmd.Annotations = RestrictTargets(__PH0__) 仅当目标是 Windows. 时才会显示该命令
// Ex: cmd.Annotations = RestrictTargets("windows", "beacon") show the command if target is a beacon on Windows.
// Ex: cmd.Annotations = RestrictTargets(__PH0__, __PH1__) 如果目标是 Windows. 上的 beacon，则显示该命令
func RestrictTargets(filters ...string) map[string]string {
	if len(filters) == 0 {
		return nil
	}

	if len(filters) == 1 {
		return map[string]string{
			console.CommandFilterKey: filters[0],
		}
	}

	filts := strings.Join(filters, ",")

	return map[string]string{
		console.CommandFilterKey: filts,
	}
}

// makeBind returns a commandBinder helper function
// makeBind 返回 commandBinder 辅助函数
// @menu  - The command menu to which the commands should be bound (either server or implant menu).
// @menu - 命令应绑定到的 The 命令菜单（服务器或 implant 菜单）。
func makeBind(cmd *cobra.Command, con *client.SliverClient) func(group string, cmds ...func(con *client.SliverClient) []*cobra.Command) {
	return func(group string, cmds ...func(con *client.SliverClient) []*cobra.Command) {
		found := false

		// Ensure the given command group is available in the menu.
		// Ensure 给定的命令组在 menu. 中可用
		if group != "" {
			for _, grp := range cmd.Groups() {
				if grp.Title == group {
					found = true
					break
				}
			}

			if !found {
				cmd.AddGroup(&cobra.Group{
					ID:    group,
					Title: group,
				})
			}
		}

		// Bind the command to the root
		// Bind 给 root 的命令
		for _, command := range cmds {
			cmd.AddCommand(command(con)...)
		}
	}
}

// commandBinder is a helper used to bind commands to a given menu, for a given "command help group".
// commandBinder 是一个帮助程序，用于将命令绑定到给定菜单，对于给定的 __PH0__.
//
// @group - Name of the group under which the command should be shown. Preferably use a string in the constants package.
// @group - 组的 Name，命令应为 shown. Preferably 在常量 package. 中使用字符串
// @ cmds - A list of functions returning a list of root commands to bind. See any package's `commands.go` file and function.
// @ cmds - A 函数列表，将根命令列表返回到 bind. See 任何包的 __PH0__ 文件和 function.
// type commandBinder func(group string, cmds ...func(con *client.SliverClient) []*cobra.Command)
// 类型 commandBinder func(group string, cmds ...func(con *client.SliverClient) []*cobra.Command)

// [ Core ]
// [ Sessions ]
// [ Execution ]
// [ Filesystem ]
// [ Info ]
// [ Network (C2)]
// [ Network tools ]
// [ Network 工具 ]
// [ Payloads ]
// [ Privileges ]
// [ Processes ]
// [ Aliases ]
// [ Extensions ]

// Commands not to bind in CLI:
// Commands 不绑定在 CLI: 中
// - portforwarders
// - 港口货运代理
// - Socks (and wg-socks ?)
// - Socks （和 wg__PH0__ ？）
// - shell ?
// - shell ？

// Take care of:
// Take 照顾：
// - double bind help command
// - 双重绑定帮助命令
// - double bind session commands
// - 双重绑定 session 命令
// - don't bind readline command in CLI.
// - 不要在 CLI. 中绑定 readline 命令
