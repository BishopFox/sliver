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
	"io/ioutil"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// ExtensionsCmd - List information about installed extensions
func ExtensionsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	if 0 < len(getInstalledManifests()) {
		PrintExtensions(con)
	} else {
		con.PrintInfof("No extensions installed, use the 'armory' command to automatically install some\n")
	}
}

// PrintExtensions - Print a list of loaded extensions
func PrintExtensions(con *console.SliverConsoleClient) {
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
		installed := ""
		if _, ok := installedManifests[extension.CommandName]; ok {
			installed = "✅"
		}
		tw.AppendRow(table.Row{
			extension.Name,
			extension.CommandName,
			strings.Join(extensionPlatforms(extension), ",\n"),
			extension.Version,
			installed,
			extension.ExtensionAuthor,
			extension.OriginalAuthor,
			extension.RepoURL,
		})
	}
	con.Println(tw.Render())
}

func extensionPlatforms(extension *ExtensionManifest) []string {
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
		data, err := ioutil.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		manifest := &ExtensionManifest{}
		err = json.Unmarshal(data, manifest)
		if err != nil {
			continue
		}
		installedManifests[manifest.CommandName] = manifest
	}
	return installedManifests
}

// ExtensionsCommandNameCompleter - Completer for installed extensions command names
func ExtensionsCommandNameCompleter(prefix string, args []string, con *console.SliverConsoleClient) []string {
	installedManifests := getInstalledManifests()
	results := []string{}
	for _, manifest := range installedManifests {
		if strings.HasPrefix(manifest.CommandName, prefix) {
			results = append(results, manifest.CommandName)
		}
	}
	return results
}
