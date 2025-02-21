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
func ExtensionsCmd(cmd *cobra.Command, con *console.SliverClient) {
	if len(GetAllExtensionManifests()) > 0 {
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
		//if _, ok := installedManifests[extension.Manifest.Name]; ok {
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
// Reads all installed extension manifests from disk, ignoring any that cannot be read or parsed.
// The returned manifests have their RootPath set to the directory containing their manifest file.
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
// loaded into memory but not permanently installed. The map is keyed by the manifest's
// Name field.
func getTemporarilyLoadedManifests() map[string]*ExtensionManifest {
	tempManifests := map[string]*ExtensionManifest{}
	for name, manifest := range loadedManifests {
		tempManifests[name] = manifest
	}
	return tempManifests
}

// GetAllExtensionManifests returns a combined list of manifest file paths from
// both installed and temporarily loaded extensions
func GetAllExtensionManifests() []string {
	manifestPaths := make(map[string]struct{}) // use map for deduplication

	// Add installed manifests
	for _, manifest := range getInstalledManifests() {
		manifestPath := filepath.Join(manifest.RootPath, ManifestFileName)
		manifestPaths[manifestPath] = struct{}{}
	}

	// Add temporarily loaded manifests
	for _, manifest := range getTemporarilyLoadedManifests() {
		manifestPath := filepath.Join(manifest.RootPath, ManifestFileName)
		manifestPaths[manifestPath] = struct{}{}
	}

	// Convert to slice
	paths := make([]string, 0, len(manifestPaths))
	for path := range manifestPaths {
		paths = append(paths, path)
	}
	return paths
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
