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

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/aka"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/backdoor"
	"github.com/bishopfox/sliver/client/command/cursed"
	"github.com/bishopfox/sliver/client/command/dllhijack"
	"github.com/bishopfox/sliver/client/command/edit"
	"github.com/bishopfox/sliver/client/command/environment"
	"github.com/bishopfox/sliver/client/command/exec"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/network"
	"github.com/bishopfox/sliver/client/command/pivots"
	"github.com/bishopfox/sliver/client/command/portfwd"
	"github.com/bishopfox/sliver/client/command/privilege"
	"github.com/bishopfox/sliver/client/command/processes"
	"github.com/bishopfox/sliver/client/command/reconfig"
	"github.com/bishopfox/sliver/client/command/registry"
	"github.com/bishopfox/sliver/client/command/rportfwd"
	"github.com/bishopfox/sliver/client/command/screenshot"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/shell"
	"github.com/bishopfox/sliver/client/command/socks"
	"github.com/bishopfox/sliver/client/command/tasks"
	"github.com/bishopfox/sliver/client/command/wasm"
	"github.com/bishopfox/sliver/client/command/wireguard"
	"github.com/bishopfox/sliver/client/command/hexedit"
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

// SliverCommands returns all commands bound to the implant menu.
// SliverCommands 返回绑定到 implant menu. 的所有命令
func SliverCommands(con *client.SliverClient) console.Commands {
	sliverCommands := func() *cobra.Command {
		sliver := &cobra.Command{
			Short: "Implant commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}
		if !con.IsCLI {
			sliver.SilenceErrors = true
			sliver.SilenceUsage = true
		}

		// Utility function to be used for binding new commands to
		// Utility 函数用于将新命令绑定到
		// the sliver menu: call the function with the name of the
		// sliver 菜单：使用名称调用该函数
		// group under which this/these commands should be added,
		// 应添加 this/these 命令的组，
		// and the group will be automatically created if needed.
		// 如果 needed. ，该组将自动创建
		bind := makeBind(sliver, con)

		// [ Core ]
		bind(consts.SliverCoreHelpGroup,
			reconfig.Commands,
			// sessions.Commands,
			// sessions.Commands，
			sessions.SliverCommands,
			kill.Commands,
			// use.Commands,
			// use.Commands，
			tasks.Commands,
			pivots.Commands,
			aka.ImplantCommands,
		)

		// [ Info ]
		bind(consts.InfoHelpGroup,
			// info.Commands,
			// info.Commands，
			info.SliverCommands,
			screenshot.Commands,
			environment.Commands,
			registry.Commands,
			extensions.SliverCommands,
		)

		// [ Filesystem ]
		bind(consts.FilesystemHelpGroup,
			edit.Commands,
			hexedit.Commands,
			filesystem.Commands,
		)

		// [ Network tools ]
		// [ Network 工具 ]
		bind(consts.NetworkHelpGroup,
			network.Commands,
			rportfwd.Commands,
			portfwd.Commands,
			socks.Commands,
			wireguard.SliverCommands,
		)

		// [ Execution ]
		bind(consts.ExecutionHelpGroup,
			shell.Commands,
			exec.Commands,
			backdoor.Commands,
			dllhijack.Commands,
			cursed.Commands,
			wasm.Commands,
		)

		// [ Privileges ]
		bind(consts.PrivilegesHelpGroup,
			privilege.Commands,
		)

		// [ Processes ]
		bind(consts.ProcessHelpGroup,
			processes.Commands,
		)

		// [ Aliases ]
		bind(consts.AliasHelpGroup)

		// [ Extensions ]
		bind(consts.ExtensionHelpGroup)

		// [ Post-command declaration setup ]----------------------------------------
		// [ Post__PH0__ 声明设置 ]----------------------------------------

		// Load Aliases
		aliasManifests := assets.GetInstalledAliasManifests()
		for _, manifest := range aliasManifests {
			_, err := alias.LoadAlias(manifest, sliver, con)
			if err != nil {
				con.PrintErrorf("Failed to load alias: %s", err)
				continue
			}
		}

		// Load Extensions
		extensionManifests := extensions.GetAllExtensionManifests()
		for _, manifest := range extensionManifests {
			mext, err := extensions.LoadExtensionManifest(manifest)
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

			for _, ext := range mext.ExtCommand {
				extensions.ExtensionRegisterCommand(ext, sliver, con)
			}
		}

		// [ Post-command declaration setup ]----------------------------------------
		// [ Post__PH0__ 声明设置 ]----------------------------------------

		// Everything below this line should preferably not be any command binding
		// 该行下方的 Everything 最好不要有任何命令绑定
		// (although you can do so without fear). If there are any final modifications
		// （尽管您可以毫无恐惧地这样做）。 If 有任何最终修改
		// to make to the server menu command tree, it time to do them here.
		// 要创建服务器菜单命令树，是时候执行它们了 here.

		sliver.InitDefaultHelpCmd()
		sliver.SetHelpCommandGroupID(consts.SliverCoreHelpGroup)

		// Compute which commands should be available based on the current session/beacon.
		// Compute 根据当前 session/beacon. 哪些命令应该可用
		con.ExposeCommands()

		return sliver
	}

	return sliverCommands
}
