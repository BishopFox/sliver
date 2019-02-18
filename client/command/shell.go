package command

import (
	"fmt"
	"io"
	"log"
	"os"
	consts "sliver/client/constants"
	"sliver/client/core"
	pb "sliver/protobuf/client"
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

	fmt.Printf(Info + "Opening shell channel with sliver ...\n")

	shellReq := &pb.ShellReq{
		SliverID:  ActiveSliver.Sliver.ID,
		EnablePTY: !noPty,
	}
	shellReqData, _ := proto.Marshal(shellReq)
	resp := <-rpc(&pb.Envelope{
		Type: consts.ShellStr,
		Data: shellReqData,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	openedShell := &sliverpb.Shell{}
	proto.Unmarshal(resp.Data, openedShell)

	tunnel := core.Tunnels.Tunnel(openedShell.TunnelID)

	go func() {
		for recvData := range tunnel.Recv {
			shellData := &sliverpb.ShellData{}
			proto.Unmarshal(recvData, shellData)
			log.Printf("[write] stdout shell with tunnel id = %d", shellData.TunnelID)
			os.Stdout.Write(shellData.Stdout)
		}
	}()

	readBuf := make([]byte, 128)
	for {
		n, err := os.Stdin.Read(readBuf)
		if err == io.EOF {
			return
		}
		data, _ := proto.Marshal(&sliverpb.ShellData{
			Stdin: readBuf[:n],
		})
		log.Printf("[read] stdin tunnel %d", openedShell.TunnelID)
		go rpc(tunnel.Send(data), defaultTimeout)
	}
}
