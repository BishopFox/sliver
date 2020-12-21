package comm

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
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

var (
	// Routes - All network routes registered in the framework.
	Routes = &routes{
		Registered: map[string]*Route{},
		mutex:      &sync.RWMutex{},
	}
	routeID = uint32(0)

	defaultNetTimeout = 10 * time.Second // Default, in case nowhere to be found.
)

type routes struct {
	Registered map[string]*Route
	mutex      *sync.RWMutex
}

// Add - A user has requested to open a route. Send requests to all nodes in the route chain,
// so they know how to handle traffic directed at a certain address, and register the route.
// For each implant node, we cut the sliverpb.Route it directly send it through its C2 RPC channel.
func (r *routes) Add(new *sliverpb.Route) (route *Route, err error) {

	// Check address / netmask / etc provided in new. Process values if needed
	ip, subnet, err := net.ParseCIDR(new.IPNet)
	if err != nil {
		ip = net.ParseIP(new.IP)
		if ip == nil {
			return nil, fmt.Errorf("Error parsing route subnet: %s", err)
		}
		m := subnet.Mask
		mask := net.IPv4Mask(m[0], m[1], m[2], m[3])
		subnet = &net.IPNet{IP: ip, Mask: mask}
	}

	// Create a blank route based on request parsing
	route = newRoute(subnet)

	// Make sure the IP is not contained in one of the active routes' destination subnet.
	err = checkExistingRoutes(route, new.SessionID)
	if err != nil {
		return nil, fmt.Errorf("Error adding route: %s", err.Error())
	}

	// This session will be the last node of the route, which will dial endpoint on its host subnet.
	var lastNodeSession *core.Session

	// This should be rewritten, especially the else part
	if new.SessionID != 0 {
		// If an implant ID is given in the request, we directly check its interfaces.
		// The new.ID is normally (and later) used for the route, but we use it as a filter for now.
		lastNodeSession = core.Sessions.Get(new.SessionID)
		err = checkSessionNetIfaces(ip, lastNodeSession)
		if err != nil {
			return nil, err
		}
	} else {
		// If no, get interfaces for all implants and verify no doublons.
		// For each implant, check network interfaces. Stop at the first one valid.
		lastNodeSession, err = checkAllSessionIfaces(subnet)
		if err != nil {
			return nil, fmt.Errorf("Error adding route: %s", err.Error())
		}
	}

	// Here, add server interfaces check. Not now for tests.

	// We should not have an empty last node session.
	if lastNodeSession == nil {
		return nil, errors.New("Error adding route: last node' session is nil, after checking all interfaces")
	}

	// We build the full route to this last node session. Check that route is not nil, in case...
	route, err = buildRouteToSession(lastNodeSession, route)
	if err != nil {
		return nil, err
	}

	// Send a C2 request to each implant node in the chain. If any error arises
	// from a node, the route will automatically cancel successful nodes.
	err = route.init()
	if err != nil {
		return nil, fmt.Errorf("Failed to init route (currently being cancelled): %s", err)
	}

	// Add to Routes map
	r.mutex.Lock()
	r.Registered[route.ID.String()] = route
	r.mutex.Unlock()

	return
}

// Remove - We notify all implant nodes on the route to stop routing traffic, and deregister the route.
// This does not kill any pending connections / data streams that the route is handling.
func (r *routes) Remove(routeID string, close bool) (err error) {

	route, found := r.Registered[routeID]
	if !found {
		return fmt.Errorf("Provided route ID (%s) does not exist", routeID)
	}

	// Send request to remove route to all implant nodes.
	err = route.remove()
	if err != nil {
		return fmt.Errorf("Error removing route: %s", err.Error())
	}
	// Close all active connections (forwarded ones, not listeners and portforwards)
	if close {
		route.Close()
	}

	// Remove route from Active
	r.mutex.Lock()
	delete(r.Registered, route.ID.String())
	r.mutex.Unlock()

	return
}

// buildRouteToSession - Given a session, we build the full route (all nodes) to this session.
// Each node will have the corresponding implant's active transport address and the Session ID.
// By default, the route destination subnet and netmask is the active route's leading to the node we are about to add.
func buildRouteToSession(sess *core.Session, new *Route) (*Route, error) {

	// The session remote address is mandatorily accesible through one
	// of the routes' destination networks, as all pivot listeners have been
	addr := strings.Split(sess.RemoteAddress, ":")[0]
	ip := net.ParseIP(addr)

	// Find a potential active route that might be leading to session, or closest to it.
	var existing *Route
	for _, rt := range Routes.Registered {
		if rt.IPNet.Contains(ip) {
			existing = rt
		}
	}

	// If we have an existing route on which to build the new one, add its nodes.
	if existing != nil {
		// Else we add to the new route:
		new.Nodes = append(new.Nodes, existing.Nodes...) // The found route nodes
		new.Nodes = append(new.Nodes, existing.Gateway)  // The gateway, itself becoming a node
	}

	// Our session is the new gateway.
	new.Gateway = sess

	// Reference the first node/gateway SSH Comm object
	if len(new.Nodes) > 0 {
		for _, comm := range Comms.Active {
			if comm.SessionID == new.Nodes[0].ID {
				new.comm = comm
			}
		}
	}

	// Or with gateway if one hop
	if new.Gateway != nil {
		for _, comm := range Comms.Active {
			if comm.SessionID == new.Gateway.ID {
				new.comm = comm
			}
		}
	}

	return new, nil
}

// checkExistingRoutes - Verifies that both route subnet/ip and sessionID allows us to add
// a route without messing with the others. Some subnets might overlap but sessionID might enable more precise routing.
func checkExistingRoutes(route *Route, sessionID uint32) error {

	if sessionID != 0 {
	}

	for _, rt := range Routes.Registered {
		if rt.IPNet.Contains(route.IPNet.IP) {
			return fmt.Errorf("Active route %s (Mask: %s, ID:%d) via (%s(%d) at %s) is colliding",
				rt.IPNet.IP.String(), route.IPNet.Mask.String(), rt.ID, rt.Gateway.Name, rt.Gateway.ID, rt.Gateway.RemoteAddress)
		}
	}

	return nil
}
