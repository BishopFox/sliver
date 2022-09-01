package cursed

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// CursedChromeCmd - Execute a .NET assembly in-memory
func CursedCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	cursedProcesses := [][]string{}
	core.CursedProcesses.Range(func(key, value interface{}) bool {
		curse := value.(*core.CursedProcess)
		cursedProcesses = append(cursedProcesses, []string{
			fmt.Sprintf("%d", curse.BindTCPPort), strings.Split(curse.SessionID, "-")[0], curse.Platform, curse.ChromeExePath, curse.DebugURL().String(),
		})
		return true
	})
	if 0 < len(cursedProcesses) {
		tw := table.NewWriter()
		tw.SetStyle(settings.GetTableStyle(con))
		tw.AppendHeader(table.Row{
			"Bind Port", "Session ID", "Platform", "Executable", "Debug URL",
		})
		for _, rowEntries := range cursedProcesses {
			row := table.Row{}
			for _, entry := range rowEntries {
				row = append(row, entry)
			}
			tw.AppendRow(table.Row(row))
		}
		con.Printf("%s\n", tw.Render())
	} else {
		con.PrintInfof("No cursed processes\n")
	}
}
