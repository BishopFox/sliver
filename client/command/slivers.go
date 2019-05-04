package command

import (
	"fmt"
	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func listSliverBuilds(ctx *grumble.Context, rpc RPCServer) {

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgListSliverBuilds,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}

	builds := &clientpb.SliverBuilds{}
	proto.Unmarshal(resp.Data, builds)
	index := 1
	for name := range builds.Configs {
		fmt.Printf("%d. %s\n", index, name)
		index++
	}
}

func listCanaries(ctx *grumble.Context, rpc RPCServer) {

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgListCanaries,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"%s\n", resp.Err)
		return
	}

	canaries := &clientpb.Canaries{}
	proto.Unmarshal(resp.Data, canaries)

}
