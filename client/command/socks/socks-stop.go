package socks

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
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// SocksStopCmd - Remove an existing tunneled port forward.
func SocksStopCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	for _, arg := range args {
		socksID, err := strconv.ParseUint(arg, 10, 32)
		if err != nil {
			con.PrintErrorf("Failed to parse Socks ID: %s\n", err)
		}

		if socksID < 1 {
			con.PrintErrorf("Must specify a valid socks5 ID\n")
			return
		}

		found := core.SocksProxies.Remove(socksID)

		if !found {
			con.PrintErrorf("No socks5 with ID %d\n", socksID)
		} else {
			con.PrintInfof("Removed socks5\n")
		}

		// close
		con.Rpc.CloseSocks(context.Background(), &sliverpb.Socks{})
	}
}
