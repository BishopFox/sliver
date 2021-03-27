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
	if err != nil || uri == nil {
		return nil, fmt.Errorf("Error parsing address %s (to URL) for route: %s", addr, err.Error())
	}

	// Directly parse addr as an IP. Using hostname ensures we don't take the port into this.
	ip := net.ParseIP(uri.Hostname())
	if ip == nil {
		return nil, fmt.Errorf("Error parsing host %s (to IP) for route", uri.Hostname())
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
