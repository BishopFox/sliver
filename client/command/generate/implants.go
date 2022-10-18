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
	"sort"
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ImplantBuildFilter - Filter implant builds
type ImplantBuildFilter struct {
	GOOS    string
	GOARCH  string
	Beacon  bool
	Session bool
	Format  string
	Debug   bool
}

// ImplantsCmd - Displays archived implant builds
func ImplantsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	implantBuildFilters := ImplantBuildFilter{}

	if 0 < len(builds.Configs) {
		PrintImplantBuilds(builds.Configs, implantBuildFilters, con)
	} else {
		con.PrintInfof("No implant builds\n")
	}
}

// PrintImplantBuilds - Print the implant builds on the server
func PrintImplantBuilds(configs map[string]*clientpb.ImplantConfig, filters ImplantBuildFilter, con *console.SliverConsoleClient) {
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
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})

	for sliverName, config := range configs {
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

		implantType := "session"
		if config.IsBeacon {
			implantType = "beacon"
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
		})
	}

	con.Println(tw.Render())
}

// ImplantBuildNameCompleter - Completer for implant build names
func ImplantBuildNameCompleter(prefix string, args []string, filters ImplantBuildFilter, con *console.SliverConsoleClient) []string {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		return []string{}
	}
	results := []string{}
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

		if strings.HasPrefix(name, prefix) {
			results = append(results, name)
		}
	}
	sort.StringSlice(results).Sort()
	return results
}

// ImplantBuildByName - Get an implant build by name
func ImplantBuildByName(name string, con *console.SliverConsoleClient) *clientpb.ImplantConfig {
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
