package generate

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// ProfilesRmCmd - Delete an implant profile.
func ProfilesRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	// name := ctx.Args.String("name")
	if name == "" {
		con.PrintErrorf("No profile name specified\n")
		return
	}
	profile := GetImplantProfileByName(name, con)
	if profile == nil {
		con.PrintErrorf("No profile found with name '%s'\n", name)
		return
	}
	confirm := false
	prompt := &survey.Confirm{Message: fmt.Sprintf("Remove '%s' profile?", name)}
	survey.AskOne(prompt, &confirm)
	if !confirm {
		return
	}
	_, err := con.Rpc.DeleteImplantProfile(context.Background(), &clientpb.DeleteReq{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("Failed to delete profile %s\n", err)
		return
	}
}
