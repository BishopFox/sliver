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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// SocksRmCmd - Remove an existing socks5 port
func SocksRmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	socksID := ctx.Flags.Int("id")
	if socksID < 1 {
		con.PrintErrorf("Must specify a valid Socks id\n")
		return
	}
	_, err := con.Rpc.CloseSocks(context.Background(), &sliverpb.SocksInfo{
		ID: uint32(socksID),
	})
	if err != nil {
		con.PrintErrorf("No Socks5 with id %d %s\n", socksID, err.Error())
	} else {
		con.PrintInfof("Removed Socks5\n")
	}
}
