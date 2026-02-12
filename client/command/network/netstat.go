package network

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
	"fmt"
	"net"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// NetstatCmd - Display active network connections on the remote system
// NetstatCmd - Display 远程系统上的活动网络连接
func NetstatCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	listening, _ := cmd.Flags().GetBool("listen")
	ip4, _ := cmd.Flags().GetBool("ip4")
	ip6, _ := cmd.Flags().GetBool("ip6")
	tcp, _ := cmd.Flags().GetBool("tcp")
	udp, _ := cmd.Flags().GetBool("udp")
	numeric, _ := cmd.Flags().GetBool("numeric")

	implantPID := getPID(session, beacon)
	activeC2 := getActiveC2(session, beacon)

	netstat, err := con.Rpc.Netstat(context.Background(), &sliverpb.NetstatReq{
		Request:   con.ActiveTarget.Request(cmd),
		TCP:       tcp,
		UDP:       udp,
		Listening: listening,
		IP4:       ip4,
		IP6:       ip6,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if netstat.Response != nil && netstat.Response.Async {
		con.AddBeaconCallback(netstat.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, netstat)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintNetstat(netstat, implantPID, activeC2, numeric, con)
		})
		con.PrintAsyncResponse(netstat.Response)
	} else {
		PrintNetstat(netstat, implantPID, activeC2, numeric, con)
	}
}

func PrintNetstat(netstat *sliverpb.Netstat, implantPID int32, activeC2 string, numeric bool, con *console.SliverClient) {
	lookup := func(skaddr *sliverpb.SockTabEntry_SockAddr) string {
		addr := skaddr.Ip
		names, err := net.LookupAddr(addr)
		if err == nil && len(names) > 0 {
			addr = names[0]
		}
		return fmt.Sprintf("%s:%d", addr, skaddr.Port)
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{"Protocol", "Local Address", "Foreign Address", "State", "PID/Program name"})

	for _, entry := range netstat.Entries {
		pid := ""
		if entry.Process != nil {
			pid = fmt.Sprintf("%d/%s", entry.Process.Pid, entry.Process.Executable)
		}
		srcAddr := fmt.Sprintf("%s:%d", entry.LocalAddr.Ip, entry.LocalAddr.Port)
		dstAddr := fmt.Sprintf("%s:%d", entry.RemoteAddr.Ip, entry.RemoteAddr.Port)
		if !numeric {
			srcAddr = lookup(entry.LocalAddr)
			dstAddr = lookup(entry.RemoteAddr)
		}
		if entry.Process != nil && entry.Process.Pid == implantPID {
			tw.AppendRow(table.Row{
				console.StyleGreen.Render(entry.Protocol),
				console.StyleGreen.Render(srcAddr),
				console.StyleGreen.Render(dstAddr),
				console.StyleGreen.Render(entry.SkState),
				console.StyleGreen.Render(pid),
			})
		} else {
			tw.AppendRow(table.Row{entry.Protocol, srcAddr, dstAddr, entry.SkState, pid})
		}
	}
	con.Printf("%s\n", tw.Render())
}

func getActiveC2(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.ActiveC2
	}
	if beacon != nil {
		return beacon.ActiveC2
	}
	return ""
}

func getPID(session *clientpb.Session, beacon *clientpb.Beacon) int32 {
	if session != nil {
		return session.PID
	}
	if beacon != nil {
		return beacon.PID
	}
	return -1
}
