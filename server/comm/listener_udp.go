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
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"sync"

	"github.com/gofrs/uuid"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

// ListenUDP - Returns a UDP listener/conn started on a valid network address anywhere in either the
// server interfaces, or any implant's interface if the latter is served by an active route.
func ListenUDP(network, host string) (conn net.PacketConn, err error) {

	addr, err := url.Parse(fmt.Sprintf("%s://%s", network, host))
	if err != nil {
		return nil, fmt.Errorf("comm listener: could not parse URL: %s://%s", network, host)
	}

	// At this level, we need a port, we don't intend to contact the application layer of a URL.
	if addr.Port() == "" || addr.Port() == "0" {
		return nil, fmt.Errorf("comm listener: invalid port number (nil or 0)")
	}

	// Check routes and interfaces
	route, err := ResolveURL(addr)
	if err != nil {
		return nil, err
	}

	// No route, use server interfaces, make a UDP dest addrss and dial.
	if route == nil {
		// Forge UDP source address first
		ip := net.ParseIP(addr.Hostname())
		port, _ := strconv.Atoi(addr.Port())
		laddr := &net.UDPAddr{IP: ip, Port: port}

		return net.ListenUDP(network, laddr)
	}

	// Else, get an abstracted UDP listener. This is blocking until when the
	// listener receives a stream from the Comm system to pass data into.
	conn, err = newListenerUDP(addr, route)
	if err != nil {
		return nil, fmt.Errorf("comm listener: %s", err.Error())
	}

	return
}

// udpListener - A Packet listener that is actually a PacketConn, either tied
// to a UDP listener running on the server's interfaces, or on one of the implants.
// Because this udpListener is actually a PacketConn, its implementation of the
// PacketConn interface is in conn_udp. Only the instantiation is found below.
type udpListener struct {
	info    *sliverpb.Handler
	sess    *core.Session
	stream  *udpStream
	pending chan io.ReadWriteCloser
	sent    int64
	recv    int64
	addr    *net.UDPAddr
}

// newListenerUDP - Creates a new listener tied to an ID, and with basic network/address information.
// This listener is tied to an implant listener, and the latter feeds the former with connections
// that are routed back to the server. Also able to stop the remote listener like server jobs.
func newListenerUDP(uri *url.URL, route *Route) (ln *udpListener, err error) {

	id, _ := uuid.NewGen().NewV1()
	ip := net.ParseIP(uri.Hostname())
	port, _ := strconv.Atoi(uri.Port())

	// Listener object.
	ln = &udpListener{
		info: &sliverpb.Handler{
			ID:        id.String(),
			Type:      sliverpb.HandlerType_Reverse,
			LHost:     uri.Hostname(),
			LPort:     int32(port),
			Transport: sliverpb.TransportProtocol_UDP,
		},
		sess:    route.Gateway,
		pending: make(chan io.ReadWriteCloser),
		addr:    &net.UDPAddr{IP: ip, Port: port},
	}

	// Application Protocol
	switch uri.Scheme {
	case "dns":
		ln.info.Application = sliverpb.ApplicationProtocol_DNS
	}

	// We first register the abstracted listener: we don't know how fast the implant
	// comm might send us a connection back. We deregister if failure.
	listenersUDP.Add(ln)

	// Forge request to implant.
	req := &sliverpb.HandlerStartReq{
		Handler: ln.info,
		Request: &commonpb.Request{SessionID: route.Gateway.ID},
	}

	// Start waiting for the Comm system to return us the stream  that
	// will be created by the handler upon arrival of the request below
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Wait for the connection.
		stream := <-ln.pending

		// Wrap the connection's stream into a UDP stream, with encoding
		ln.stream = &udpStream{
			r: gob.NewDecoder(stream),
			w: gob.NewEncoder(stream),
			c: stream,
		}
	}()

	// We now request to the implant to start its listener.
	res := &sliverpb.HandlerStart{}
	err = remoteHandlerRequest(route.Gateway, req, res)
	if err != nil {
		listenersTCP.Remove(ln.info.ID)
		return nil, fmt.Errorf("comm listener RPC error: %s", err.Error())
	}
	if !res.Success {
		listenersTCP.Remove(ln.info.ID)
		return nil, fmt.Errorf("comm listener: %s", res.Response.Err)
	}

	// Wait for the listener to aquire its stream from the Comm system.
	wg.Wait()
	rLog.Infof("Bound UDP inbound stream: server <-- %s", ln.addr.String())

	return ln, nil
}

var (
	// listeners - All instantiated active connection listeners. These listeners
	// are either tied to a listener running on an implant host, or the C2 server.
	listenersUDP = &udpListeners{
		active: map[string]*udpListener{},
		mutex:  &sync.RWMutex{},
	}
)

type udpListeners struct {
	active map[string]*udpListener
	mutex  *sync.RWMutex
}

// Get - Get a session by ID
func (c *udpListeners) Get(id string) *udpListener {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.active[id]
}

// Add - Add a sliver to the hive (atomically)
func (c *udpListeners) Add(l *udpListener) *udpListener {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.active[l.info.ID] = l
	return l
}

// Remove - Remove a sliver from the hive (atomically)
func (c *udpListeners) Remove(id string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	l := c.active[id]
	if l != nil {
		delete(c.active, id)
	}
}
