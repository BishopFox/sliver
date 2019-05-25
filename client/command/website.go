package command

import (
	"fmt"
	"strings"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/golang/protobuf/proto"

	"github.com/desertbit/grumble"
)

func websites(ctx *grumble.Context, rpc RPCServer) {
	if len(ctx.Args) < 1 {
		listWebsites(ctx, rpc)
		return
	}
	if ctx.Flags.String("website") == "" {
		fmt.Println(Warn + "Subcommand must specify a --website")
		return
	}
	switch strings.ToLower(ctx.Args[0]) {
	case "ls":
		listWebsiteContent(ctx, rpc)
	case "add":
		addWebsiteContent(ctx, rpc)
	case "rm":
		removeWebsiteContent(ctx, rpc)
	default:
		fmt.Println(Warn + "Invalid subcommand, see 'help websites'")
	}
}

func listWebsites(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteList,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

	websites := &clientpb.Websites{}
	proto.Unmarshal(resp.Data, websites)

}

func listWebsiteContent(ctx *grumble.Context, rpc RPCServer) {
	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteList,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

	websites := &clientpb.Websites{}
	proto.Unmarshal(resp.Data, websites)

}

func addWebsiteContent(ctx *grumble.Context, rpc RPCServer) {

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteAddContent,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

}

func removeWebsiteContent(ctx *grumble.Context, rpc RPCServer) {

	resp := <-rpc(&sliverpb.Envelope{
		Type: clientpb.MsgWebsiteRemoveContent,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", resp.Err)
		return
	}

}
