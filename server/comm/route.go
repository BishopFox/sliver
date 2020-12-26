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
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/yl2chen/cidranger"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

// Route - A network route. This object is autonomous and has all
// references needed so as to route traffic matching the route.
type Route struct {
	ID          uuid.UUID        // A unique ID for this route. Used everywhere.
	IPNet       net.IPNet        // Target IP network.
	Network     cidranger.Ranger // CIDR ranger used to check for subnets/bignets.
	Gateway     *core.Session    // The session' host has access to the target IPNet
	Nodes       []*core.Session  // All nodes between the implant gateway and the server.
	Active      bool             // Is this route up and running ?
	Connections []net.Conn       // All active connections. We don't keep their ID, and timeouts are set.
	comm        *Comm            // The multiplexer to which we pass connections.
	mutex       *sync.RWMutex

	// Add specific, cascading keepalives, like context, and pass them to commands.
}

// newRoute - Create a new route based on an address in CIDR notation, or an address with a netmask provided.
func newRouteTo(subnet *net.IPNet) *Route {

	id, _ := uuid.NewGen().NewV1() // New route always has a new UUID.
	route := &Route{
		ID:     id,
		IPNet:  *subnet,
		Active: false,
		mutex:  &sync.RWMutex{},
	}
	// Add network for more precise processing.
	route.Network = cidranger.NewPCTrieRanger()
	route.Network.Insert(cidranger.NewBasicRangerEntry(*subnet))

	return route
}

// Dial - Get a network connection to a host in this route. Available networks are tcp/udp/unix/ip
func (r *Route) Dial(network string, host string) (conn net.Conn, err error) {
	return r.DialContext(context.Background(), network, host)
}

// DialContext - Get a network connection to a host in this route, with a Context. See Dial() for networks.
func (r *Route) DialContext(ctx context.Context, network string, host string) (conn net.Conn, err error) {

	// Get RHost/RPort
	uri, _ := url.Parse(fmt.Sprintf("%s://%s", network, host))
	if uri == nil {
		return nil, fmt.Errorf("Address parsing failed: %s", host)
	}

	info := newConnInfo(uri, r)                 // Prepare connection info with route elements.
	conn, err = r.comm.dial(info)               // Instantiate connection over Comms
	r.Connections = append(r.Connections, conn) // Add connection to active

	rLog.Infof("[route] Dialing (%s/%s) %s --> %s (ID: %s)", info.Transport.String(), info.Application.String(),
		conn.LocalAddr().String(), conn.RemoteAddr().String(), info.ID)
	return
}

// ToProtobuf - Returns the protobuf information of this route, used for requests to implant nodes.
func (r *Route) ToProtobuf() *sliverpb.Route {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	// Info
	rt := &sliverpb.Route{
		ID:    r.ID.String(),
		IP:    r.IPNet.IP.String(),
		IPNet: r.IPNet.String(),
		Mask:  r.IPNet.Mask.String(),
	}
	// Nodes
	for _, node := range r.Nodes {
		n := &sliverpb.Node{
			ID:       node.ID,
			Name:     node.Name,
			Host:     node.RemoteAddress,
			Hostname: node.Hostname,
		}
		rt.Nodes = append(rt.Nodes, n)
	}
	// Gateway session
	rt.Gateway = &sliverpb.Node{
		ID:       r.Gateway.ID,
		Name:     r.Gateway.Name,
		Host:     r.Gateway.RemoteAddress,
		Hostname: r.Gateway.Hostname,
	}
	// Current connections & settings.
	for _, cc := range r.Connections {
		rHost := strings.Split(cc.RemoteAddr().String(), ":")[0]
		rPort, _ := strconv.Atoi(strings.Split(cc.RemoteAddr().String(), ":")[1])
		lHost := strings.Split(cc.LocalAddr().String(), ":")[0]
		lPort, _ := strconv.Atoi(strings.Split(cc.LocalAddr().String(), ":")[1])

		connInfo := &sliverpb.ConnectionInfo{
			RHost: rHost,
			RPort: int32(rPort),
			LHost: lHost,
			LPort: int32(lPort),
		}
		if cc.RemoteAddr().Network() == "tcp" {
			connInfo.Transport = sliverpb.TransportProtocol_TCP
		}
		if cc.RemoteAddr().Network() == "udp" {
			connInfo.Transport = sliverpb.TransportProtocol_UDP
		}

		switch cc.RemoteAddr().Network() {
		case "tcp", "tcp4", "tcp6":
		case "udp":
		case "ip":
		}

		rt.Connections = append(rt.Connections, connInfo)
	}

	return rt
}

// Close - Closes all connections that being actively monitored by this route.
// This does not includes active port forwards, and connections linked to handlers.
func (r *Route) Close() {
	rLog.Warnf("[route]: Closing all active (forward) connections !")
	for _, conn := range r.Connections {
		err := conn.Close()
		if err != nil {
			rLog.Errorf("Error closing route connection: %s", err.Error())
		}
		r.Connections = r.Connections[1:] // Delete connection
	}
}

// String - Forges a string of this route target network.
func (r *Route) String() string {
	return fmt.Sprintf("[via %s]", r.Gateway.RemoteAddress)
}
