package command

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/commonpb"
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

func setEnv(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if len(ctx.Args) != 2 {
		fmt.Printf(Warn + "Usage: setenv KEY VALUE\n")
		return
	}

	name := ctx.Args[0]
	value := ctx.Args[1]

	envInfo, err := rpc.SetEnv(context.Background(), &sliverpb.SetEnvReq{
		Variable: &commonpb.EnvVar{
			Key:   name,
			Value: value,
		},
		Request: ActiveSession.Request(ctx),
	})
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if envInfo.Response != nil && envInfo.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s", envInfo.Response.Err)
		return
	}
	fmt.Printf(Info+"set %s to %s\n", name, value)
}

func unsetEnv(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}

	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "Usage: unsetenv KEY\n")
		return
	}

	name := ctx.Args[0]
	if name == "" {
		return
	}
	unsetResp, err := rpc.UnsetEnv(context.Background(), &sliverpb.UnsetEnvReq{
		Name:    name,
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if unsetResp.Response != nil && unsetResp.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", unsetResp.Response.Err)
		return
	}
	fmt.Printf(Info+"Successfully unset %s\n", name)
}
