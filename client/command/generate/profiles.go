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
	"strings"

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
		con.PrintInfof("No profiles, see `%s %s help`\n", consts.ProfilesStr, consts.NewStr)
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
	tw.SortBy([]table.SortBy{
		{Name: "Profile Name", Mode: table.Asc},
	})

	for _, profile := range profiles {
		config := profile.Config

		obfuscation := "disabled"
		if config.ObfuscateSymbols {
			obfuscation = "enabled"
		}
		implantType := "session"
		if config.IsBeacon {
			implantType = "beacon"
		}
		c2URLs := []string{}
		for index, c2 := range config.C2 {
			c2URLs = append(c2URLs, fmt.Sprintf("[%d] %s", index+1, c2.URL))
		}
		tw.AppendRow(table.Row{
			profile.Name,
			implantType,
			fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
			strings.Join(c2URLs, "\n"),
			fmt.Sprintf("%v", config.Debug),
			fmt.Sprintf("%v", config.Format),
			obfuscation,
			getLimitsString(config),
		})
	}

	con.Printf("%s\n", tw.Render())
}

func getImplantProfiles(con *console.SliverConsoleClient) []*clientpb.ImplantProfile {
	pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	return pbProfiles.Profiles
}

// GetImplantProfileByName - Get an implant profile by a specific name
func GetImplantProfileByName(name string, con *console.SliverConsoleClient) *clientpb.ImplantProfile {
	pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	for _, profile := range pbProfiles.Profiles {
		if profile.Name == name {
			return profile
		}
	}
	return nil
}

// ProfileNameCompleter - Completer for implant build names
func ProfileNameCompleter(prefix string, args []string, con *console.SliverConsoleClient) []string {
	pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		return []string{}
	}
	results := []string{}
	for _, profile := range pbProfiles.Profiles {
		if strings.HasPrefix(profile.Name, prefix) {
			results = append(results, profile.Name)
		}
	}
	return results
}
