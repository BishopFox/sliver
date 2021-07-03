package jobs

import (
	"context"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

func DNSListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {

	domains := strings.Split(ctx.Flags.String("domains"), ",")
	for _, domain := range domains {
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}
	}

	lport := uint16(ctx.Flags.Int("lport"))

	con.PrintInfof("Starting DNS listener with parent domain(s) %v ...\n", domains)
	dns, err := con.Rpc.StartDNSListener(context.Background(), &clientpb.DNSListenerReq{
		Domains:    domains,
		Port:       uint32(lport),
		Canaries:   !ctx.Flags.Bool("no-canaries"),
		Persistent: ctx.Flags.Bool("persistent"),
	})
	con.Println()
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", dns.JobID)
	}
}
