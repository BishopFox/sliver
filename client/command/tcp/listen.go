package tcp

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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ListenPivotCmd - Start a TCP pivot listener on the remote system.
func ListenPivotCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	bind, _ := cmd.Flags().GetString("bind")
	lport, _ := cmd.Flags().GetUint16("lport")
	listener, err := con.Rpc.PivotStartListener(context.Background(), &sliverpb.PivotStartListenerReq{
		Type:        sliverpb.PivotType_TCP,
		BindAddress: fmt.Sprintf("%s:%d", bind, lport),
		Request:     con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}
	if listener.Response != nil && listener.Response.Err != "" {
		con.PrintErrorf("%s\n", listener.Response.Err)
		return
	}
	con.PrintInfof("Started tcp pivot listener %s with id %d\n", listener.BindAddress, listener.ID)
}
