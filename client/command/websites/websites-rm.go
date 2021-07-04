package websites

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// WebsiteRmCmd - Remove a website and all its static content
func WebsiteRmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	_, err := con.Rpc.WebsiteRemove(context.Background(), &clientpb.Website{
		Name: ctx.Args.String("name"),
	})
	if err != nil {
		con.PrintErrorf("Failed to remove website %s", err)
		return
	}
}
