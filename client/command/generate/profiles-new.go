package generate

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

func ProfilesNewCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	name := ctx.Flags.String("profile-name")
	if name == "" {
		con.PrintErrorf("Invalid profile name\n")
		return
	}
	config := parseCompileFlags(ctx, con)
	if config == nil {
		return
	}
	profile := &clientpb.ImplantProfile{
		Name:   name,
		Config: config,
	}
	resp, err := con.Rpc.SaveImplantProfile(context.Background(), profile)
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Saved new profile %s\n", resp.Name)
	}
}
