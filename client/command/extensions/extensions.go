package extensions

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

// ExtensionsCmd - List information about installed extensions.
func ExtensionsCmd(cmd *cobra.Command, con *console.SliverClient) {
	if 0 < len(getInstalledManifests()) {
		PrintExtensions(con)
	} else {
		con.PrintInfof("No extensions installed, use the 'armory' command to automatically install some\n")
	}
}

// PrintExtensions - Print a list of loaded extensions.
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
		installed := ""
		if _, ok := installedManifests[extension.Manifest.Name]; ok {
			installed = "✅"
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
		installedManifests[manifest.Name] = manifest
	}
	return installedManifests
}

// ExtensionsCommandNameCompleter - Completer for installed extensions command names.
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
