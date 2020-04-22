package command

import (
	"fmt"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func persist(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	if ActiveSliver.Sliver.OS == "windows" {
		fmt.Printf(Warn + "Not Implemented\n")
		return
	}
	if !isUserAnAdult() {
		return
	}
	data, _ := proto.Marshal(&sliverpb.PersistReq{
		SliverID: ActiveSliver.Sliver.ID,
		Filename: ActiveSliver.Sliver.Filename,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgPersistReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
	}
	persistConfig := &sliverpb.Persist{}
	err := proto.Unmarshal(resp.Data, persistConfig)
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
	}
}
