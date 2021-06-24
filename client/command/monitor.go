package command

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
)

func monitorStartCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	resp, err := rpc.MonitorStart(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s", err)
		return
	}
	if resp != nil && resp.Err != "" {
		fmt.Printf(Warn+"%s", resp.Err)
		return
	}
	fmt.Printf(Info + "Started monitoring threat intel platforms for implants hashes")
}

func monitorStopCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	_, err := rpc.MonitorStop(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"%s", err)
		return
	}
	fmt.Printf(Info + "Stopped monitoring threat intel platforms for implants hashes")
}
