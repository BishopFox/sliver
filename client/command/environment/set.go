package environment

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
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// EnvSetCmd - Set a remote environment variable
func EnvSetCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	name := ctx.Args.String("name")
	value := ctx.Args.String("value")
	if name == "" || value == "" {
		con.PrintErrorf("Usage: setenv KEY VALUE\n")
		return
	}

	envInfo, err := con.Rpc.SetEnv(context.Background(), &sliverpb.SetEnvReq{
		Variable: &commonpb.EnvVar{
			Key:   name,
			Value: value,
		},
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if envInfo.Response != nil && envInfo.Response.Async {
		con.AddBeaconCallback(envInfo.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, envInfo)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintSetEnvInfo(name, value, envInfo, con)
		})
		con.PrintAsyncResponse(envInfo.Response)
	} else {
		PrintSetEnvInfo(name, value, envInfo, con)
	}

}

// PrintSetEnvInfo - Print the set environment info
func PrintSetEnvInfo(name string, value string, envInfo *sliverpb.SetEnv, con *console.SliverConsoleClient) {
	if envInfo.Response != nil && envInfo.Response.Err != "" {
		con.PrintErrorf("%s\n", envInfo.Response.Err)
		return
	}
	con.PrintInfof("Set %s to %s\n", name, value)
}
