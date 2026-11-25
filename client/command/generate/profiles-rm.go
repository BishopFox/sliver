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
	"fmt"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	ctx, cancel := con.GrpcContext(cmd)
	defer cancel()
	if _, err := GetProfileDefinition(ctx, name, con); err == nil {
		confirm := false
		_ = forms.Confirm(fmt.Sprintf("Remove profile definition %q? Completed builds remain available.", name), &confirm)
		if !confirm {
			return
		}
		if _, err := con.Rpc.DeleteBuildProfile(ctx, &clientpb.BuildProfileRef{Name: name}); err != nil {
			con.PrintErrorf("Failed to delete profile %s: %s\n", name, err)
			return
		}
		con.PrintInfof("Removed profile %s; completed builds remain available\n", name)
		return
	} else if status.Code(err) != codes.Unimplemented {
		con.PrintErrorf("Get profile %s: %s\n", name, err)
		return
	}
	profile := GetImplantProfileByName(name, con)
	if profile == nil {
		con.PrintErrorf("No profile found with name '%s'\n", name)
		return
	}
	confirm := false
	_ = forms.Confirm(fmt.Sprintf("Remove '%s' profile?", name), &confirm)
	if !confirm {
		return
	}
	_, err := con.Rpc.DeleteImplantProfile(ctx, &clientpb.DeleteReq{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("Failed to delete profile %s\n", err)
		return
	}
}
