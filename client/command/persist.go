package command

import (
	"fmt"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
)

func persist(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	if ctx.Flags.String("filename") == "" {
		fmt.Printf(Warn + "Empty Filename\n")
	}
	if ActiveSliver.Sliver.OS == "windows" {
		fmt.Printf(Warn + "Not Implemented\n")
		return
	}
	if !isUserAnAdult() {
		return
	}
	assembly, err := ioutil.ReadFile(ctx.Flags.String("filename"))
	if err != nil {
		fmt.Printf(Warn+"Error reading file: %s", err)
		return
	}
	data, _ := proto.Marshal(&sliverpb.PersistReq{
		SliverID: ActiveSliver.Sliver.ID,
		Assembly: assembly,
		Minutes:  uint32(ctx.Flags.Int("minutes")),
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgPersistReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
	}
	persistConfig := &sliverpb.Persist{}
	err = proto.Unmarshal(resp.Data, persistConfig)
	if err != nil {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
	}
}
