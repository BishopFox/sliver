package pivots

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

// PivotsCmd - Display pivots for all sessions
func PivotsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	timeout := ctx.Flags.Int("timeout")
	sessionID := ctx.Flags.String("id")
	if sessionID != "" {
		session := con.GetSession(sessionID)
		if session == nil {
			return
		}
		printPivots(session, int64(timeout), con)
	} else {
		session := con.ActiveSession.Get()
		if session != nil {
			printPivots(session, int64(timeout), con)
		} else {
			sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			for _, session := range sessions.Sessions {
				printPivots(session, int64(timeout), con)
			}
		}
	}
}

func printPivots(session *clientpb.Session, timeout int64, con *console.SliverConsoleClient) {
	pivotList, err := con.Rpc.ListPivots(context.Background(), &sliverpb.PivotListReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   timeout,
			Async:     false,
		},
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if pivotList.Response != nil && pivotList.Response.Err != "" {
		con.PrintErrorf("%s\n", pivotList.Response.Err)
		return
	}

	if len(pivotList.Entries) > 0 {
		con.PrintInfof("Session %d\n", session.ID)
		outputBuf := bytes.NewBufferString("")
		table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

		fmt.Fprintf(table, "type\taddress\t\n")
		fmt.Fprintf(table, "%s\t%s\t\n",
			strings.Repeat("=", len("type")),
			strings.Repeat("=", len("address")),
		)

		for _, entry := range pivotList.Entries {
			fmt.Fprintf(table, "%s\t%s\t\n", entry.Type, entry.Remote)
		}
		table.Flush()
		con.Printf(outputBuf.String())
	} else {
		con.PrintInfof("No pivots found for session %d\n", session.ID)
	}

}
