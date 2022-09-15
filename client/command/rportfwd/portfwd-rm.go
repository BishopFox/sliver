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
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"github.com/desertbit/grumble"
)

// StartRportFwdListenerCmd - Start listener for reverse port forwarding on implant
func StopRportFwdListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	lID := ctx.Flags.Int("id")

	rportfwdlistener, err := con.Rpc.StopRportfwdListener(context.Background(), &sliverpb.RportFwdStopListenerReq{
		Request: con.ActiveTarget.Request(ctx),
		ID:      uint32(lID),
	})
	if err != nil {

		con.PrintWarnf("%s\n", err)
		return
	}
	if rportfwdlistener.Response != nil && rportfwdlistener.Response.Async {
		con.AddBeaconCallback(rportfwdlistener.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, rportfwdlistener)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			con.PrintInfof("Stopped port forwarding %s -> %s\n", rportfwdlistener.BindAddress, rportfwdlistener.ForwardAddress)
		})
		con.PrintAsyncResponse(rportfwdlistener.Response)
	} else {
		con.PrintInfof("Stopped port forwarding %s -> %s\n", rportfwdlistener.BindAddress, rportfwdlistener.ForwardAddress)
	}
}
