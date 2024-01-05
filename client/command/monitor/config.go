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

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func MonitorConfigCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {

	resp, err := con.Rpc.MonitorListConfig(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintWTConfig(resp, con)
}

func MonitorAddConfigCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {

	apiKey, _ := cmd.Flags().GetString("apiKey")
	apiPassword, _ := cmd.Flags().GetString("apiPassword")
	apiType, _ := cmd.Flags().GetString("type")

	MonitoringProvider := &clientpb.MonitoringProvider{Type: apiType, APIKey: apiKey}

	if apiType == "xforce" {
		MonitoringProvider.APIPassword = apiPassword
	}

	resp, err := con.Rpc.MonitorAddConfig(context.Background(), MonitoringProvider)
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

func MonitorDelConfigCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {

	resp, err := con.Rpc.MonitorListConfig(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	config, err := selectWatchtowerConfig(resp)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	_, err = con.Rpc.MonitorDelConfig(context.Background(), config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Removed monitoring configuration\n")
}

// PrintWTConfig - Print the current watchtower configuration
func PrintWTConfig(configs *clientpb.MonitoringProviders, con *console.SliverClient) {
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

func selectWatchtowerConfig(configs *clientpb.MonitoringProviders) (*clientpb.MonitoringProvider, error) {

	var options []string
	for _, config := range configs.Providers {
		options = append(options, config.Type)
	}

	selected := ""
	prompt := &survey.Select{
		Message: "Select a configuration:",
		Options: options,
	}
	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return nil, err
	}

	for _, provider := range configs.Providers {
		if provider.Type == selected {
			return provider, nil
		}
	}
	return nil, nil
}
