package sliver

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

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// WireGuardPortFwd - Manage WireGuard-based port forwarders. Lists them by default.
type WireGuardPortFwd struct {
}

// Execute - List WireGuard-based port forwarders.
func (w *WireGuardPortFwd) Execute(args []string) (err error) {

	fwdList, err := transport.RPC.WGListForwarders(context.Background(), &sliverpb.WGTCPForwardersReq{
		Request: core.ActiveSessionRequest(),
	})

	if err != nil {
		fmt.Printf(Error+"Error: %v", err)
		return
	}

	if fwdList.Response != nil && fwdList.Response.Err != "" {
		fmt.Printf(Error+"Error: %s\n", fwdList.Response.Err)
		return
	}

	if fwdList.Forwarders == nil || len(fwdList.Forwarders) == 0 {
		fmt.Printf(Info + "No port forwards\n")
		return
	}

	table := util.NewTable("")
	headers := []string{"ID", "Local Address", "Remote Address"}
	headLen := []int{5, 20, 20}
	table.SetColumns(headers, headLen)

	for _, fwd := range fwdList.Forwarders {
		table.Append([]string{strconv.Itoa(int(fwd.ID)), fwd.LocalAddr, fwd.RemoteAddr})
	}
	table.Output()

	return
}

// WireGuardPortFwdAdd - Add a port forward from the WireGuard tun interface to a host on the target network.
type WireGuardPortFwdAdd struct {
	Options struct {
		Bind   int32  `long:"bind" short:"b" description:"port to listen on the WireGuard tun interface" default:"1080"`
		Remote string `long:"remote" short:"r" description:"remote target host:port (e.g., 10.0.0.1:445)" required:"yes"`
	} `group:"forwarder options"`
}

// Execute - Add a port forward from the WireGuard tun interface to a host on the target network.
func (w *WireGuardPortFwdAdd) Execute(args []string) (err error) {

	remoteHost, remotePort, err := net.SplitHostPort(w.Options.Remote)
	if err != nil {
		fmt.Print(Error+"Failed to parse remote target %s\n", err)
		return
	}

	pfwdAdd, err := transport.RPC.WGStartPortForward(context.Background(), &sliverpb.WGPortForwardStartReq{
		LocalPort:     w.Options.Bind,
		RemoteAddress: w.Options.Remote,
		Request:       core.ActiveSessionRequest(),
	})

	if err != nil {
		fmt.Printf(Error+"Error: %v", err)
		return
	}

	if pfwdAdd.Response != nil && pfwdAdd.Response.Err != "" {
		fmt.Printf(Error+"Error: %s\n", pfwdAdd.Response.Err)
		return
	}
	fmt.Printf(Info+"Port forwarding %s -> %s:%s\n", pfwdAdd.Forwarder.LocalAddr, remoteHost, remotePort)

	return
}

// WireGuardPortFwdRm - Remove a port forward from the WireGuard tun interface
type WireGuardPortFwdRm struct {
	Args struct {
		ID []int32 `description:"forward rule ID" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Remove a port forward from the WireGuard tun interface
func (w *WireGuardPortFwdRm) Execute(args []string) (err error) {

	for _, id := range w.Args.ID {

		if id == -1 {
			continue
		}

		stopReq, err := transport.RPC.WGStopPortForward(context.Background(), &sliverpb.WGPortForwardStopReq{
			ID:      id,
			Request: core.ActiveSessionRequest(),
		})

		if err != nil {
			fmt.Printf(Error+"Error: %v", err)
			continue
		}

		if stopReq.Response != nil && stopReq.Response.Err != "" {
			fmt.Printf(Error+"Error: %v\n", stopReq.Response.Err)
			continue
		}

		if stopReq.Forwarder != nil {
			fmt.Printf(Error+"Removed port forwarding rule %s -> %s\n", stopReq.Forwarder.LocalAddr, stopReq.Forwarder.RemoteAddr)
			continue
		}

	}
	return
}

// WireGuardSocks - Manage WireGuard-based Socks proxies. Prints them by default
type WireGuardSocks struct {
}

// Execute - Lists WireGuard-based Socks proxies.
func (w *WireGuardSocks) Execute(args []string) (err error) {

	socksList, err := transport.RPC.WGListSocksServers(context.Background(), &sliverpb.WGSocksServersReq{
		Request: core.ActiveSessionRequest(),
	})

	if err != nil {
		fmt.Printf(Warning+"Error: %v", err)
		return
	}

	if socksList.Response != nil && socksList.Response.Err != "" {
		fmt.Printf(Warning+"Error: %s\n", socksList.Response.Err)
		return
	}

	if socksList.Servers == nil || len(socksList.Servers) == 0 {
		fmt.Printf(Info + "No WireGuard Socks proxies\n")
		return
	}

	table := util.NewTable("WireGuard Socks Proxies \n")

	headers := []string{"ID", "Local Address"}
	headLen := []int{5, 20}
	table.SetColumns(headers, headLen)

	for _, proxy := range socksList.Servers {
		table.Append([]string{strconv.Itoa(int(proxy.ID)), proxy.LocalAddr})
	}
	table.Output()

	return
}

// WireGuardSocksStart - Start a socks5 listener on the WireGuard tun interface
type WireGuardSocksStart struct {
	Options struct {
		Bind int32 `long:"bind" short:"b" description:"port to listen on the WireGuard tun interface" default:"3090"`
	}
}

// Execute - Start a socks5 listener on the WireGuard tun interface
func (w *WireGuardSocksStart) Execute(args []string) (err error) {

	socks, err := transport.RPC.WGStartSocks(context.Background(), &sliverpb.WGSocksStartReq{
		Port:    w.Options.Bind,
		Request: core.ActiveSessionRequest(),
	})

	if err != nil {
		fmt.Printf(Error+"Error: %v", err)
		return
	}

	if socks.Response != nil && socks.Response.Err != "" {
		fmt.Printf(Error+"Error: %s\n", err)
		return
	}

	if socks.Server != nil {
		fmt.Printf(Info+"Started SOCKS server on %s\n", socks.Server.LocalAddr)
	}
	return
}

// WireGuardSocksStop - Stop a socks5 listener on the WireGuard tun interface
type WireGuardSocksStop struct {
	Args struct {
		ID []int32 `description:"socks server ID" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Stop a socks5 listener on the WireGuard tun interface
func (w *WireGuardSocksStop) Execute(args []string) (err error) {

	for _, socksID := range w.Args.ID {

		if socksID == -1 {
			continue
		}

		stopReq, err := transport.RPC.WGStopSocks(context.Background(), &sliverpb.WGSocksStopReq{
			ID:      int32(socksID),
			Request: core.ActiveSessionRequest(),
		})

		if err != nil {
			fmt.Printf(Error+"Error: %v", err)
			continue
		}

		if stopReq.Response != nil && stopReq.Response.Err != "" {
			fmt.Printf(Error+"Error: %v\n", stopReq.Response.Err)
			continue
		}

		if stopReq.Server != nil {
			fmt.Printf(Info+"Removed socks listener rule %s \n", stopReq.Server.LocalAddr)
		}
	}

	return
}
