package network

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
	"sort"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// IfconfigCmd - Display network interfaces on the remote system
func IfconfigCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	ifconfig, err := con.Rpc.Ifconfig(context.Background(), &sliverpb.IfconfigReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	all, _ := cmd.Flags().GetBool("all")
	if ifconfig.Response != nil && ifconfig.Response.Async {
		con.AddBeaconCallback(ifconfig.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, ifconfig)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintIfconfig(ifconfig, all, con)
		})
		con.PrintAsyncResponse(ifconfig.Response)
	} else {
		PrintIfconfig(ifconfig, all, con)
	}
}

// PrintIfconfig - Print the ifconfig response
func PrintIfconfig(ifconfig *sliverpb.Ifconfig, all bool, con *console.SliverClient) {
	var err error
	interfaces := ifconfig.NetInterfaces
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Index < interfaces[j].Index
	})
	hidden := 0
	for index, iface := range interfaces {
		tw := table.NewWriter()
		tw.SetStyle(settings.GetTableWithBordersStyle(con))
		tw.SetTitle(fmt.Sprintf(console.Bold+"%s"+console.Normal, iface.Name))
		tw.SetColumnConfigs([]table.ColumnConfig{
			{Name: "#", AutoMerge: true},
			{Name: "IP Address", AutoMerge: true},
			{Name: "MAC Address", AutoMerge: true},
		})
		rowConfig := table.RowConfig{AutoMerge: true}
		tw.AppendHeader(table.Row{"#", "IP Addresses", "MAC Address"}, rowConfig)
		macAddress := ""
		if 0 < len(iface.MAC) {
			macAddress = iface.MAC
		}
		ips := []string{}
		for _, ip := range iface.IPAddresses {
			// Try to find local IPs and colorize them
			subnet := -1
			if strings.Contains(ip, "/") {
				parts := strings.Split(ip, "/")
				subnetStr := parts[len(parts)-1]
				subnet, err = strconv.Atoi(subnetStr)
				if err != nil {
					subnet = -1
				}
			}
			if 0 < subnet && subnet <= 32 && !isLoopback(ip) {
				ips = append(ips, fmt.Sprintf(console.Bold+console.Green+"%s"+console.Normal, ip))
			} else if all {
				ips = append(ips, fmt.Sprintf("%s", ip))
			}
		}
		if !all && len(ips) < 1 {
			hidden++
			continue
		}
		if 0 < len(ips) {
			for _, ip := range ips {
				tw.AppendRow(table.Row{iface.Index, ip, macAddress}, rowConfig)
			}
		} else {
			tw.AppendRow(table.Row{iface.Index, " ", macAddress}, rowConfig)
		}
		con.Printf("%s\n", tw.Render())
		if index+1 < len(interfaces) {
			con.Println()
		}
	}
	if !all {
		con.Printf("%d adapters not shown.\n", hidden)
	}
}

func isLoopback(ip string) bool {
	if strings.HasPrefix(ip, "127") || strings.HasPrefix(ip, "::1") {
		return true
	}
	return false
}
