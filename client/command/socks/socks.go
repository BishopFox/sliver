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
	"bytes"
	"context"
	"fmt"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

// SocksCmd - Displays information about the Socks5 port
func SocksCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	socks, err := con.Rpc.ListSocks(context.Background(), &commonpb.Empty{})
	if len(socks.List) == 0 || err != nil {
		con.PrintInfof("No port socks5\n")
		return
	}
	sort.Slice(socks.List[:], func(i, j int) bool {
		return socks.List[i].ID < socks.List[j].ID
	})
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "ID\tSession ID\tBind Address\tUsername\tPassword\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Session ID")),
		strings.Repeat("=", len("Bind Address")),
		strings.Repeat("=", len("Username")),
		strings.Repeat("=", len("Password")),
	)
	for _, p := range socks.List {
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\t\n", p.ID, p.Request.SessionID, p.BindAddr, p.Username, p.Password)
	}
	table.Flush()
	con.Printf("%s", outputBuf.String())
}
