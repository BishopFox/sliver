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
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// SpawnDllCmd - Spawn execution of a DLL on the remote system
func SpawnDllCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	dllArgs := strings.Join(ctx.Args.StringList("arguments"), " ")
	binPath := ctx.Args.String("filepath")
	processName := ctx.Flags.String("process")
	exportName := ctx.Flags.String("export")

	binData, err := os.ReadFile(binPath)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Executing reflective dll %s", binPath), ctrl)
	spawndll, err := con.Rpc.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
		Data:        binData,
		ProcessName: processName,
		Args:        dllArgs,
		EntryPoint:  exportName,
		Request:     con.ActiveTarget.Request(ctx),
		Kill:        !ctx.Flags.Bool("keep-alive"),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	ctrl <- true
	<-ctrl

	hostName := getHostname(session, beacon)
	if spawndll.Response != nil && spawndll.Response.Async {
		con.AddBeaconCallback(spawndll.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, spawndll)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}

			HandleSpawnDLLResponse(spawndll, binPath, hostName, ctx, con)
		})
		con.PrintAsyncResponse(spawndll.Response)
	} else {
		HandleSpawnDLLResponse(spawndll, binPath, hostName, ctx, con)
	}
}

func HandleSpawnDLLResponse(spawndll *sliverpb.SpawnDll, binPath string, hostName string, ctx *grumble.Context, con *console.SliverConsoleClient) {
	saveLoot := ctx.Flags.Bool("loot")
	lootName := ctx.Flags.String("name")

	if spawndll.GetResponse().GetErr() != "" {
		con.PrintErrorf("Failed to spawn dll: %s\n", spawndll.GetResponse().GetErr())
		return
	}

	PrintExecutionOutput(spawndll.GetResult(), ctx.Flags.Bool("save"), ctx.Command.Name, hostName, con)

	if saveLoot {
		LootExecute([]byte(spawndll.GetResult()), lootName, ctx.Command.Name, binPath, hostName, con)
	}
}
