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
	"strconv"
	"strings"

	"github.com/desertbit/grumble"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func IfconfigCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	ifconfig, err := con.Rpc.Ifconfig(context.Background(), &sliverpb.IfconfigReq{
		Request: con.ActiveSession.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	for ifaceIndex, iface := range ifconfig.NetInterfaces {
		fmt.Printf("%s%s%s (%d)\n", console.Bold, iface.Name, console.Normal, ifaceIndex)
		if 0 < len(iface.MAC) {
			fmt.Printf("   MAC Address: %s\n", iface.MAC)
		}
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
				fmt.Printf(console.Bold+console.Green+"    IP Address: %s%s\n", ip, console.Normal)
			} else if 32 < subnet && !isLoopback(ip) {
				fmt.Printf(console.Bold+console.Cyan+"    IP Address: %s%s\n", ip, console.Normal)
			} else {
				fmt.Printf("    IP Address: %s\n", ip)
			}
		}
	}
}

func isLoopback(ip string) bool {
	if strings.HasPrefix(ip, "127") || strings.HasPrefix(ip, "::1") {
		return true
	}
	return false
}
