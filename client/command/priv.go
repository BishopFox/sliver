package command

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func runAs(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	username := ctx.Flags.String("username")
	process := ctx.Flags.String("process")
	arguments := ctx.Flags.String("args")

	if username == "" {
		fmt.Printf(Warn + "please specify a username\n")
		return
	}

	if process == "" {
		fmt.Printf(Warn + "please specify a process path\n")
		return
	}

	runAsResp, err := rpc.RunAs(context.Background(), &sliverpb.RunAsReq{
		Request:     ActiveSession.Request(ctx),
		Username:    username,
		ProcessName: process,
		Args:        arguments,
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if runAsResp.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", runAsResp.GetResponse().GetErr())
		return
	}

	fmt.Printf(Info+"Sucessfully ran %s %s on %s\n", process, arguments, session.GetName())
}

func impersonate(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	if len(ctx.Args) != 1 {
		fmt.Printf(Warn + "You must provide a username. See `help impersonate`\n")
		return
	}
	username := ctx.Args[0]
	impResp, err := rpc.Impersonate(context.Background(), &sliverpb.ImpersonateReq{
		Request:  ActiveSession.Request(ctx),
		Username: username,
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if impResp.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", impResp.GetResponse().GetErr())
		return
	}
	fmt.Printf(Info+"Successfully impersonated %s\n", username)
}

func revToSelf(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	_, err := rpc.RevToSelf(context.Background(), &sliverpb.RevToSelfReq{
		Request: ActiveSession.Request(ctx),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	fmt.Printf(Info + "Back to self...")
}

func getsystem(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		return
	}
	process := ctx.Flags.String("process")
	config := getActiveSliverConfig()
	ctrl := make(chan bool)
	go spin.Until("Attempting to create a new sliver session as 'NT AUTHORITY\\SYSTEM'...", ctrl)

	getsystemResp, err := rpc.GetSystem(context.Background(), &clientpb.GetSystemReq{
		Request:        ActiveSession.Request(ctx),
		Config:         config,
		HostingProcess: process,
	})

	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}
	if getsystemResp.GetResponse().GetErr() != "" {
		fmt.Printf(Warn+"Error: %s\n", getsystemResp.GetResponse().GetErr())
		return
	}
	fmt.Printf("\n" + Info + "A new SYSTEM session should pop soon...\n")
}
