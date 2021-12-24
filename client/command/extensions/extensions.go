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
	PrintExtensions(con)
}

// PrintExtensions - Print a list of loaded extensions
func PrintExtensions(con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Name",
		"Platforms",
		"Version",
		"Extension Author",
		"Original Author",
		"Installed",
		"Repository",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 6, Align: text.AlignCenter},
	})

	installedManifests := getInstalledManifests()
	for _, extension := range loadedExtensions {
		installed := ""
		if _, ok := installedManifests[extension.Name]; ok {
			installed = "✅"
		}
		tw.AppendRow(table.Row{
			extension.Name,
			strings.Join(extensionPlatforms(extension), ","),
			extension.Version,
			extension.ExtensionAuthor,
			extension.OriginalAuthor,
			installed,
			extension.RepoURL,
		})
	}
	con.Println(tw.Render())
}

func extensionPlatforms(extension *ExtensionManifest) []string {
	platforms := []string{}
	for _, entry := range extension.Files {
		platforms = append(platforms, entry.OS)
	}
	return platforms
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
		installedManifests[manifest.Name] = manifest
	}
	return installedManifests
}
