package armory

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
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
	armoryPK := getArmoryPublicKey(armoryName)

	// If the armory with the name is not found, print a warning
	if cmd.Flags().Changed("armory") && armoryPK == "" {
		con.PrintWarnf("Could not find a configured armory named %q - searching all configured armories\n\n", armoryName)
	}

	// Check packages for updates
	aliasUpdates := checkForAliasUpdates(armoryPK)
	extUpdates := checkForExtensionUpdates(armoryPK)

	// Display a table of results
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
				version in the armory. This worked when we had one armory, but with multiple potential armories,
				we have to have some type of versioning. We do not have an official versioning scheme for packages,
				so we will rely on the author of the package incrementing the version identifier somehow to determine
				if a package is newer.
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
				version in the armory. This worked when we had one armory, but with multiple potential armories,
				we have to have some type of versioning. We do not have an official versioning scheme for packages,
				so we will rely on the author of the package incrementing the version identifier somehow to determine
				if a package is newer.
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
		an alias and an extension exist with the same name, we cannot simply combine the two maps.

		We will assume that no two aliases and no two extensions have the same name.
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
	tw.SetTitle(console.Bold + fmt.Sprintf(title, len(aliasUpdates), aliasSuffix, len(extensionUpdates), extensionSuffix) + console.Normal)
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
