package command

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func getEnv(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	var name string
	if len(ctx.Args) > 0 {
		name = ctx.Args[0]
	}

	envInfo, err := rpc.GetEnv(context.Background(), &sliverpb.EnvReq{
		Name:    name,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	for _, envVar := range envInfo.Variables {
		fmt.Printf("%s=%s\n", envVar.Key, envVar.Value)
	}
}
