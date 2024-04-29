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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// ProfilesNewCmd - Create a new implant profile.
func ProfilesNewCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	// name := ctx.Args.String("name")
	_, config := parseCompileFlags(cmd, con)
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
		con.PrintInfof("Saved new implant profile %s\n", resp.Name)
	}
}

// ProfilesNewBeaconCmd - Create a new beacon profile.
func ProfilesNewBeaconCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	// name := ctx.Args.String("name")
	if name == "" {
		con.PrintErrorf("No profile name specified\n")
		return
	}
	_, config := parseCompileFlags(cmd, con)
	if config == nil {
		return
	}
	config.IsBeacon = true
	err := parseBeaconFlags(cmd, config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
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
		con.PrintInfof("Saved new implant profile (beacon) %s\n", resp.Name)
	}
}
