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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
)

// ProfilesGenerateCmd - Generate an implant binary based on a profile
func ProfilesGenerateCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		con.PrintErrorf("No profile name specified\n")
		return
	}
	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	profile := GetImplantProfileByName(name, con)
	if profile != nil {
		disableSGN, _ := cmd.Flags().GetBool("disable-sgn")
		implantFile, err := compile(profile.Config, disableSGN, save, con)
		if err != nil {
			return
		}
		profile.Config.Name = buildImplantName(implantFile.Name)
		_, err = con.Rpc.SaveImplantProfile(context.Background(), profile)
		if err != nil {
			con.PrintErrorf("could not update implant profile: %v\n", err)
			return
		}
	} else {
		con.PrintErrorf("No profile with name '%s'", name)
	}
}

func buildImplantName(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}
