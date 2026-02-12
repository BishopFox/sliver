package armory

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
	"regexp"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/console"
)

// ArmorySearchCmd - Search for packages by name
// ArmorySearchCmd - 按名称搜索 package
func ArmorySearchCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	con.PrintInfof("Refreshing package cache ... ")
	clientConfig := parseArmoryHTTPConfig(cmd)
	refresh(clientConfig)
	con.Printf(console.Clearln + "\r")

	rawNameExpr := args[0]
	// rawNameExpr := ctx.Args.String("name")
	// 从 ctx 读取 name 参数
	if rawNameExpr == "" {
		con.PrintErrorf("Please specify a search term!\n")
		return
	}
	nameExpr, err := regexp.Compile(rawNameExpr)
	if err != nil {
		con.PrintErrorf("Invalid regular expression: %s\n", err)
		return
	}

	aliases, exts := packageManifestsInCache()
	matchedAliases := []*alias.AliasManifest{}
	for _, a := range aliases {
		if nameExpr.MatchString(a.CommandName) {
			matchedAliases = append(matchedAliases, a)
		}
	}
	matchedExts := []*extensions.ExtensionManifest{}
	for _, extm := range exts {
		for _, ext := range extm.ExtCommand {
			if nameExpr.MatchString(ext.CommandName) {
				matchedExts = append(matchedExts, extm)
			}
		}
	}
	if len(matchedAliases) == 0 && len(matchedExts) == 0 {
		con.PrintInfof("No packages found matching '%s'\n", rawNameExpr)
		return
	}
	PrintArmoryPackages(matchedAliases, matchedExts, con)
}
