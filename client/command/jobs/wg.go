package jobs

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

func WGListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	lport := uint16(ctx.Flags.Int("lport"))
	nport := uint16(ctx.Flags.Int("nport"))
	keyExchangePort := uint16(ctx.Flags.Int("key-port"))

	con.PrintInfof("Starting Wireguard listener ...\n")
	wg, err := con.Rpc.StartWGListener(context.Background(), &clientpb.WGListenerReq{
		Port:       uint32(lport),
		NPort:      uint32(nport),
		KeyPort:    uint32(keyExchangePort),
		Persistent: ctx.Flags.Bool("persistent"),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", wg.JobID)
	}
}
