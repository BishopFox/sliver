package monitor

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

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

func MonitorConfigCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {

	resp, err := con.Rpc.MonitorListConfig(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintWTConfig(resp, con)
}

func MonitorAddConfigCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {

	apiKey := ctx.Flags.String("apiKey")
	apiPassword := ctx.Flags.String("apiPassword")
	apiType := ctx.Flags.String("type")

	MonitoringProvider := &clientpb.MonitoringProvider{Type: apiType, APIKey: apiKey}

	if apiType == "xforce" {
		MonitoringProvider.APIPassword = apiPassword
	}

	resp, err := con.Rpc.MonitorAddConfig(context.Background(), &clientpb.MonitoringProvider{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if resp != nil && resp.Err != "" {
		con.PrintErrorf("%s\n", resp.Err)
		return
	}
	con.PrintInfof("Added monitoring configuration\n")
}

func MonitorDelConfigCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {

	apiKey := ctx.Flags.String("apiKey")
	apiPassword := ctx.Flags.String("apiPassword")
	apiType := ctx.Flags.String("type")

	MonitoringProvider := &clientpb.MonitoringProvider{Type: apiType, APIKey: apiKey}

	if apiType == "xforce" {
		MonitoringProvider.APIPassword = apiPassword
	}

	resp, err := con.Rpc.MonitorDelConfig(context.Background(), &clientpb.MonitoringProvider{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if resp != nil && resp.Err != "" {
		con.PrintErrorf("%s\n", resp.Err)
		return
	}
	con.PrintInfof("Added monitoring configuration\n")
}

// PrintWTConfig - Print the current watchtower configuration
func PrintWTConfig(configs *clientpb.MonitoringProviders, con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))

	tw.AppendHeader(table.Row{
		"ID",
		"Type",
		"APIKey",
		"APIPassword",
	})

	color := console.Normal

	for _, config := range configs.Providers {
		row := table.Row{}
		row = append(row, fmt.Sprintf(color+"%s"+console.Normal, config.ID))
		row = append(row, fmt.Sprintf(color+"%s"+console.Normal, config.Type))
		row = append(row, fmt.Sprintf(color+"%s"+console.Normal, config.APIKey))
		row = append(row, fmt.Sprintf(color+"%s"+console.Normal, config.APIPassword))
		tw.AppendRow(row)
	}

	con.Printf("%s\n", tw.Render())
}
