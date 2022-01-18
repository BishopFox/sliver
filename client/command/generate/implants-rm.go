package generate

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// ImplantsRmCmd - Deletes an archived implant build from the server
func ImplantsRmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	name := ctx.Args.String("name")
	if name == "" {
		con.PrintErrorf("No name specified\n")
		return
	}
	build := ImplantBuildByName(name, con)
	if build == nil {
		con.PrintErrorf("No implant build found with name '%s'\n", name)
		return
	}
	confirm := false
	prompt := &survey.Confirm{Message: fmt.Sprintf("Remove '%s' build?", name)}
	survey.AskOne(prompt, &confirm)
	if !confirm {
		return
	}
	_, err := con.Rpc.DeleteImplantBuild(context.Background(), &clientpb.DeleteReq{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("Failed to delete implant %s\n", err)
		return
	}
}
