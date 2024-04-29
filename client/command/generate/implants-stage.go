package generate

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// ImplantsStageCmd - Serve a previously generated build
func ImplantsStageCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Unable to load implant builds '%s'\n", err)
		return
	}

	options := []string{}
	for name := range builds.Configs {
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
