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
	"fmt"
	"strings"

	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func execute(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
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
	data, _ := proto.Marshal(&sliverpb.ExecuteReq{
		SliverID: ActiveSliver.Sliver.ID,
		Path:     cmdPath,
		Args:     strings.Split(args, " "),
		Output:   output,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgExecuteReq,
		Data: data,
	}, defaultTimeout)

	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	execResp := &sliverpb.Execute{}
	err := proto.Unmarshal(resp.Data, execResp)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}
	if execResp.Error != "" {
		fmt.Printf(Warn+"Error: %s\n", execResp.Error)
		return
	}
	if output {
		fmt.Printf(Info+"Output:\n%s", execResp.Result)
	}
}
