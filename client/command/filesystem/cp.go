package filesystem

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
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func CpCmd(cmd *cobra.Command, con *console.SliverClient, args []string) (err error) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	if len(args) != 2 {
		con.PrintErrorf("Please specify a source and destination filename.\n")
		return
	}

	src := args[0]
	dst := args[1]

	cp, err := con.Rpc.Cp(context.Background(), &sliverpb.CpReq{
		Request: con.ActiveTarget.Request(cmd),
		Src:     src,
		Dst:     dst,
	})
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}

	cp.Src, cp.Dst = src, dst

	if cp.Response != nil && cp.Response.Async {
		con.AddBeaconCallback(cp.Response, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, cp)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
		})
	} else {
		PrintCp(cp, con)
	}

	return
}

func PrintCp(cp *sliverpb.Cp, con *console.SliverClient) {
	if cp.Response != nil && cp.Response.Err != "" {
		con.PrintErrorf("%s\n", cp.Response.Err)
		return
	}

	con.PrintInfof("Copied '%s' to '%s' (%d bytes written)\n", cp.Src, cp.Dst, cp.BytesWritten)
}
