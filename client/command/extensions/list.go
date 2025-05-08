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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// ExtensionMatch holds the details of a matched extension

type ExtensionMatch struct {
	CommandName string
	Hash        string
	BinPath     string
}

// FindExtensionMatches searches through loaded extensions for matching hashes
// Returns a map of hash to ExtensionMatch (match will be nil if hash wasn't found)
func FindExtensionMatches(targetHashes []string) map[string]*ExtensionMatch {
	results := make(map[string]*ExtensionMatch)
	pathCache := make(map[string]*ExtensionMatch)

	// Initialize results map with all target hashes
	for _, hash := range targetHashes {
		results[hash] = nil
	}

	// Search for matches
	for _, extCmd := range loadedExtensions {
		if extCmd == nil || len(extCmd.Files) == 0 {
			continue
		}

		for _, file := range extCmd.Files {
			fullPath := filepath.Join(extCmd.Manifest.RootPath, file.Path)

			// Check cache first
			var match *ExtensionMatch
			if cached, exists := pathCache[fullPath]; exists {
				match = cached
			} else {
				// Calculate hash if not cached
				fileData, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}

				hashBytes := sha256.Sum256(fileData)
				fileHash := hex.EncodeToString(hashBytes[:])

				match = &ExtensionMatch{
					CommandName: extCmd.CommandName,
					Hash:        fileHash,
					BinPath:     fullPath,
				}
				pathCache[fullPath] = match
			}

			if slices.Contains(targetHashes, match.Hash) {
				results[match.Hash] = match
			}
		}
	}


	return results
}

// PrintExtensionMatches prints the extension matches in a formatted table
func PrintExtensionMatches(matches map[string]*ExtensionMatch, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Command Name",
		"Sha256 Hash",
		"Bin Path",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Command Name", Mode: table.Asc},
	})

	for hash, match := range matches {
		if match != nil {
			tw.AppendRow(table.Row{
				match.CommandName,
				hash,
				match.BinPath,
			})
		} else {
			tw.AppendRow(table.Row{
				"",
				hash,
				"",
			})
		}
	}

	con.Println(tw.Render())
}

// ExtensionsListCmd - List all extension loaded on the active session/beacon.
func ExtensionsListCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if extList.Response != nil && extList.Response.Err != "" {
		con.PrintErrorf("%s\n", extList.Response.Err)
		return
	}

	if len(extList.Names) == 0 {
		return
	}

	con.PrintInfof("Loaded extensions:\n\n")
	matches := FindExtensionMatches(extList.Names)
	PrintExtensionMatches(matches, con)
}
