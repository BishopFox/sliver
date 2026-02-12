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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// AliasesCmd - The alias command.
// AliasesCmd - alias 命令。
func AliasesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) error {
	if 0 < len(loadedAliases) {
		PrintAliases(con)
	} else {
		con.PrintInfof("No aliases installed, use the 'armory' command to automatically install some\n")
	}

	return nil
}

// PrintAliases - Print a list of loaded aliases.
// PrintAliases - 打印已加载 alias 列表。
func PrintAliases(con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Name",
		"Command Name",
		"Platforms",
		"Version",
		"Installed",
		".NET Assembly",
		"Reflective",
		"Tool Author",
		"Repository",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 5, Align: text.AlignCenter},
	})

	installedManifests := getInstalledManifests()
	for _, aliasPkg := range loadedAliases {
		installed := ""
		if _, ok := installedManifests[aliasPkg.Manifest.CommandName]; ok {
			installed = "✅"
		}
		tw.AppendRow(table.Row{
			aliasPkg.Manifest.Name,
			aliasPkg.Manifest.CommandName,
			strings.Join(aliasPlatforms(aliasPkg.Manifest), ",\n"),
			aliasPkg.Manifest.Version,
			installed,
			aliasPkg.Manifest.IsAssembly,
			aliasPkg.Manifest.IsReflective,
			aliasPkg.Manifest.OriginalAuthor,
			aliasPkg.Manifest.RepoURL,
		})
	}
	con.Println(tw.Render())
}

// AliasCommandNameCompleter - Completer for installed extensions command names.
// AliasCommandNameCompleter - 已安装 extension 命令名的补全器。
func AliasCommandNameCompleter(prefix string, args []string, con *console.SliverClient) []string {
	results := []string{}
	for name := range loadedAliases {
		if strings.HasPrefix(name, prefix) {
			results = append(results, name)
		}
	}
	return results
}

func aliasPlatforms(aliasPkg *AliasManifest) []string {
	platforms := map[string]string{}
	for _, entry := range aliasPkg.Files {
		platforms[fmt.Sprintf("%s/%s", entry.OS, entry.Arch)] = ""
	}
	keys := []string{}
	for key := range platforms {
		keys = append(keys, key)
	}
	return keys
}

func getInstalledManifests() map[string]*AliasManifest {
	manifestPaths := assets.GetInstalledAliasManifests()
	installedManifests := map[string]*AliasManifest{}
	for _, manifestPath := range manifestPaths {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		manifest := &AliasManifest{}
		err = json.Unmarshal(data, manifest)
		if err != nil {
			continue
		}
		installedManifests[manifest.CommandName] = manifest
	}
	return installedManifests
}

// AliasCommandNameCompleter - Completer for installed extensions command names.
// AliasCommandNameCompleter - 已安装 extension 命令名的补全器。
func AliasCompleter() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := []string{}
		for name := range loadedAliases {
			results = append(results, name)
		}
		return carapace.ActionValues(results...).Tag("aliases")
	})
}
