package portfwd

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
	"log"
	"net"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/desertbit/grumble"
)

// PortfwdAddCmd - Add a new tunneled port forward
func PortfwdAddCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.GetActiveC2() == "dns" {
		con.PrintWarnf("Current C2 is DNS, this is going to be a very slow tunnel!\n")
	}
	if session.Transport == "wg" {
		con.PrintWarnf("Current C2 is WireGuard, we recommend using the `wg-portfwd` command!\n")
	}
	remoteAddr := ctx.Flags.String("remote")
	if remoteAddr == "" {
		con.PrintErrorf("Must specify a remote target host:port\n")
		return
	}
	remoteHost, remotePort, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		con.PrintErrorf("Failed to parse remote target %s\n", err)
		return
	}
	if remotePort == "3389" {
		con.PrintWarnf("RDP is generally broken over tunneled portfwds, we recommend using WireGuard portfwds\n")
	}
	bindAddr := ctx.Flags.String("bind")
	if remoteAddr == "" {
		con.PrintErrorf("Must specify a bind target host:port")
		return
	}

	tcpProxy := &tcpproxy.Proxy{}
	channelProxy := &core.ChannelProxy{
		Rpc:             con.Rpc,
		Session:         session,
		RemoteAddr:      remoteAddr,
		BindAddr:        bindAddr,
		KeepAlivePeriod: 60 * time.Second,
		DialTimeout:     30 * time.Second,
	}
	tcpProxy.AddRoute(bindAddr, channelProxy)
	core.Portfwds.Add(tcpProxy, channelProxy)

	go func() {
		err := tcpProxy.Run()
		if err != nil {
			log.Printf("Proxy error %s", err)
		}
	}()

	con.PrintInfof("Port forwarding %s -> %s:%s\n", bindAddr, remoteHost, remotePort)
}
