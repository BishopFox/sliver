package c2

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

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// TCPPivot - Start a TCP pivot listener
type TCPPivot struct {
	Options struct {
		LHost string `long:"lhost" short:"l" description:"interface address to bind listener to" default:"0.0.0.0"`
		LPort int    `long:"lport" short:"p" description:"listener TCP listen port" default:"9898"`
	} `group:"mTLS listener options"`
}

// Execute - Start a TCP pivot listener
func (tp *TCPPivot) Execute(args []string) (err error) {

	server := tp.Options.LHost
	lport := uint16(tp.Options.LPort)
	address := fmt.Sprintf("%s:%d", server, lport)

	_, err = transport.RPC.TCPListener(context.Background(), &sliverpb.TCPPivotReq{
		Address: address,
		Request: core.ActiveSessionRequest(),
	})

	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}

	fmt.Printf(util.Info+"Listening on tcp://%s \n", address)
	return
}

// NamedPipePivot - Start a Named pipe pivot listener
type NamedPipePivot struct {
	Options struct {
		Name string `long:"name" short:"n" description:"name of the pipe" required:"yes"`
	} `group:"named pipe options"`
}

// Execute - Start a named pipe pivot listener
func (tp *NamedPipePivot) Execute(args []string) (err error) {

	pipeName := tp.Options.Name
	_, err = transport.RPC.NamedPipes(context.Background(), &sliverpb.NamedPipesReq{
		PipeName: pipeName,
		Request:  core.ActiveSessionRequest(),
	})

	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}

	fmt.Printf(util.Info+"Listening on %s", "\\\\.\\pipe\\"+pipeName+" \n")
	return
}

// Pivots - Pivots management command, prints them by default
type Pivots struct {
	Options struct {
		SessionID int32 `long:"id" short:"i" description:"session for which to print pivots"`
	} `group:"pivot options"`
}

// Execute - Pivots management command, prints them by default
func (p *Pivots) Execute(args []string) (err error) {
	rpc := transport.RPC
	timeout := core.GetCommandTimeout()
	sessionID := p.Options.SessionID
	if sessionID != 0 {
		session := core.GetSession(string(sessionID))
		if session == nil {
			return
		}
		printPivots(session, int64(timeout), rpc)
	} else {
		session := core.ActiveSession
		if session != nil {
			printPivots(session.Session, int64(timeout), rpc)
		} else {
			sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
			if err != nil {
				fmt.Printf(util.Error+"Error: %v", err)
				return nil
			}
			if len(sessions.Sessions) == 0 {
				fmt.Printf(util.Info + "No pivoted sessions \n")
				return nil
			}
			for _, session := range sessions.Sessions {
				printPivots(session, int64(timeout), rpc)
			}
		}
	}
	return
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
		fmt.Printf(util.Error+"Error: %v", err)
		return
	}

	if pivotList.Response != nil && pivotList.Response.Err != "" {
		fmt.Printf(util.Error+"Error: %s", pivotList.Response.Err)
		return
	}

	if len(pivotList.Entries) > 0 {
		fmt.Printf(util.Info+"Session %d\n", session.ID)
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
		fmt.Printf(util.Info+"No pivots found for session %d\n", session.ID)
	}

}
