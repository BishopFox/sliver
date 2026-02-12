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
	"fmt"
	"os"

	"github.com/bishopfox/sliver/client/command/aka"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/armory"
	"github.com/bishopfox/sliver/client/command/beacons"
	"github.com/bishopfox/sliver/client/command/builders"
	"github.com/bishopfox/sliver/client/command/c2profiles"
	"github.com/bishopfox/sliver/client/command/certificates"
	"github.com/bishopfox/sliver/client/command/clean"
	"github.com/bishopfox/sliver/client/command/crack"
	"github.com/bishopfox/sliver/client/command/creds"
	"github.com/bishopfox/sliver/client/command/exit"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/hosts"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/jobs"
	"github.com/bishopfox/sliver/client/command/licenses"
	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/command/mcp"
	"github.com/bishopfox/sliver/client/command/monitor"
	"github.com/bishopfox/sliver/client/command/operators"
	"github.com/bishopfox/sliver/client/command/reaction"
	"github.com/bishopfox/sliver/client/command/serverctx"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/settings"
	shellcodeencoders "github.com/bishopfox/sliver/client/command/shellcode-encoders"
	sgn "github.com/bishopfox/sliver/client/command/shikata-ga-nai"
	"github.com/bishopfox/sliver/client/command/socks"
	"github.com/bishopfox/sliver/client/command/taskmany"
	"github.com/bishopfox/sliver/client/command/update"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/command/websites"
	"github.com/bishopfox/sliver/client/command/wireguard"
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

// ServerCommands returns all commands bound to the server menu, optionally
// ServerCommands 返回绑定到服务器菜单的所有命令，可选
// accepting a function returning a list of additional (admin) commands.
// 接受返回附加 (admin) commands. 列表的函数
func ServerCommands(con *client.SliverClient, serverCmds func() []*cobra.Command) console.Commands {
	serverCommands := func() *cobra.Command {
		server := &cobra.Command{
			Short: "Server commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}
		if !con.IsCLI {
			server.SilenceErrors = true
			server.SilenceUsage = true
		}

		// Utility function to be used for binding new commands to
		// Utility 函数用于将新命令绑定到
		// the sliver menu: call the function with the name of the
		// sliver 菜单：使用名称调用该函数
		// group under which this/these commands should be added,
		// 应添加 this/these 命令的组，
		// and the group will be automatically created if needed.
		// 如果 needed. ，该组将自动创建
		bind := makeBind(server, con)

		if serverCmds != nil {
			server.AddGroup(&cobra.Group{ID: consts.MultiplayerHelpGroup, Title: consts.MultiplayerHelpGroup})
			server.AddCommand(serverCmds()...)
		}

		// [ Bind commands ] --------------------------------------------------------
		// [ Bind 命令 ] --------------------------------------------------------

		// Below are bounds all commands of the server menu, gathered by the group
		// Below 是服务器菜单的所有命令的边界，由组收集
		// under which they should be printed in help messages and/or completions.
		// 它们应该打印在帮助消息中 and/or completions.
		// You can either add a new bindCommands() call with a new group (which will
		// You 可以添加一个新的 bindCommands() 调用和一个新组（这将
		// be automatically added to the command tree), or add your commands in one of
		// 自动添加到命令树中），或者将您的命令添加到其中之一
		// the present calls.
		// 现在的calls.

		// Core
		bind(consts.GenericHelpGroup,
			exit.Command,
			serverctx.Commands,
			licenses.Commands,
			settings.Commands,
			alias.Commands,
			extensions.Commands,
			armory.Commands,
			update.Commands,
			operators.Commands,
			creds.Commands,
			crack.Commands,
			certificates.Commands,
			clean.Command,
			aka.ServerCommands,
		)

		// C2 Network
		bind(consts.NetworkHelpGroup,
			jobs.Commands,
			mcp.Commands,
			websites.Commands,
			wireguard.Commands,
			c2profiles.Commands,
			socks.RootCommands,
		)

		// Payloads
		bind(consts.PayloadsHelpGroup,
			sgn.Commands,
			shellcodeencoders.Commands,
			generate.Commands,
			builders.Commands,
		)

		// Slivers
		bind(consts.SliverHelpGroup,
			use.Commands,
			info.Commands,
			sessions.Commands,
			beacons.Commands,
			monitor.Commands,
			loot.Commands,
			hosts.Commands,
			reaction.Commands,
			taskmany.Command,
		)

		// [ Post-command declaration setup ]-----------------------------------------
		// [ Post__PH0__ 声明设置 ]--------------------------------------------

		// Load Extensions
		// Similar to the SliverCommand loading, without adding the commands to the
		// Similar 到 SliverCommand 加载，而不将命令添加到
		// Server command tree. This is done to ensure that the extensions are loaded
		// Server 命令 tree. This 完成以确保加载扩展
		// before the server is started, so that the extensions are registered.
		// 在服务器启动之前，以便扩展名是 registered.
		extensionManifests := extensions.GetAllExtensionManifests()
		for _, manifest := range extensionManifests {
			_, err := extensions.LoadExtensionManifest(manifest)
			// Absorb error in case there's no extensions manifest
			// 如果没有扩展清单，则会出现 Absorb 错误
			if err != nil {
				//con doesn't appear to be initialised here?
				//con 似乎没有在这里初始化？
				//con.PrintErrorf("Failed to load extension: %s", err)
				//con.PrintErrorf（__PH0__，错误）
				fmt.Printf("Failed to load extension: %s\n", err)
				continue
			}

			//for _, ext := range mext.ExtCommand {
			//for _, ext := 范围 mext.ExtCommand {
			//	extensions.ExtensionRegisterCommand(ext, sliver, con)
			//	extensions.ExtensionRegisterCommand（分机，sliver，con）
			//}
		}

		// Everything below this line should preferably not be any command binding
		// 该行下方的 Everything 最好不要有任何命令绑定
		// (although you can do so without fear). If there are any final modifications
		// （尽管您可以毫无恐惧地这样做）。 If 有任何最终修改
		// to make to the server menu command tree, it time to do them here.
		// 要创建服务器菜单命令树，是时候执行它们了 here.

		// Only load reactions when the console is going to be started.
		// 当控制台将是 started. 时 Only 负载反应
		if !con.IsCLI {
			n, err := reaction.LoadReactions()
			if err != nil && !os.IsNotExist(err) {
				con.PrintErrorf("Failed to load reactions: %s\n", err)
			} else if n > 0 {
				con.PrintInfof("Loaded %d reaction(s) from disk\n", n)
			}
		}

		server.InitDefaultHelpCmd()
		server.SetHelpCommandGroupID(consts.GenericHelpGroup)

		return server
	}

	return serverCommands
}
