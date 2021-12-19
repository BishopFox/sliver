package pivots

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

// StopPivotListenerCmd - Start a TCP pivot listener on the remote system
func StopPivotListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	id := uint32(ctx.Flags.Int("id"))
	if id == uint32(0) {
		pivotListeners, err := con.Rpc.PivotListeners(context.Background(), &sliverpb.PivotListenersReq{
			Request: con.ActiveTarget.Request(ctx),
		})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		if len(pivotListeners.Listeners) == 0 {
			con.PrintInfof("No pivot listeners running on this session\n")
			return
		}
		selectedListener, err := SelectPivotListener(pivotListeners.Listeners, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		id = selectedListener.ID
	}
	_, err := con.Rpc.PivotStopListener(context.Background(), &sliverpb.PivotStopListenerReq{
		ID:      id,
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Stopped pivot listener\n")
}
