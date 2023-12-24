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
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/console"
)

// ArmoryUpdateCmd - Update all installed extensions/aliases
func ArmoryUpdateCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	con.PrintInfof("Refreshing package cache ... ")
	clientConfig := parseArmoryHTTPConfig(cmd)
	refresh(clientConfig)
	con.Printf(console.Clearln + "\r")

	// Aliases
	aliasUpdates := checkForAliasUpdates(clientConfig, con)
	if 0 < len(aliasUpdates) {
		con.PrintInfof("%d alias(es) out of date: %s\n", len(aliasUpdates), strings.Join(aliasUpdates, ", "))
		for _, aliasName := range aliasUpdates {
			err := installAliasPackageByName(aliasName, clientConfig, con)
			if err != nil {
				con.PrintErrorf("Failed to update %s: %s\n", aliasName, err)
			}
		}
	} else {
		con.PrintInfof("All aliases up to date!\n")
	}

	// Extensions
	extUpdates := checkForExtensionUpdates(clientConfig, con)
	if 0 < len(extUpdates) {
		con.PrintInfof("%d extension(s) out of date: %s\n", len(extUpdates), strings.Join(extUpdates, ", "))
		for _, extName := range extUpdates {
			err := installExtensionPackageByName(extName, clientConfig, con)
			if err != nil {
				con.PrintErrorf("Failed to update %s: %s\n", extName, err)
			}
		}
	} else {
		con.PrintInfof("All extensions up to date!\n")
	}
}

func checkForAliasUpdates(clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) []string {
	cachedAliases, _ := packagesInCache()
	results := []string{}
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
			// Right now we don't try to enforce any kind of versioning, it is assumed if the version from
			// the armory differs at all from the local version, the extension is out of date.
			if latestAlias.CommandName == localManifest.CommandName && latestAlias.Version != localManifest.Version {
				results = append(results, localManifest.CommandName)
			}
		}
	}
	return results
}

func checkForExtensionUpdates(clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) []string {
	_, cachedExtensions := packagesInCache()
	results := []string{}
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
			// Right now we don't try to enforce any kind of versioning, it is assumed if the version from
			// the armory differs at all from the local version, the extension is out of date.
			if latestExt.Name == localManifest.Name && latestExt.Version != localManifest.Version {
				results = append(results, localManifest.Name)
			}
		}
	}

	return results
}
