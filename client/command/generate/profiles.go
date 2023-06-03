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

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// ProfilesCmd - Display implant profiles
func ProfilesCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
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
func ProfileNameCompleter(con *console.SliverConsoleClient) carapace.Action {
	comps := func(ctx carapace.Context) carapace.Action {
		var action carapace.Action

		pbProfiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage(fmt.Sprintf("No profiles, err: %s", err.Error()))
		}

		if len(pbProfiles.Profiles) == 0 {
			return carapace.ActionMessage("No saved implant profiles")
		}

		results := []string{}
		sessions := []string{}

		for _, profile := range pbProfiles.Profiles {

			osArch := fmt.Sprintf("[%s/%s]", profile.Config.GOOS, profile.Config.GOARCH)
			buildFormat := profile.Config.Format.String()

			profileType := ""
			if profile.Config.IsBeacon {
				profileType = "(B)"
			} else {
				profileType = "(S)"
			}

			var domains []string
			for _, c2 := range profile.Config.C2 {
				domains = append(domains, c2.GetURL())
			}

			desc := fmt.Sprintf("%s %s %s %s", profileType, osArch, buildFormat, strings.Join(domains, ","))

			if profile.Config.IsBeacon {
				results = append(results, profile.Name)
				results = append(results, desc)
			} else {
				sessions = append(sessions, profile.Name)
				sessions = append(sessions, desc)
			}
		}

		return action.Invoke(ctx).Merge(
			carapace.ActionValuesDescribed(sessions...).Tag("sessions").Invoke(ctx),
			carapace.ActionValuesDescribed(results...).Tag("beacons").Invoke(ctx),
		).ToA()
	}

	return carapace.ActionCallback(comps)
}
