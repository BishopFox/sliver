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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// Execute - Run a command on the remote system
func ExecuteCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	cmdPath := ctx.Args.String("command")
	args := ctx.Args.StringList("arguments")
	output := ctx.Flags.Bool("silent")
	token := ctx.Flags.Bool("token")
	var exec *sliverpb.Execute
	var err error
	if token {
		exec, err = con.Rpc.ExecuteToken(context.Background(), &sliverpb.ExecuteTokenReq{
			Request: con.ActiveSession.Request(ctx),
			Path:    cmdPath,
			Args:    args,
			Output:  !output,
		})
	} else {
		exec, err = con.Rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
			Request: con.ActiveSession.Request(ctx),
			Path:    cmdPath,
			Args:    args,
			Output:  !output,
		})
	}

	if err != nil {
		con.PrintErrorf("%s", err)
	} else if !output {
		if exec.Status != 0 {
			con.PrintErrorf("Exited with status %d!\n", exec.Status)
			if exec.Result != "" {
				con.PrintInfof("Output:\n%s\n", exec.Result)
			}
		} else {
			con.PrintInfof("Output:\n%s\n", exec.Result)
			if ctx.Flags.Bool("loot") && 0 < len(exec.Result) {
				name := fmt.Sprintf("[exec] %s %s", cmdPath, strings.Join(args, " "))
				err = loot.AddLootFile(con.Rpc, name, "console.txt", []byte(exec.Result), false)
				if err != nil {
					con.PrintErrorf("Failed to save output as loot: %s\n", err)
				} else {
					con.PrintInfof("Output saved as loot\n")
				}
			}
		}
	}
}
