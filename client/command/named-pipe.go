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

	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func namedPipeListener(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if ActiveSliver.Sliver.OS != "windows" {
		fmt.Printf(Warn + "Not Implemented\n")
		return
	}

	pipeName := ctx.Flags.String("name")

	if pipeName == "" {
		fmt.Printf(Warn + "-n parameter missing\n")
		return
	}
	namedPipe := &sliverpb.NamedPipesReq{
		SliverID: ActiveSliver.Sliver.ID,
		PipeName: pipeName,
	}
	data, _ := proto.Marshal(namedPipe)
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgNamedPipesReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	namedPipes := &sliverpb.NamedPipes{}
	err := proto.Unmarshal(resp.Data, namedPipes)
	if err != nil {
		fmt.Printf(Warn + "Failed to decode response\n")
		return
	}
	if namedPipes.Err != "" {
		fmt.Printf(Warn+"Error: %s", namedPipes.Err)
	} else {
		fmt.Printf(Info+"Listening on %s", "\\\\.\\pipe\\"+namedPipe.GetPipeName())
	}
}
