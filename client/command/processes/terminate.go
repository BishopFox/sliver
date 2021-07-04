package processes

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func TerminateCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	pid := ctx.Args.Uint("pid")
	terminated, err := con.Rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
		Request: con.ActiveSession.Request(ctx),
		Pid:     int32(pid),
		Force:   ctx.Flags.Bool("force"),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Process %d has been terminated\n", terminated.Pid)
	}

}
