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
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// ImplantBuildFilter - Filter implant builds.
type ImplantBuildFilter struct {
	GOOS    string
	GOARCH  string
	Beacon  bool
	Session bool
	Format  string
	Debug   bool
}

// ImplantsCmd - Displays archived implant builds.
func ImplantsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	implantBuildFilters := ImplantBuildFilter{}

	if 0 < len(builds.Configs) {
		PrintImplantBuilds(builds, implantBuildFilters, con)
	} else {
		con.PrintInfof("No implant builds\n")
	}
}

// PrintImplantBuilds - Print the implant builds on the server
func PrintImplantBuilds(builds *clientpb.ImplantBuilds, filters ImplantBuildFilter, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Name",
		"Implant Type",
		"Template",
		"OS/Arch",
		"Format",
		"Command & Control",
		"Debug",
		"C2 Config",
		"ID",
		"Stage",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})

	for sliverName, config := range builds.Configs {
		if filters.GOOS != "" && config.GOOS != filters.GOOS {
			continue
		}
		if filters.GOARCH != "" && config.GOARCH != filters.GOARCH {
			continue
		}
		if filters.Beacon && !config.IsBeacon {
			continue
		}
		if filters.Session && config.IsBeacon {
			continue
		}
		if filters.Debug && config.Debug {
			continue
		}
		if filters.Format != "" && (!strings.EqualFold(config.Format.String(), filters.Format) ||
			!strings.HasPrefix(strings.ToLower(config.Format.String()), strings.ToLower(filters.Format))) {
			continue
		}

		implantType := ""
		if builds.ResourceIDs[sliverName].Type != "" {
			implantType = builds.ResourceIDs[sliverName].Type
		}

		if config.IsBeacon {
			implantType += "beacon"
		} else {
			implantType += "session"
		}
		c2URLs := []string{}
		for index, c2 := range config.C2 {
			c2URLs = append(c2URLs, fmt.Sprintf("[%d] %s", index+1, c2.URL))
		}
		if config.TemplateName == "" {
			config.TemplateName = "sliver"
		}
		tw.AppendRow(table.Row{
			sliverName,
			implantType,
			config.TemplateName,
			fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
			config.Format,
			strings.Join(c2URLs, "\n"),
			fmt.Sprintf("%v", config.Debug),
			fmt.Sprintf(config.HTTPC2ConfigName),
			fmt.Sprintf("%v", builds.ResourceIDs[sliverName].Value),
			fmt.Sprintf("%v", builds.Staged[sliverName]),
		})
	}

	con.Println(tw.Render())
	con.Println("\n")
}

// ImplantBuildNameCompleter - Completer for implant build names.
func ImplantBuildNameCompleter(con *console.SliverClient) carapace.Action {
	comps := func(ctx carapace.Context) carapace.Action {
		var action carapace.Action

		builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to get implant builds: %s", err.Error())
		}

		filters := &ImplantBuildFilter{}

		results := []string{}
		sessions := []string{}

		for name, config := range builds.Configs {
			if filters.GOOS != "" && config.GOOS != filters.GOOS {
				continue
			}
			if filters.GOARCH != "" && config.GOARCH != filters.GOARCH {
				continue
			}
			if filters.Beacon && !config.IsBeacon {
				continue
			}
			if filters.Session && config.IsBeacon {
				continue
			}
			if filters.Debug && config.Debug {
				continue
			}
			if filters.Format != "" && !strings.EqualFold(config.Format.String(), filters.Format) {
				continue
			}

			osArch := fmt.Sprintf("[%s/%s]", config.GOOS, config.GOARCH)
			buildFormat := config.Format.String()

			profileType := ""
			if config.IsBeacon {
				profileType = "(B)"
			} else {
				profileType = "(S)"
			}

			var domains []string
			for _, c2 := range config.C2 {
				domains = append(domains, c2.GetURL())
			}

			desc := fmt.Sprintf("%s %s %s %s", profileType, osArch, buildFormat, strings.Join(domains, ","))

			if config.IsBeacon {
				results = append(results, name)
				results = append(results, desc)
			} else {
				sessions = append(sessions, name)
				sessions = append(sessions, desc)
			}
		}

		return action.Invoke(ctx).Merge(
			carapace.ActionValuesDescribed(sessions...).Tag("session builds").Invoke(ctx),
			carapace.ActionValuesDescribed(results...).Tag("beacon builds").Invoke(ctx),
		).ToA()
	}

	return carapace.ActionCallback(comps)
}

// ImplantBuildByName - Get an implant build by name.
func ImplantBuildByName(name string, con *console.SliverClient) *clientpb.ImplantConfig {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil
	}
	for sliverName, build := range builds.Configs {
		if sliverName == name {
			return build
		}
	}
	return nil
}
