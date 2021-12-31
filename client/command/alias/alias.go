package alias

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
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// AliasesCmd - The alias command
func AliasesCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	if 0 < len(loadedAliases) {
		PrintAliases(con)
	} else {
		con.PrintInfof("No aliases installed, use the 'armory' command to automatically install some\n")
	}
}

// PrintAliases - Print a list of loaded aliases
func PrintAliases(con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Name",
		"Command Name",
		".NET Assembly",
		"Reflective",
		"Help",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})

	for _, alias := range loadedAliases {
		tw.AppendRow(table.Row{
			alias.Manifest.Name,
			alias.Manifest.CommandName,
			alias.Manifest.IsAssembly,
			alias.Manifest.IsReflective,
			alias.Manifest.Help,
		})
	}
	con.Println(tw.Render())
}

// AliasCommandNameCompleter - Completer for installed extensions command names
func AliasCommandNameCompleter(prefix string, args []string, con *console.SliverConsoleClient) []string {
	results := []string{}
	for name := range loadedAliases {
		if strings.HasPrefix(name, prefix) {
			results = append(results, name)
		}
	}
	return results
}
