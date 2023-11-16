package generate

import (
	"context"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// ImplantsStageCmd - Serve a previously generated build
func ImplantsStageCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Unable to load implant builds '%s'\n", err)
		return
	}

	options := []string{}
	for name, _ := range builds.Configs {
		options = append(options, name)
	}

	prompt := &survey.MultiSelect{
		Message: "Select sessions and beacons to expose:",
		Options: options,
	}
	selected := []string{}
	survey.AskOne(prompt, &selected)

	_, err = con.Rpc.StageImplantBuild(context.Background(), &clientpb.ImplantStageReq{Build: selected})
	if err != nil {
		con.PrintErrorf("Failed to serve implant %s\n", err)
		return
	}
}
