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
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/util"
)

type VersionInformation struct {
	OldVersion string
	NewVersion string
	ArmoryName string
}

type PackageType uint

const (
	AliasPackage PackageType = iota
	ExtensionPackage
)

type UpdateIdentifier struct {
	Type PackageType
	Name string
}

// ArmoryUpdateCmd - Update all installed extensions/aliases
// ArmoryUpdateCmd - 更新所有已安装 extension/alias
func ArmoryUpdateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var selectedUpdates []UpdateIdentifier
	var err error

	con.PrintInfof("Refreshing package cache ... ")
	clientConfig := parseArmoryHTTPConfig(cmd)
	refresh(clientConfig)
	con.Printf(console.Clearln + "\r")

	armoryName, err := cmd.Flags().GetString("armory")
	if err != nil {
		con.PrintErrorf("Could not parse %q flag: %s\n", "armory", err)
		return
	}

	// Find PK for the armory name
	// 根据 armory 名称查找 PK
	armoryPK := getArmoryPublicKey(armoryName)

	// If the armory with the name is not found, print a warning
	// 如果未找到该名称的 armory，则打印告警
	if cmd.Flags().Changed("armory") && armoryPK == "" {
		con.PrintWarnf("Could not find a configured armory named %q - searching all configured armories\n\n", armoryName)
	}

	// Check packages for updates
	// 检查 package 更新
	aliasUpdates := checkForAliasUpdates(armoryPK)
	extUpdates := checkForExtensionUpdates(armoryPK)

	// Display a table of results
	// 显示结果表格
	if len(aliasUpdates) > 0 || len(extUpdates) > 0 {
		updateKeys := sortUpdateIdentifiers(aliasUpdates, extUpdates)
		displayAvailableUpdates(con, updateKeys, aliasUpdates, extUpdates)
		selectedUpdates, err = getUpdatesFromUser(updateKeys)
		if err != nil {
			if errors.Is(err, forms.ErrUserAborted) {
				return
			}
			con.PrintErrorf("%s\n", err.Error())
			return
		}
		if len(selectedUpdates) == 0 {
			return
		}
	} else {
		con.PrintSuccessf("All packages are up to date\n")
		return
	}

	for _, update := range selectedUpdates {
		switch update.Type {
		case AliasPackage:
			aliasVersionInfo, ok := aliasUpdates[update.Name]
			if !ok {
				continue
			}
			updatedPackage, err := getPackageForCommand(update.Name, armoryPK, aliasVersionInfo.NewVersion)
			if err != nil {
				con.PrintErrorf("Could not get update package for alias %s: %s\n", update.Name, err)
				// Keep going because other packages might not encounter errors
				// 继续执行，因为其他 package 可能不会遇到错误
				continue
			}
			err = installAliasPackage(updatedPackage, false, clientConfig, con)
			if err != nil {
				con.PrintErrorf("Failed to update %s: %s\n", update.Name, err)
			}
		case ExtensionPackage:
			extVersionInfo, ok := extUpdates[update.Name]
			if !ok {
				continue
			}
			updatedPackage, err := getPackageForCommand(update.Name, armoryPK, extVersionInfo.NewVersion)
			if err != nil {
				con.PrintErrorf("Could not get update package for extension %s: %s\n", update.Name, err)
				continue
			}
			err = installExtensionPackage(updatedPackage, false, clientConfig, con)
			if err != nil {
				con.PrintErrorf("Failed to update %s: %s\n", update.Name, err)
			}
		default:
			continue
		}
	}
}

func checkForAliasUpdates(armoryPK string) map[string]VersionInformation {
	cachedAliases, _ := packageManifestsInCache()
	results := make(map[string]VersionInformation)
	for _, aliasManifestPath := range assets.GetInstalledAliasManifests() {
		data, err := os.ReadFile(aliasManifestPath)
		if err != nil {
			continue
		}
		localManifest, err := alias.ParseAliasManifest(data)
		if err != nil {
			continue
		}
		for _, latestAlias := range cachedAliases {
			/*
				We used to check if the version identifiers were different between the installed version and the
				我们过去会检查已安装版本与 armory 版本的版本标识是否不同。
				version in the armory. This worked when we had one armory, but with multiple potential armories,
				这种方式在只有一个 armory 时可行，但在存在多个 armory 时，
				we have to have some type of versioning. We do not have an official versioning scheme for packages,
				我们必须依赖某种版本机制。我们没有官方的 package 版本规范，
				so we will rely on the author of the package incrementing the version identifier somehow to determine
				因此只能依赖 package 作者通过递增版本标识来判断
				if a package is newer.
				某个 package 是否更新。
			*/
			if latestAlias.CommandName == localManifest.CommandName && latestAlias.Version > localManifest.Version {
				if latestAlias.ArmoryPK == armoryPK || armoryPK == "" {
					results[localManifest.CommandName] = VersionInformation{
						OldVersion: localManifest.Version,
						NewVersion: latestAlias.Version,
						ArmoryName: latestAlias.ArmoryName,
					}
				}
			}
		}
	}
	return results
}

func checkForExtensionUpdates(armoryPK string) map[string]VersionInformation {
	_, cachedExtensions := packageManifestsInCache()
	// Return a map of extension name to minimum version to install
	// 返回 extension 名称到最低安装版本的映射
	results := make(map[string]VersionInformation)
	for _, extManifestPath := range assets.GetInstalledExtensionManifests() {
		data, err := os.ReadFile(extManifestPath)
		if err != nil {
			continue
		}
		localManifest, err := extensions.ParseExtensionManifest(data)
		if err != nil {
			continue
		}
		for _, latestExt := range cachedExtensions {
			/*
				We used to check if the version identifiers were different between the installed version and the
				我们过去会检查已安装版本与 armory 版本的版本标识是否不同。
				version in the armory. This worked when we had one armory, but with multiple potential armories,
				这种方式在只有一个 armory 时可行，但在存在多个 armory 时，
				we have to have some type of versioning. We do not have an official versioning scheme for packages,
				我们必须依赖某种版本机制。我们没有官方的 package 版本规范，
				so we will rely on the author of the package incrementing the version identifier somehow to determine
				因此只能依赖 package 作者通过递增版本标识来判断
				if a package is newer.
				某个 package 是否更新。
			*/
			if latestExt.Name == localManifest.Name && latestExt.Version > localManifest.Version {
				if latestExt.ArmoryPK == armoryPK || armoryPK == "" {
					results[localManifest.Name] = VersionInformation{
						OldVersion: localManifest.Version,
						NewVersion: latestExt.Version,
						ArmoryName: latestExt.ArmoryName,
					}
				}
			}
		}
	}

	return results
}

func sortUpdateIdentifiers(aliasUpdates, extensionUpdates map[string]VersionInformation) []UpdateIdentifier {
	/*
		This function helps us keep updates straight when the user chooses from them. Just in case
		这个函数用于在用户选择更新项时保持映射关系清晰。因为可能存在
		an alias and an extension exist with the same name, we cannot simply combine the two maps.
		同名 alias 与 extension，所以不能直接合并两个 map。

		We will assume that no two aliases and no two extensions have the same name.
		我们假设 alias 内部与 extension 内部都不存在重名。
	*/

	result := []UpdateIdentifier{}

	aliasNames := util.Keys(aliasUpdates)
	extensionNames := util.Keys(extensionUpdates)
	for _, name := range aliasNames {
		result = append(result, UpdateIdentifier{
			Type: AliasPackage,
			Name: name,
		})
	}
	for _, name := range extensionNames {
		result = append(result, UpdateIdentifier{
			Type: ExtensionPackage,
			Name: name,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

func displayAvailableUpdates(con *console.SliverClient, updateKeys []UpdateIdentifier, aliasUpdates, extensionUpdates map[string]VersionInformation) {
	var (
		aliasSuffix     string
		extensionSuffix string
		title           string = "Available Updates (%d alias%s, %d extension%s)"
	)

	tw := table.NewWriter()
	tw.SetAutoIndex(true)
	tw.SetStyle(settings.GetTableStyle(con))
	if len(aliasUpdates) != 1 {
		aliasSuffix = "es"
	}
	if len(extensionUpdates) != 1 {
		extensionSuffix = "s"
	}
	tw.SetTitle(console.StyleBold.Render(fmt.Sprintf(title, len(aliasUpdates), aliasSuffix, len(extensionUpdates), extensionSuffix)))
	tw.AppendHeader(table.Row{
		"Package Name",
		"Package Type",
		"Installed Version",
		"Available Version",
	})

	for _, key := range updateKeys {
		var (
			packageName    string
			packageType    string
			packageVersion VersionInformation
			ok             bool
		)
		switch key.Type {
		case AliasPackage:
			packageVersion, ok = aliasUpdates[key.Name]
			if !ok {
				continue
			}
			packageName = key.Name
			packageType = "Alias"
		case ExtensionPackage:
			packageVersion, ok = extensionUpdates[key.Name]
			if !ok {
				continue
			}
			packageName = key.Name
			packageType = "Extension"
		default:
			continue
		}
		tw.AppendRow(table.Row{
			packageName,
			packageType,
			packageVersion.OldVersion,
			fmt.Sprintf("%s (Armory: %s)", packageVersion.NewVersion, packageVersion.ArmoryName),
		})
	}

	con.Printf("%s\n\n", tw.Render())
}

func getUpdatesFromUser(updateKeys []UpdateIdentifier) ([]UpdateIdentifier, error) {
	options := make([]forms.ArmoryUpdateOption, 0, len(updateKeys))
	optionIndex := make(map[string]UpdateIdentifier, len(updateKeys))

	for _, key := range updateKeys {
		packageType := "Alias"
		if key.Type == ExtensionPackage {
			packageType = "Extension"
		}
		optionID := fmt.Sprintf("%d:%s", key.Type, key.Name)
		optionLabel := fmt.Sprintf("%s (%s)", key.Name, packageType)
		options = append(options, forms.ArmoryUpdateOption{
			ID:    optionID,
			Label: optionLabel,
		})
		optionIndex[optionID] = key
	}

	result, err := forms.ArmoryUpdateForm(options)
	if err != nil {
		return nil, err
	}

	chosenUpdates := make([]UpdateIdentifier, 0, len(result.SelectedIDs))
	for _, selectedID := range result.SelectedIDs {
		if update, ok := optionIndex[selectedID]; ok {
			chosenUpdates = append(chosenUpdates, update)
		}
	}

	return chosenUpdates, nil
}
