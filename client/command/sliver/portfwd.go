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
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/maxlandon/readline"
)

var (
	portfwdLog = log.ClientLogger.WithField("portfwd", "portfwd")
)

// Portfwd - Port forwards mangement command; Prints them by default.
// Not that this command will only have subcommands available in the Sliver menu,
// and that this precise struct will only be assigned once in the server package.
type Portfwd struct{}

// Execute -  Print port forwarders for all sessions, and current above
func (p *Portfwd) Execute(args []string) (err error) {

	portfwds := core.Portfwds.List()
	if len(portfwds) == 0 {
		fmt.Printf(Info + "No port forwards\n")
		return
	}
	sort.Slice(portfwds[:], func(i, j int) bool {
		return portfwds[i].ID < portfwds[j].ID
	})

	// Table headers
	headers := []string{"ID", "Session ID", "Local Address", "Remote Address"}
	headLen := []int{5, 10, 20, 20}

	// We might use two different tables depending on if we have a current session or not.
	sessForwarders := util.NewTable(readline.Bold(readline.Blue("Current Session \n")))
	sessForwarders.SetColumns(headers, headLen)
	var sessCount int

	allForwarders := util.NewTable(readline.Bold(readline.Blue("All sessions\n")))
	allForwarders.SetColumns(headers, headLen)
	var allCount int

	// Add forwarders to their table
	for _, p := range portfwds {
		row := []string{strconv.Itoa(p.ID), strconv.Itoa(int(p.SessionID)), p.BindAddr, p.RemoteAddr}
		if core.ActiveSession != nil && p.SessionID == core.ActiveSession.ID {
			sessForwarders.Append(row)
			sessCount++
		} else {
			allForwarders.Append(row)
			allCount++
		}
	}

	// Print any or both tables, adjusting for newlines
	if sessCount > 0 {
		sessForwarders.Output()
	}
	if allCount > 0 {
		if sessCount > 0 {
			fmt.Println()
		}
		allForwarders.Output()
	}

	return
}

// PortfwdAdd - Create a new port forwarding tunnel.
type PortfwdAdd struct {
	Options struct {
		Bind   string `long:"bind" short:"b" description:"bind port forward to interface" default:"127.0.0.1:8080" required:"yes"`
		Remote string `long:"remote" short:"r" description:"remote target host:port (e.g., 10.0.0.1:445)" required:"yes"`
	} `group:"forwarder options"`
}

// Execute - Create a new port forwarding tunnel.
func (p *PortfwdAdd) Execute(args []string) (err error) {
	session := core.ActiveSession

	if session.GetActiveC2() == "dns" {
		fmt.Printf(Warning + "Current C2 is DNS, this is going to be a very slow tunnel!\n")
	}
	if session.Transport == "wg" {
		fmt.Printf(Warning + "Current C2 is WireGuard, we recommend using the `wg-portfwd` command!\n")
	}

	bindAddr := p.Options.Bind
	remoteAddr := p.Options.Remote
	remoteHost, remotePort, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		fmt.Print(Error+"Failed to parse remote target %s\n", err)
		return
	}
	if remotePort == "3389" {
		fmt.Print(Warning + "RDP is unstable over tunneled portfwds, we recommend using WireGuard portfwds\n")
	}

	tcpProxy := &tcpproxy.Proxy{}
	channelProxy := &core.ChannelProxy{
		Rpc:             transport.RPC,
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
			portfwdLog.Errorf("Proxy error: %s", err)
		}
	}()

	fmt.Printf(Info+"Port forwarding %s -> %s:%s\n", bindAddr, remoteHost, remotePort)
	return
}

// PortfwdRm - Remove a port forwarding tunnel
type PortfwdRm struct {
	Args struct {
		ID []int `description:"port forwarder ID" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Remove a port forwarding tunnel
func (p *PortfwdRm) Execute(args []string) (err error) {

	for _, portfwdID := range p.Args.ID {
		found := core.Portfwds.Remove(portfwdID)
		if !found {
			fmt.Printf(Error+"No portfwd with id %d\n", portfwdID)
		} else {
			fmt.Println(Info + "Removed portfwd")
		}
	}
	return
}
