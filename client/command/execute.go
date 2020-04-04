package command

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

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func execute(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "Please provide a path. See `help execute` for more info.\n")
		return
	}

	cmdPath := ctx.Args[0]
	args := ctx.Flags.String("args")
	if len(args) != 0 {
		args = cmdPath + " " + args
	}
	output := ctx.Flags.Bool("output")
	exec, err := rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
		Request: ActiveSession.Request(ctx),
		Path:    cmdPath,
		Args:    strings.Split(args, " "),
		Output:  output,
	})
	if err != nil {
		fmt.Printf(Warn+"%s", err)
	} else if output {
		fmt.Printf(Info+"Output:\n%s\n", exec.Result)
	}
}
