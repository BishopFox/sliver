package pivots

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func TCPListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	server := ctx.Flags.String("server")
	lport := uint16(ctx.Flags.Int("lport"))
	address := fmt.Sprintf("%s:%d", server, lport)

	_, err := con.Rpc.TCPListener(context.Background(), &sliverpb.TCPPivotReq{
		Address: address,
		Request: con.ActiveSession.Request(ctx),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Listening on tcp://%s", address)
}
