package command

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
)

func monitorStartCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	resp, err := rpc.MonitorStart(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s", err)
		return
	}
	if resp != nil && resp.Err != "" {
		fmt.Printf(Warn+"%s", resp.Err)
		return
	}
	fmt.Printf(Info + "Started monitoring threat intel platforms for implants hashes")
}

func monitorStopCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	_, err := rpc.MonitorStop(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s", err)
		return
	}
	fmt.Printf(Info + "Stopped monitoring threat intel platforms for implants hashes")
}
