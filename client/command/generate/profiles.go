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
	"fmt"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ProfilesCmd - Display implant profiles
func ProfilesCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	profiles := getImplantProfiles(con)
	if profiles == nil {
		return
	}
	if len(profiles) == 0 {
		con.PrintInfof("No profiles, create one with `%s`\n", consts.NewStr)
		return
	} else {
		PrintProfiles(profiles, con)
	}
}

// PrintProfiles - Print the profiles
func PrintProfiles(profiles []*clientpb.ImplantProfile, con *console.SliverConsoleClient) {

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Profile Name",
		"Implant Type",
		"Platform",
		"Command & Control",
		"Debug",
		"Format",
		"Obfuscation",
		"Limitations",
	})

	for _, profile := range profiles {
		config := profile.Config
		if 0 < len(config.C2) {
			obfuscation := "disabled"
			if config.ObfuscateSymbols {
				obfuscation = "enabled"
			}
			implantType := "session"
			if config.IsBeacon {
				implantType = "beacon"
			}
			tw.AppendRow(table.Row{
				profile.Name,
				implantType,
				fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
				fmt.Sprintf("[1] %s", config.C2[0].URL),
				fmt.Sprintf("%v", config.Debug),
				fmt.Sprintf("%v", config.Format),
				obfuscation,
				getLimitsString(config),
			})
		}
		if 1 < len(config.C2) {
			for index, c2 := range config.C2[1:] {
				tw.AppendRow(table.Row{
					"",
					"",
					"",
					fmt.Sprintf("[%d] %s", index+2, c2.URL),
					"",
					"",
					"",
					"",
				})
			}
		}
		tw.AppendRow(table.Row{
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
		})
	}

	con.Printf("%s\n", tw.Render())
}

func getImplantProfiles(con *console.SliverConsoleClient) []*clientpb.ImplantProfile {
	pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Error %s", err)
		return nil
	}
	return pbProfiles.Profiles
}

func GetImplantProfileByName(name string, con *console.SliverConsoleClient) *clientpb.ImplantProfile {
	pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Error %s", err)
		return nil
	}
	for _, profile := range pbProfiles.Profiles {
		if profile.Name == name {
			return profile
		}
	}
	return nil
}
