package pivots

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// NamedPipeListenerCmd - Start a named pipe pivot listener on the remote system
func NamedPipeListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	if session.OS != "windows" {
		con.PrintErrorf("Not implemented for %s\n", session.OS)
		return
	}

	pipeName := ctx.Flags.String("name")

	if pipeName == "" {
		con.PrintErrorf("-n parameter missing\n")
		return
	}

	_, err := con.Rpc.NamedPipes(context.Background(), &sliverpb.NamedPipesReq{
		PipeName: pipeName,
		Request:  con.ActiveTarget.Request(ctx),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Listening on %s", "\\\\.\\pipe\\"+pipeName)
}
