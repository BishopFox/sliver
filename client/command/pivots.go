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
	"context"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
)

func namedPipeListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if session.OS != "windows" {
		fmt.Printf(Warn+"Not implemented for %s\n", session.OS)
		return
	}

	pipeName := ctx.Flags.String("name")

	if pipeName == "" {
		fmt.Printf(Warn + "-n parameter missing\n")
		return
	}

	_, err := rpc.NamedPipes(context.Background(), &sliverpb.NamedPipesReq{
		PipeName: pipeName,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	fmt.Printf(Info+"Listening on %s", "\\\\.\\pipe\\"+pipeName)
}

func tcpListener(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	server := ctx.Flags.String("server")
	lport := uint16(ctx.Flags.Int("lport"))
	address := fmt.Sprintf("%s:%d", server, lport)

	_, err := rpc.TCPListener(context.Background(), &sliverpb.TCPPivotReq{
		Address: address,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	fmt.Printf(Info+"Listening on tcp://%s", address)
}