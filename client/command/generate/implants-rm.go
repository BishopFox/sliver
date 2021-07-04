package generate

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// ImplantsRmCmd - Deletes an archived implant build from the server
func ImplantsRmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	_, err := con.Rpc.DeleteImplantBuild(context.Background(), &clientpb.DeleteReq{
		Name: ctx.Args.String("implant-name"),
	})
	if err != nil {
		con.PrintErrorf("Failed to delete implant %s\n", err)
		return
	}
}
