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
	"net/url"
)

// ResolveIP - Given an IP, find a route to the host.
func ResolveIP(ip net.IP) (route *Route, err error) {
	for _, r := range Routes.Registered {
		found, err := r.Network.Contains(ip)
		if err != nil {
			rLog.Errorf("Error checking IP for route (ID: %s) subnet %s: %s", r.ID, r.IPNet.String(), err.Error())
			continue
		}
		if found {
			return r, nil
		}
	}
	return nil, nil
}

// ResolveURL - Given a URL, find a route to the host.
func ResolveURL(uri *url.URL) (route *Route, err error) {

	// Directly parse addr as an IP. Using hostname ensures we don't take the port into this.
	ip := net.ParseIP(uri.Hostname())
	if ip == nil {
		return nil, fmt.Errorf("Error parsing host %s (to IP) for route: %s", uri.Hostname(), err.Error())
	}

	// Then check for each route that the IP is contained.
	for _, r := range Routes.Registered {
		found, err := r.Network.Contains(ip)
		if err != nil {
			rLog.Errorf("Error checking IP for route (ID: %s) subnet %s: %s", r.ID, r.IPNet.String(), err.Error())
			continue
		}
		if found {
			return r, nil
		}
	}

	return nil, nil
}

// ResolveAddress - Gives a route to a string address. Used to trigger remote listeners, setup portforwards, etc.
// If no routes exist for this address, returns a nil Route and no error: callers will either know the
// address belongs to one of the server's interfaces, or they will check on their own to make sure.
func ResolveAddress(addr string) (route *Route, err error) {

	// Parse address into a URL.
	uri, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("Error parsing address %s (to URL) for route: %s", addr, err.Error())
	}

	// Directly parse addr as an IP. Using hostname ensures we don't take the port into this.
	ip := net.ParseIP(uri.Hostname())
	if ip == nil {
		return nil, fmt.Errorf("Error parsing host %s (to IP) for route: %s", uri.Hostname(), err.Error())
	}

	// Then check for each route that the IP is contained.
	for _, r := range Routes.Registered {
		found, err := r.Network.Contains(ip)
		if err != nil {
			rLog.Errorf("Error checking IP for route (ID: %s) subnet %s: %s", r.ID, r.IPNet.String(), err.Error())
			continue
		}
		if found {
			return r, nil
		}
	}

	return nil, nil
}

// ListenHandlerID - A module or a handler has sent a request to its gateway session, with a given handler ID.
// We pass this ID and the route, initialize nodes, create a new tracker and return an abstracted listener.
//
// @id      => A unique ID for this handler, used by all nodes to keep track of associated connections.
// @network => The type of protocol (transport: "tcp/udp/unix", or application: "https/dns/named_pipe/etc.")
//             This is just for helping the Comm system to populate listener and connection informations,
//             and it will NOT output connections with a transport like "h2+ssh+myass". This just determines
//              if outgoing connections are either TCP/UDP/Unix connections, and we treat them as such.
// @host    => The listener host:port adress.
// @route   => Gives the nodes in the route.
func ListenHandlerID(id, network, host string, route *Route) (ln net.Listener, err error) {
	// If route is not nil
	if route == nil {
		return nil, errors.New("tried to create abstracted listener on nil route")
	}

	// Create abstracted listener, with chans, and register it.
	tracker := newListener(id, network, host)
	listeners.Add(tracker)

	// Maybe send id to all nodes in route.

	return tracker, nil
}
