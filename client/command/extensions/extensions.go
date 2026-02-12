package extensions

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// ExtensionsCmd - List information about installed extensions.
// ExtensionsCmd - List 有关已安装 extensions. 的信息
func ExtensionsCmd(cmd *cobra.Command, con *console.SliverClient) {
	if len(GetAllExtensionManifests()) > 0 {
		PrintExtensions(con)
	} else {
		con.PrintInfof("No extensions installed, use the 'armory' command to automatically install some\n")
	}
}

// PrintExtensions - Print a list of loaded extensions.
// PrintExtensions - Print 已加载 extensions. 的列表
func PrintExtensions(con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Name",
		"Command Name",
		"Platforms",
		"Version",
		"Installed",
		"Extension Author",
		"Original Author",
		"Repository",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 5, Align: text.AlignCenter},
	})

	installedManifests := getInstalledManifests()
	for _, extension := range loadedExtensions {
		//for _, extension := range extensionm.ExtCommand {
		//对于 _，扩展名 := 范围 extensionm.ExtCommand {
		installed := ""
		//if _, ok := installedManifests[extension.Manifest.Name]; ok {
		//如果_，好的：= installedManifests[extension.Manifest.Name];好的 {
		for _, installedManifest := range installedManifests {
			if extension.Manifest.RootPath == installedManifest.RootPath {
				installed = "✅"
				break
			}
		}
		tw.AppendRow(table.Row{
			extension.Manifest.Name,
			extension.CommandName,
			strings.Join(extensionPlatforms(extension), ",\n"),
			extension.Manifest.Version,
			installed,
			extension.Manifest.ExtensionAuthor,
			extension.Manifest.OriginalAuthor,
			extension.Manifest.RepoURL,
		})
		//}
	}
	con.Println(tw.Render())
}

func extensionPlatforms(extension *ExtCommand) []string {
	platforms := map[string]string{}
	for _, entry := range extension.Files {
		platforms[fmt.Sprintf("%s/%s", entry.OS, entry.Arch)] = ""
	}
	keys := []string{}
	for key := range platforms {
		keys = append(keys, key)
	}
	return keys
}

// getInstalledManifests - Returns a mapping of extension names to their parsed manifest objects.
// getInstalledManifests - Returns 扩展名称到其解析清单的映射 objects.
// Reads all installed extension manifests from disk, ignoring any that cannot be read or parsed.
// Reads 来自磁盘的所有已安装扩展清单，忽略任何无法读取的或 parsed.
// The returned manifests have their RootPath set to the directory containing their manifest file.
// The 返回的清单将其 RootPath 设置为包含其清单 file. 的目录
func getInstalledManifests() map[string]*ExtensionManifest {
	manifestPaths := assets.GetInstalledExtensionManifests()
	installedManifests := map[string]*ExtensionManifest{}
	for _, manifestPath := range manifestPaths {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		manifest := &ExtensionManifest{}
		err = json.Unmarshal(data, manifest)
		if err != nil {
			continue
		}
		manifest.RootPath = filepath.Dir(manifestPath)
		installedManifests[manifest.Name] = manifest
	}
	return installedManifests
}

// getTemporarilyLoadedManifests returns a map of extension manifests that are currently
// getTemporarilyLoadedManifests 返回当前扩展清单的映射
// loaded into memory but not permanently installed. The map is keyed by the manifest's
// 加载到内存中但不是永久的 installed. The 映射由清单的键控
// Name field.
func getTemporarilyLoadedManifests() map[string]*ExtensionManifest {
	tempManifests := map[string]*ExtensionManifest{}
	for name, manifest := range loadedManifests {
		tempManifests[name] = manifest
	}
	return tempManifests
}

// GetAllExtensionManifests returns a combined list of manifest file paths from
// GetAllExtensionManifests 返回清单文件路径的组合列表
// both installed and temporarily loaded extensions
// 已安装和临时加载的扩展
func GetAllExtensionManifests() []string {
	manifestPaths := make(map[string]struct{}) // use map for deduplication
	manifestPaths := make(map[string]struct{}) // 使用map进行重复数据删除

	// Add installed manifests
	// Add 安装清单
	for _, manifest := range getInstalledManifests() {
		manifestPath := filepath.Join(manifest.RootPath, ManifestFileName)
		manifestPaths[manifestPath] = struct{}{}
	}

	// Add temporarily loaded manifests
	// Add 临时加载的清单
	for _, manifest := range getTemporarilyLoadedManifests() {
		manifestPath := filepath.Join(manifest.RootPath, ManifestFileName)
		manifestPaths[manifestPath] = struct{}{}
	}

	// Convert to slice
	// Convert 进行切片
	paths := make([]string, 0, len(manifestPaths))
	for path := range manifestPaths {
		paths = append(paths, path)
	}
	return paths
}

// ExtensionsCommandNameCompleter - Completer for installed extensions command names.
// ExtensionsCommandNameCompleter - Completer 用于已安装的扩展命令 names.
func ExtensionsCommandNameCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		//installedManifests := getInstalledManifests()
		results := []string{}
		for _, manifest := range loadedExtensions {
			results = append(results, manifest.CommandName)
			results = append(results, manifest.Help)
		}

		return carapace.ActionValuesDescribed(results...).Tag("extension commands")
	})
}

func ManifestCompleter() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := []string{}
		for k := range loadedManifests {
			results = append(results, k)
		}
		return carapace.ActionValues(results...).Tag("extensions")
	})
}
