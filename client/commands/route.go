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
	"strings"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/commpb"
)

// Route - Route management commands
type Route struct{}

// Execute - Command. Default prints the current routes
func (r *Route) Execute(args []string) (err error) {

	routes, err := transport.RPC.GetRoutes(context.Background(), &commpb.RoutesReq{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return
	}

	if len(routes.Active) == 0 {
		fmt.Printf(util.Info + "No registered / active network routes\n")
		return
	}

	printRoutes(routes.Active)
	return
}

func printRoutes(routes []*commpb.Route) {

	table := util.NewTable(readline.Bold(readline.Yellow("Network Routes")))
	headers := []string{"Network CIDR", "Gateway Session", "Connections", "Node IDs", "ID"}
	headLen := []int{0, 0, 0, 0, 0}
	table.SetColumns(headers, headLen)

	for _, route := range routes {
		id := readline.Dim(route.ID)
		network := readline.Bold(route.IPNet)

		gateway := fmt.Sprintf("%s - %s%d%s", route.Gateway.Name, readline.BLUE, route.Gateway.ID, readline.RESET)
		var tcp int
		var udp int
		for _, c := range route.Connections {
			if c.Transport == commpb.Transport_TCP {
				tcp++
			}
			if c.Transport == commpb.Transport_UDP {
				udp++
			}
		}
		connections := fmt.Sprintf("%d tcp / %d udp", tcp, udp)

		nodeIDs := ""
		for _, n := range route.Nodes {
			nodeIDs += fmt.Sprintf("%d ->", n.ID)
		}
		nodeIDs = strings.TrimSuffix(nodeIDs, "->")

		table.AppendRow([]string{network, gateway, connections, nodeIDs, id})
	}
	table.Output()
}

// RouteAdd - Add a network route.
type RouteAdd struct {
	Options struct {
		CIDR      string `long:"cidr" short:"n" description:"IP network in CIDR notation (ex: 192.168.1.1/24)"`
		Mask      string `long:"mask" short:"m" description:"(optional) specify network mask"`
		SessionID uint32 `long:"session-id" short:"s" description:"(optional) bind this route network to a precise implant, in case two routes collide"`
	} `group:"route options"`
}

// Execute - Command
func (r *RouteAdd) Execute(args []string) (err error) {
	if r.Options.CIDR == "" {
		fmt.Printf(util.Error + "Missing --cidr flag for route\n")
		return
	}

	// For now we don't process or get an implantID for this route.
	routeAdd, err := transport.RPC.AddRoute(context.Background(), &commpb.RouteAddReq{
		Route: &commpb.Route{
			IPNet:     r.Options.CIDR,
			Mask:      r.Options.Mask,
			SessionID: r.Options.SessionID,
		},
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	} else if routeAdd.Response.Err != "" {
		fmt.Printf(util.Error+"%s\n", routeAdd.Response.Err)
	} else {
		fmt.Printf(util.Info+"Added new route (%s)\n", readline.Yellow(r.Options.CIDR))
	}

	return
}

// RouteRemove  - Delete an active network route.
type RouteRemove struct {
	Positional struct {
		RouteID []string `description:"ID of route to delete"`
	} `positional-args:"yes"`

	Options struct {
		CIDR      string `long:"cidr" short:"n" description:"close routes matching this network"`
		SessionID uint32 `long:"session-id" short:"s" description:"close routes matching this session"`
		Close     bool   `long:"close-conns" short:"C" description:"close all active connections going route(s)"`
	} `group:"route options"`
}

// Execute - Command
func (r *RouteRemove) Execute(args []string) (err error) {

	// Get routes
	routes, err := transport.RPC.GetRoutes(context.Background(), &commpb.RoutesReq{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
		return
	}

	// List of routes we will delete
	toDelete := []*commpb.Route{}

	// Get routes whose network is contained in the --network filter
	network := r.Options.CIDR
	if network != "" {
		for _, route := range routes.Active {
			_, ipnet, err := net.ParseCIDR(network)
			if err != nil {
				fmt.Printf("Failed to parse --network value: %s", network)
				return nil
			}
			ip, _, _ := net.ParseCIDR(route.IPNet)
			if ip != nil && ipnet.Contains(ip) {
				toDelete = append(toDelete, route)
			}
		}
	}

	// By route ID
	for _, id := range r.Positional.RouteID {
		for _, route := range routes.Active {
			if route.ID == id {
				toDelete = append(toDelete, route)
			}
		}
	}

	// All routes matching the above filters are deleted/closed.
	for _, route := range toDelete {
		deleteReq := &commpb.RouteDeleteReq{
			Route: route,
			Close: r.Options.Close,
		}
		deleted, err := transport.RPC.DeleteRoute(context.Background(), deleteReq)
		if err != nil {
			fmt.Printf(util.RPCError+"%s\n", err)
			return nil
		} else if !deleted.Success {
			fmt.Printf(util.Error+"%s\n", deleted.Response.Err)
		} else {
			fmt.Printf(util.Info+"Removed route to network %s (Session: %s)\n", route.IPNet, route.Gateway.Name)
		}
		if deleted.CloseError != "" {
			fmt.Printf(util.Error+"Warning route connection closed: %s\n", deleted.Response.Err)
		}
	}

	return
}
