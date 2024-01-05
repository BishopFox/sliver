package loot

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

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
)

// LootFetchCmd - Display the contents of or download a piece of loot
func LootFetchCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	loot, err := SelectLoot(cmd, con.Rpc)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	loot, err = con.Rpc.LootContent(context.Background(), loot)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintLootFile(loot, con)

	if save, _ := cmd.Flags().GetString("save"); save != "" {
		savedTo, err := saveLootToDisk(cmd, loot)
		if err != nil {
			con.PrintErrorf("Failed to save loot %s\n", err)
		}
		if savedTo != "" {
			con.PrintInfof("Saved loot to %s\n", savedTo)
		}
	}
}
