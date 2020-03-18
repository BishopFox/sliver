package command

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
	"fmt"
	"net"
	"strings"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func netstat(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	listening := ctx.Flags.Bool("listen")
	ip4 := ctx.Flags.Bool("ip4")
	ip6 := ctx.Flags.Bool("ip6")
	tcp := ctx.Flags.Bool("tcp")
	udp := ctx.Flags.Bool("udp")

	data, _ := proto.Marshal(&sliverpb.NetstatRequest{
		SliverID:  ActiveSliver.Sliver.ID,
		TCP:       tcp,
		UDP:       udp,
		Listening: listening,
		IP4:       ip4,
		IP6:       ip6,
	})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgNetstatReq,
		Data: data,
	}, defaultTimeout)
	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}
	netstatResp := &sliverpb.NetstatResponse{}
	err := proto.Unmarshal(resp.Data, netstatResp)
	if err != nil {
		fmt.Printf(Warn + "Failed to decode response\n")
		return
	}
	displayEntries(netstatResp.Entries)
}

func displayEntries(entries []*sliverpb.SockTabEntry) {
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

	fmt.Printf("Proto %-23s %-23s %-12s %-16s\n", "Local Addr", "Foreign Addr", "State", "PID/Program name")
	for _, e := range entries {
		p := ""
		if e.Proc != nil {
			p = fmt.Sprintf("%d/%s", e.Proc.Pid, e.Proc.Executable)
		}
		saddr := lookup(e.LocalAddr)
		daddr := lookup(e.RemoteAddr)
		if e.Proc != nil && e.Proc.Pid == ActiveSliver.Sliver.PID && isSliverAddr(daddr) {
			fmt.Printf("%s%-5s %-23.23s %-23.23s %-12s %-16s%s\n", green, e.Proto, saddr, daddr, e.SkState, p, normal)
		} else {

			fmt.Printf("%-5s %-23.23s %-23.23s %-12s %-16s\n", e.Proto, saddr, daddr, e.SkState, p)
		}
	}
}

func isSliverAddr(daddr string) bool {
	parts := strings.Split(daddr, ":")
	if len(parts) != 3 {
		return false
	}
	c2Addr := strings.Split(ActiveSliver.Sliver.ActiveC2, "://")[1]
	return strings.Join(parts[:2], ":") == c2Addr
}
