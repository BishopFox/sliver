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
	"net"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func NetstatCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	listening := ctx.Flags.Bool("listen")
	ip4 := ctx.Flags.Bool("ip4")
	ip6 := ctx.Flags.Bool("ip6")
	tcp := ctx.Flags.Bool("tcp")
	udp := ctx.Flags.Bool("udp")

	netstat, err := con.Rpc.Netstat(context.Background(), &sliverpb.NetstatReq{
		Request:   con.ActiveSession.Request(ctx),
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
	displayEntries(netstat.Entries, con)
}

func displayEntries(entries []*sliverpb.SockTabEntry, con *console.SliverConsoleClient) {
	lookup := func(skaddr *sliverpb.SockTabEntry_SockAddr) string {
		const IPv4Strlen = 17
		addr := skaddr.Ip
		names, err := net.LookupAddr(addr)
		if err == nil && len(names) > 0 {
			addr = names[0]
		}
		if len(addr) > IPv4Strlen {
			addr = addr[:IPv4Strlen]
		}
		return fmt.Sprintf("%s:%d", addr, skaddr.Port)
	}

	con.Printf("Proto %-23s %-23s %-12s %-16s\n", "Local Addr", "Foreign Addr", "State", "PID/Program name")
	session := con.ActiveSession.GetInteractive()
	for _, e := range entries {
		p := ""
		if e.Process != nil {
			p = fmt.Sprintf("%d/%s", e.Process.Pid, e.Process.Executable)
		}
		srcAddr := lookup(e.LocalAddr)
		dstAddr := lookup(e.RemoteAddr)
		if e.Process != nil && e.Process.Pid == session.PID && isSliverAddr(dstAddr, con) {
			con.Printf("%s%-5s %-23.23s %-23.23s %-12s %-16s%s\n",
				console.Green, e.Protocol, srcAddr, dstAddr, e.SkState, p, console.Normal)
		} else {
			con.Printf("%-5s %-23.23s %-23.23s %-12s %-16s\n",
				e.Protocol, srcAddr, dstAddr, e.SkState, p)
		}
	}
}

func isSliverAddr(dstAddr string, con *console.SliverConsoleClient) bool {
	parts := strings.Split(dstAddr, ":")
	if len(parts) != 3 {
		return false
	}
	c2Addr := strings.Split(con.ActiveSession.GetInteractive().ActiveC2, "://")[1]
	return strings.Join(parts[:2], ":") == c2Addr
}
