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
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/desertbit/grumble"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func routes(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	routes, err := rpc.Routes(context.Background(), &sliverpb.RoutesReq{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}
	// Add filters here

	if len(routes.Active) == 0 {
		fmt.Printf("No registered / active network routes")
		return
	}

	printRoutes(routes.Active)
}

func addRoute(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if ctx.Flags.String("subnet") == "" {
		fmt.Printf(Warn + "Missing subnet flag for route\n")
		return
	}

	// For now we don't process or get an implantID for this route.
	routeAdd, err := rpc.AddRoute(context.Background(), &sliverpb.AddRouteReq{
		Route: &sliverpb.Route{
			IPNet:     ctx.Flags.String("subnet"),
			Mask:      ctx.Flags.String("netmask"),
			SessionID: uint32(ctx.Flags.Uint("session-id")),
		},
	})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	} else if routeAdd.Response.Err != "" {
		fmt.Printf(Warn+"%s\n", routeAdd.Response.Err)
	} else {
		fmt.Printf(Info+"Successfully added route %s", ctx.Flags.String("subnet"))
	}
}

func removeRoute(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	// Get routes
	routes, err := rpc.Routes(context.Background(), &sliverpb.RoutesReq{})
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	// List of routes we will delete
	toDelete := []*sliverpb.Route{}

	// For now, just catch exactly matching networks.
	network := ctx.Flags.String("network")
	if network != "" {
		for _, route := range routes.Active {
			if route.IPNet == network {
				toDelete = append(toDelete, route)
			}
		}
	}

	// By route ID
	id := ctx.Flags.String("id")
	if id != "" {
		for _, route := range routes.Active {
			if route.ID == id {
				toDelete = append(toDelete, route)
			}
		}
	}

	// Add active filter

	for _, route := range toDelete {
		deleteReq := &sliverpb.RmRouteReq{
			Route: route,
			Close: ctx.Flags.Bool("close"),
		}
		deleted, err := rpc.RemoveRoute(context.Background(), deleteReq)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		} else if !deleted.Success {
			fmt.Printf(Warn+"%s\n", deleted.Response.Err)
		} else {
			fmt.Printf(Info+"Removed route to network %s (Session: %s)", route.IPNet, route.Gateway.Name)
		}
		if deleted.CloseError != "" {
			fmt.Printf(Warn+"Warning route connection closed: %s\n", deleted.Response.Err)
		}
	}
}

func printRoutes(routes []*sliverpb.Route) {

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "Network\tGateway\tConnections\tNode IDs\tRoute ID\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Network")),
		strings.Repeat("=", len("Gateway")),
		strings.Repeat("=", len("Connections")),
		strings.Repeat("=", len("Node IDs")),
		strings.Repeat("=", len("Route ID")))

	for _, route := range routes {

		gateway := fmt.Sprintf("%s(%d)", route.Gateway.Name, route.Gateway.ID)
		var tcp int
		var udp int
		for _, c := range route.Connections {
			if c.Transport == sliverpb.TransportProtocol_TCP {
				tcp++
			}
			if c.Transport == sliverpb.TransportProtocol_UDP {
				udp++
			}
		}
		connections := fmt.Sprintf("%d tcp / %d udp", tcp, udp)

		nodeIDs := ""
		for _, n := range route.Nodes {
			nodeIDs += fmt.Sprintf("%d ->", n.ID)
		}
		nodeIDs = strings.TrimSuffix(nodeIDs, "->")

		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
			route.IPNet,
			gateway,
			connections,
			nodeIDs,
			route.ID,
		)
	}
	fmt.Printf(outputBuf.String())
}
