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

	"github.com/bishopfox/sliver/protobuf/commpb"
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
// For each implant node, we cut the commpb.Route it directly send it through its C2 RPC channel.
func (r *routes) Add(new *commpb.Route) (route *Route, err error) {

	// Check address / netmask / etc provided in new. Process values if needed
	ip, subnet, err := net.ParseCIDR(new.IPNet)
	if err != nil {
		ip = net.ParseIP(new.IP)
		if ip == nil {
			return nil, fmt.Errorf("(Error parsing route subnet: %s)", err)
		}
		m := subnet.Mask
		mask := net.IPv4Mask(m[0], m[1], m[2], m[3])
		subnet = &net.IPNet{IP: ip, Mask: mask}
	}

	// If there are no registered sessions, we cannot add any route.
	if len(core.Sessions.All()) == 0 {
		return nil, errors.New("Not active or registered sessions: cannot add any network route")
	}

	// Create a blank route based on request parsing
	route = newRouteTo(subnet)

	// Make sure the IP is not contained in one of the active routes' destination subnet.
	err = checkExistingRoutes(route, new.SessionID)
	if err != nil {
		return nil, err
	}

	// This session will be the last node of the route, which will dial endpoint on its host subnet.
	targetSession, err := getSessionForSubnet(new.SessionID, route)
	if err != nil {
		return nil, err
	}

	// Here, add server interfaces check. Not now for tests.

	// We should not have an empty last node session.
	if targetSession == nil {
		return nil, errors.New("last node session is nil, after checking all interfaces")
	}

	// We build the full route to this last node session. Check that route is not nil, in case...
	route, err = buildRouteToSession(targetSession, route)
	if err != nil {
		return nil, err
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

// Given a route (with its target network), we find the session that has access to it.
// If the sessionID is different from 0, we only check against this session, otherwise we go across all of them.
func getSessionForSubnet(sessionID uint32, route *Route) (session *core.Session, err error) {

	// This should be rewritten, especially the else part
	if sessionID != 0 {
		// If an implant ID is given in the request, we directly check its interfaces.
		// The new.ID is normally (and later) used for the route, but we use it as a filter for now.
		session = core.Sessions.Get(sessionID)
		err = checkSessionNetIfaces(route.IPNet.IP, session)
		if err != nil {
			return nil, err
		}
	} else {
		// If no, get interfaces for all implants and verify no doublons.
		// For each implant, check network interfaces. Stop at the first one valid.
		session, err = checkAllSessionIfaces(&route.IPNet)
		if err != nil {
			return nil, err
		}
	}

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
			if comm.session.ID == new.Nodes[0].ID {
				new.comm = comm
			}
		}
	}

	// Or with gateway if one hop
	if new.Gateway != nil {
		for _, comm := range Comms.Active {
			if comm.session.ID == new.Gateway.ID {
				new.comm = comm
			}
		}
	}

	return new, nil
}

// checkExistingRoutes - Verifies that both route subnet/ip and sessionID allows us to add
// a route without messing with the others. Some subnets might overlap but sessionID might enable more precise routing.
func checkExistingRoutes(route *Route, sessionID uint32) error {

	// Only check the implant hasn't two identical routes.
	if sessionID != 0 {
		for _, rt := range Routes.Registered {
			if rt.comm.session.ID == sessionID && rt.IPNet.Contains(route.IPNet.IP) {
				return fmt.Errorf("Route %s (Mask: %d, ID:%s) [via %s(%d) at %s] is colliding",
					rt.IPNet.IP.String(), route.IPNet.Mask, rt.ID.String(),
					rt.Gateway.Name, rt.Gateway.ID, rt.Gateway.RemoteAddress)
			}
		}
	}

	// Else check all routes.
	for _, rt := range Routes.Registered {
		if rt.IPNet.Contains(route.IPNet.IP) {
			return fmt.Errorf("Route %s (Mask: %d, ID:%s) [via %s(%d) at %s] is colliding",
				rt.IPNet.IP.String(), route.IPNet.Mask, rt.ID.String(),
				rt.Gateway.Name, rt.Gateway.ID, rt.Gateway.RemoteAddress)
		}
	}

	return nil
}
