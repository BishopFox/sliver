package jobs

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

func MTLSListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	server := ctx.Flags.String("server")
	lport := uint16(ctx.Flags.Int("lport"))

	con.PrintInfof("Starting mTLS listener ...\n")
	mtls, err := con.Rpc.StartMTLSListener(context.Background(), &clientpb.MTLSListenerReq{
		Host:       server,
		Port:       uint32(lport),
		Persistent: ctx.Flags.Bool("persistent"),
	})
	con.Println()
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", mtls.JobID)
	}
}
