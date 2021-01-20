package commands

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
	"net"
	"strconv"
	"strings"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Ifconfig - Show session network interfaces
type Ifconfig struct{}

// Execute - Show session network interfaces
func (i *Ifconfig) Execute(args []string) (err error) {

	ifconfig, err := transport.RPC.Ifconfig(context.Background(), &sliverpb.IfconfigReq{
		Request: ContextRequest(cctx.Context.Sliver.Session),
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}

	for ifaceIndex, iface := range ifconfig.NetInterfaces {
		fmt.Printf("%s%s%s (%d)\n", bold, iface.Name, normal, ifaceIndex)
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
				fmt.Printf(bold+green+"    IP Address: %s%s\n", ip, normal)
			} else if 32 < subnet && !isLoopback(ip) {
				fmt.Printf(bold+cyan+"    IP Address: %s%s\n", ip, normal)
			} else {
				fmt.Printf("    IP Address: %s\n", ip)
			}
		}
	}
	return
}

func isLoopback(ip string) bool {
	if strings.HasPrefix(ip, "127") || strings.HasPrefix(ip, "::1") {
		return true
	}
	return false
}

// Netstat - Print session active sockets
type Netstat struct {
	Options struct {
		TCP    bool `long:"tcp" description:"Exclude TCP connections"`
		UDP    bool `long:"udp" description:"Include UDP connections"`
		IPv4   bool `long:"ip4" description:"Exclude IPv4 address sockets"`
		IPv6   bool `long:"ip6" description:"Include IPv6 address sockets"`
		Listen bool `long:"listen" description:"Include listening sockets"`
	} `group:"netstat options"`
}

// Execute - Command
func (n *Netstat) Execute(args []string) (err error) {

	listening := n.Options.Listen
	ip4 := !n.Options.IPv4 // By default WE DO NOT EXCLUDE IPv4
	ip6 := n.Options.IPv6
	tcp := !n.Options.TCP // By default WE DO NOT EXCLUDE TCP
	udp := n.Options.UDP

	netstat, err := transport.RPC.Netstat(context.Background(), &sliverpb.NetstatReq{
		TCP:       tcp,
		UDP:       udp,
		Listening: listening,
		IP4:       ip4,
		IP6:       ip6,
		Request:   ContextRequest(cctx.Context.Sliver.Session),
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}
	displayEntries(netstat.Entries)

	return
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
	session := cctx.Context.Sliver
	for _, e := range entries {
		p := ""
		if e.Process != nil {
			p = fmt.Sprintf("%d/%s", e.Process.Pid, e.Process.Executable)
		}
		srcAddr := lookup(e.LocalAddr)
		dstAddr := lookup(e.RemoteAddr)
		if e.Process != nil && e.Process.Pid == session.PID && isSliverAddr(dstAddr) {
			fmt.Printf("%s%-5s %-23.23s %-23.23s %-12s %-16s%s\n",
				green, e.Protocol, srcAddr, dstAddr, e.SkState, p, normal)
		} else {
			fmt.Printf("%-5s %-23.23s %-23.23s %-12s %-16s\n",
				e.Protocol, srcAddr, dstAddr, e.SkState, p)
		}
	}
}

func isSliverAddr(dstAddr string) bool {
	parts := strings.Split(dstAddr, ":")
	if len(parts) != 3 {
		return false
	}
	c2Addr := strings.Split(cctx.Context.Sliver.ActiveC2, "://")[1]
	return strings.Join(parts[:2], ":") == c2Addr
}
