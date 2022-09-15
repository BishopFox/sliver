package rportfwd

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
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"github.com/desertbit/grumble"
)

// StartRportFwdListenerCmd - Start listener for reverse port forwarding on implant
func RportFwdListenersCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	rportfwdlisteners, err := con.Rpc.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {

		con.PrintWarnf("%s\n", err)
		return
	}
	if rportfwdlisteners.Response != nil && rportfwdlisteners.Response.Async {
		con.AddBeaconCallback(rportfwdlisteners.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, rportfwdlisteners)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintRportFwdListeners(rportfwdlisteners, ctx.Flags, con)
		})
		con.PrintAsyncResponse(rportfwdlisteners.Response)
	} else {
		PrintRportFwdListeners(rportfwdlisteners, ctx.Flags, con)
	}
}

func PrintRportFwdListeners(rportfwdlisteners *sliverpb.RportFwdListeners, flags grumble.FlagMap, con *console.SliverConsoleClient) {
	if rportfwdlisteners.Response != nil && rportfwdlisteners.Response.Err != "" {
		con.PrintErrorf("%s\n", rportfwdlisteners.Response.Err)
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	for _, listener := range rportfwdlisteners.Listeners {
		fmt.Fprintf(table, "%d\t%s -> %s\n", listener.ID, listener.BindAddress, listener.ForwardAddress)
	}

	table.Flush()
	con.Printf("%s\n", outputBuf.String())

}
