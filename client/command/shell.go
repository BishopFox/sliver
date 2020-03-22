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

// import (
// 	"fmt"
// 	"io"
// 	"log"
// 	"os"

// 	"github.com/bishopfox/sliver/client/core"
// 	"github.com/bishopfox/sliver/protobuf/rpcpb"
// 	"github.com/bishopfox/sliver/protobuf/sliverpb"

// 	"github.com/desertbit/grumble"
// 	"github.com/golang/protobuf/proto"
// 	"golang.org/x/crypto/ssh/terminal"
// )

// const (
// 	windows = "windows"
// )

// func shell(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
// 	session := ActiveSession.Get()
// 	if session == nil {
// 		return
// 	}

// 	if !isUserAnAdult() {
// 		return
// 	}

// 	shellPath := ctx.Flags.String("shell-path")
// 	noPty := ctx.Flags.Bool("no-pty")
// 	if ActiveSliver.Sliver.OS == windows {
// 		noPty = true // Windows of course doesn't have PTYs
// 	}
// 	runInteractive(shellPath, noPty, server)
// }

// func runInteractive(shellPath string, noPty bool, rpc rpcpb.SliverRPCClient) {
// 	fmt.Printf(Info + "Opening shell tunnel (EOF to exit) ...\n\n")

// 	tunnelInfo, err := rpc.CreateTunnel(context.Background(), &clientpb.CreateTunnelReq{
// 		Request: ActiveSession.Request(),
// 	})
// 	if err != nil {
// 		fmt.Printf(Warn+"%s\n", err)
// 		return
// 	}

// 	shell, err := rpc.Shell(context.Background(), &sliverpb.ShellReq{
// 		Request:   ActiveSession.Request(),
// 		EnablePTY: !noPty,
// 		TunnelID:  tunnelInfo.ID,
// 		Path:      shellPath,
// 	})
// 	if err != nil {
// 		fmt.Printf(Warn+"%s\n", err)
// 		return
// 	}

// 	rpcTunnel, err := rpc.Tunnel(context.Background())
// 	if err != nil {
// 		fmt.Printf(Warn+"%s", err)
// 		return
// 	}

// 	var oldState *terminal.State
// 	if !noPty {
// 		oldState, err = terminal.MakeRaw(0)
// 		log.Printf("Saving terminal state: %v", oldState)
// 		if err != nil {
// 			fmt.Printf(Warn + "Failed to save terminal state")
// 			return
// 		}
// 	}

// 	go func() {
// 		_, err := io.Copy(os.Stdout, tunnel)
// 		if err != nil {
// 			fmt.Printf(Warn+"Error writing to stdout: %v", err)
// 			return
// 		}
// 	}()
// 	for {
// 		_, err := io.Copy(tunnel, os.Stdin)
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			fmt.Printf(Warn+"Error reading from stdin: %v", err)
// 			break
// 		}
// 	}
// 	if !noPty {
// 		terminal.Restore(0, oldState)
// 	}
// }
