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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

// SocksStopCmd - Remove an existing tunneled port forward.
func SocksStopCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	socksID, _ := cmd.Flags().GetUint64("id")
	if socksID < 1 {
		con.PrintErrorf("Must specify a valid socks5 id\n")
		return
	}
	found := core.SocksProxies.Remove(socksID)
	if !found {
		con.PrintErrorf("No socks5 with id %d\n", socksID)
	} else {
		con.PrintInfof("Removed socks5\n")
	}

	// close
	con.Rpc.CloseSocks(context.Background(), &sliverpb.Socks{})
}
