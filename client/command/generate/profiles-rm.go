package generate

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// ProfilesRmCmd - Delete an implant profile
func ProfilesRmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	_, err := con.Rpc.DeleteImplantProfile(context.Background(), &clientpb.DeleteReq{
		Name: ctx.Args.String("profile-name"),
	})
	if err != nil {
		con.PrintErrorf("Failed to delete profile %s\n", err)
		return
	}
}
