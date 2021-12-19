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
	"strings"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// ExecuteCmd - Run a command on the remote system
func ExecuteCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	cmdPath := ctx.Args.String("command")
	args := ctx.Args.StringList("arguments")
	token := ctx.Flags.Bool("token")
	output := ctx.Flags.Bool("output")
	stdout := ctx.Flags.String("stdout")
	stderr := ctx.Flags.String("stderr")

	if output && beacon != nil {
		con.PrintWarnf("Using --output in beacon mode, if the command blocks the task will never complete\n\n")
	}

	var exec *sliverpb.Execute
	var err error

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Executing %s %s ...", cmdPath, strings.Join(args, " ")), ctrl)
	if token {
		exec, err = con.Rpc.ExecuteToken(context.Background(), &sliverpb.ExecuteTokenReq{
			Request: con.ActiveTarget.Request(ctx),
			Path:    cmdPath,
			Args:    args,
			Output:  output,
			Stderr:  stderr,
			Stdout:  stdout,
		})
	} else {
		exec, err = con.Rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
			Request: con.ActiveTarget.Request(ctx),
			Path:    cmdPath,
			Args:    args,
			Output:  output,
			Stderr:  stderr,
			Stdout:  stdout,
		})
	}
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	if exec.Response != nil && exec.Response.Async {
		con.AddBeaconCallback(exec.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, exec)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintExecute(exec, ctx, con)
		})
		con.PrintAsyncResponse(exec.Response)
	} else {
		PrintExecute(exec, ctx, con)
	}
}

// PrintExecute - Print the output of an executed command
func PrintExecute(exec *sliverpb.Execute, ctx *grumble.Context, con *console.SliverConsoleClient) {
	ignoreStderr := ctx.Flags.Bool("ignore-stderr")
	saveLoot := ctx.Flags.Bool("loot")
	stdout := ctx.Flags.String("stdout")
	stderr := ctx.Flags.String("stderr")
	cmdPath := ctx.Args.String("command")
	args := ctx.Args.StringList("arguments")

	output := ctx.Flags.Bool("output")
	if !output {
		if exec.Status == 0 {
			con.PrintInfof("Command executed successfully\n")
		} else {
			con.PrintErrorf("Exit code %d\n", exec.Status)
		}
		return
	}

	combined := ""
	if stdout == "" {
		combined = string(exec.Stdout)
		con.PrintInfof("Output:\n%s", combined)
	} else {
		con.PrintInfof("Stdout saved at %s\n", stdout)
	}

	if stderr == "" {
		if !ignoreStderr && 0 < len(exec.Stderr) {
			combined = fmt.Sprintf("%s\nStderr:\n%s", combined, string(exec.Stderr))
			con.PrintInfof("Stderr:\n%s", string(exec.Stderr))
		}
	} else {
		con.PrintInfof("Stderr saved at %s\n", stderr)
	}

	if exec.Status != 0 {
		con.PrintErrorf("Exited with status %d!\n", exec.Status)
	}
	if saveLoot && 0 < len(combined) {
		name := fmt.Sprintf("[exec] %s %s", cmdPath, strings.Join(args, " "))
		err := loot.AddLootFile(con.Rpc, name, "console.txt", []byte(combined), false)
		if err != nil {
			con.PrintErrorf("Failed to save output as loot: %s\n", err)
		} else {
			con.PrintInfof("Output saved as loot\n")
		}
	}
}
