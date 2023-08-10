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
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// StopPivotListenerCmd - Start a TCP pivot listener on the remote system.
func StopPivotListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	id, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		con.PrintErrorf("Failed to parse pivot ID: %s\n", err)
		return
	}

	if id == 0 {
		pivotListeners, err := con.Rpc.PivotSessionListeners(context.Background(), &sliverpb.PivotListenersReq{
			Request: con.ActiveTarget.Request(cmd),
		})
		if err != nil {
			con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
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
		id = uint64(selectedListener.ID)
	}
	_, err = con.Rpc.PivotStopListener(context.Background(), &sliverpb.PivotStopListenerReq{
		ID:      uint32(id),
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}
	con.PrintInfof("Stopped pivot listener\n")
}
