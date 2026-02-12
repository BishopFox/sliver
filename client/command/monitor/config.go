package monitor

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
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
// PrintWTConfig - Print 当前瞭望塔配置
func PrintWTConfig(configs *clientpb.MonitoringProviders, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))

	tw.AppendHeader(table.Row{
		"ID",
		"Type",
		"APIKey",
		"APIPassword",
	})

	for _, config := range configs.Providers {
		row := table.Row{}
		row = append(row, config.ID)
		row = append(row, config.Type)
		row = append(row, config.APIKey)
		row = append(row, config.APIPassword)
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
	err := forms.Select("Select a configuration:", options, &selected)
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
