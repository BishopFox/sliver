package command

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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

func namedPipeListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if session.OS != "windows" {
		fmt.Printf(Warn+"Not implemented for %s\n", session.OS)
		return
	}

	pipeName := ctx.Flags.String("name")

	if pipeName == "" {
		fmt.Printf(Warn + "-n parameter missing\n")
		return
	}

	_, err := rpc.NamedPipes(context.Background(), &sliverpb.NamedPipesReq{
		PipeName: pipeName,
		Request:  ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	fmt.Printf(Info+"Listening on %s", "\\\\.\\pipe\\"+pipeName)
}

func tcpListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	server := ctx.Flags.String("server")
	lport := uint16(ctx.Flags.Int("lport"))
	address := fmt.Sprintf("%s:%d", server, lport)

	_, err := rpc.TCPListener(context.Background(), &sliverpb.TCPPivotReq{
		Address: address,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	fmt.Printf(Info+"Listening on tcp://%s", address)
}

func listPivots(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	timeout := ctx.Flags.Int("timeout")
	sessionID := ctx.Flags.String("id")
	if sessionID != "" {
		session := GetSession(sessionID, rpc)
		if session == nil {
			return
		}
		printPivots(session, int64(timeout), rpc)
	} else {
		session := ActiveSession.Get()
		if session != nil {
			printPivots(session, int64(timeout), rpc)
		} else {
			sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
			if err != nil {
				fmt.Printf(Warn+"Error: %v", err)
				return
			}
			for _, session := range sessions.Sessions {
				printPivots(session, int64(timeout), rpc)
			}
		}
	}
}

func printPivots(session *clientpb.Session, timeout int64, rpc rpcpb.SliverRPCClient) {
	pivotList, err := rpc.ListPivots(context.Background(), &sliverpb.PivotListReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   timeout,
			Async:     false,
		},
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if pivotList.Response != nil && pivotList.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s", pivotList.Response.Err)
		return
	}

	if len(pivotList.Entries) > 0 {
		fmt.Printf(Info+"Session %d\n", session.ID)
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
		fmt.Printf(outputBuf.String())
	} else {
		fmt.Printf(Info+"No pivots found for session %d\n", session.ID)
	}

}
