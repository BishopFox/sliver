package hosts

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// HostsRmCmd - Remove a host from the database.
func HostsRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	host, err := SelectHost(con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	_, err = con.Rpc.HostRm(context.Background(), host)
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}
	con.PrintInfof("Removed host from database\n")
}
