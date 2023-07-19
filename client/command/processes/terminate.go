package processes

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
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// TerminateCmd - Terminate a process on the remote system
func TerminateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		con.PrintErrorf("No active session or beacon\n")
		return
	}

	pid, err := strconv.Atoi(args[0])
	if err != nil {
		con.PrintErrorf("Invalid PID: %s (could not convert to int)", args[0])
		return
	}

	force, _ := cmd.Flags().GetBool("force")

	terminated, err := con.Rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
		Request: con.ActiveTarget.Request(cmd),
		Pid:     int32(pid),
		Force:   force,
	})
	if err != nil {
		con.PrintErrorf("Terminate failed: %s", err)
		return
	}

	if terminated.Response != nil && terminated.Response.Async {
		con.AddBeaconCallback(terminated.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, terminated)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintTerminate(terminated, con)
		})
		con.PrintAsyncResponse(terminated.Response)
	} else {
		PrintTerminate(terminated, con)
	}
}

// PrintTerminate - Print the results of the terminate command
func PrintTerminate(terminated *sliverpb.Terminate, con *console.SliverClient) {
	if terminated.Response != nil && terminated.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", terminated.Response.GetErr())
	} else {
		con.PrintInfof("Process %d has been terminated\n", terminated.Pid)
	}
}
