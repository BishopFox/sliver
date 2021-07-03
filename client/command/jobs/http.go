package jobs

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

func HTTPListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	domain := ctx.Flags.String("domain")
	lport := uint16(ctx.Flags.Int("lport"))

	con.PrintInfof("Starting HTTP %s:%d listener ...\n", domain, lport)
	http, err := con.Rpc.StartHTTPListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:     domain,
		Website:    ctx.Flags.String("website"),
		Port:       uint32(lport),
		Secure:     false,
		Persistent: ctx.Flags.Bool("persistent"),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", http.JobID)
	}
}
