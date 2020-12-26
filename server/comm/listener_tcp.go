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
	"strconv"
	"sync"

	"github.com/gofrs/uuid"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

// ListenTCP - Returns a TCP listener started on a valid network address anywhere in either
// the server interfaces, or any implant's interface if the latter is served by an active route.
func ListenTCP(network, host string) (ln net.Listener, err error) {

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

	// No route, use server interfaces, make a TCP dest addrss and dial.
	if route == nil {
		ip := net.ParseIP(addr.Hostname())
		port, _ := strconv.Atoi(addr.Port())
		tcpAddr := &net.TCPAddr{IP: ip, Port: port}

		return net.ListenTCP("tcp", tcpAddr)
	}

	// This produces a valid TCPAddr, which also remotely starts the handler on the implant.
	ln, err = newListenerTCP(addr, route)
	if err != nil {
		return nil, fmt.Errorf("comm listener: %s", err.Error())
	}

	return ln, nil
}

// tcpListener - An abstract tcpListener tied to an implant and listening for any incoming connection.
type tcpListener struct {
	info     *sliverpb.Handler
	sess     *core.Session
	pending  chan net.Conn // Processed streams, out as net.Conn
	errClose chan error
	addr     net.Addr
}

// newListenerTCP - Creates a new listener tied to an ID, with complete network/address information.
// This listener is tied to an implant listener, and the latter feeds the former with connections
// that are routed back to the server. Automatically starts/stops the remote actual listener.
func newListenerTCP(uri *url.URL, route *Route) (*tcpListener, error) {

	id, _ := uuid.NewGen().NewV1()
	ip := net.ParseIP(uri.Hostname())
	port, _ := strconv.Atoi(uri.Port())

	// Listener object.
	ln := &tcpListener{
		info: &sliverpb.Handler{
			ID:        id.String(),
			Type:      sliverpb.HandlerType_Reverse,
			Transport: sliverpb.TransportProtocol_TCP,
			LHost:     uri.Hostname(),
			LPort:     int32(port),
		},
		sess:     route.Gateway,
		pending:  make(chan net.Conn, 100),
		errClose: make(chan error, 1),
		addr:     &net.TCPAddr{IP: ip, Port: port},
	}

	// Application Protocol
	switch uri.Scheme {
	case "mtls":
		ln.info.Application = sliverpb.ApplicationProtocol_MTLS
	case "http":
		ln.info.Application = sliverpb.ApplicationProtocol_HTTP
	case "https":
		ln.info.Application = sliverpb.ApplicationProtocol_HTTPS
	case "socks5":
		ln.info.Application = sliverpb.ApplicationProtocol_Socks5
	case "pipe":
		ln.info.Application = sliverpb.ApplicationProtocol_NamedPipe
	}

	// We first register the abstracted listener: we don't know how fast the implant
	// comm might send us a connection back. We deregister if failure.
	listenersTCP.Add(ln)

	// Forge request to implant.
	req := &sliverpb.HandlerStartReq{
		Handler: ln.info,
		Request: &commonpb.Request{SessionID: route.Gateway.ID},
	}

	// We request to the implant to start its listener.
	res := &sliverpb.HandlerStart{}
	err := remoteHandlerRequest(route.Gateway, req, res)
	if err != nil {
		listenersTCP.Remove(ln.info.ID)
		return nil, fmt.Errorf("comm listener RPC error: %s", err.Error())
	}
	if !res.Success {
		listenersTCP.Remove(ln.info.ID)
		return nil, fmt.Errorf("comm listener: %s", res.Response.Err)
	}

	return ln, nil
}

// Accept - Implements net.Listener Accept(), by providing connections from the routing system.
func (t *tcpListener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-t.pending:
	case err, ok = <-t.errClose:
		if !ok {
			if err.Error() == "closed" {
				err = errors.New("accept on closed listener")
				return
			}
			return
		}
	}
	return
}

// Addr - Implements net.Listener Addr().
func (t *tcpListener) Addr() net.Addr {
	return t.addr
}

// Close - Implements net.Listener Close(), and closes the remote handler working on the implant.
func (t *tcpListener) Close() (err error) {

	// Does not accept more connections and notify callers the listener is closed.
	close(t.pending)
	t.errClose <- errors.New("closed")

	// Close all pending connections
	for cc := range t.pending {
		err = cc.Close()
		if err != nil {
			rLog.Errorf("Listener (ID: %s) failed to close pending connection: %s",
				t.info.ID, err.Error())
		}
	}

	// Send request to implant to close the actual listener, and log errors.
	if t.sess != nil {
		lnReq := &sliverpb.HandlerCloseReq{
			Handler: t.info,
			Request: &commonpb.Request{SessionID: t.sess.ID},
		}
		lnRes := &sliverpb.HandlerClose{}
		err = remoteHandlerRequest(t.sess, lnReq, lnRes)
		if err != nil {
			rLog.Errorf("Listener (ID: %s) failed to close its remote peer (RPC error): %s",
				t.info.ID, err.Error())
		}
		if !lnRes.Success {
			rLog.Errorf("Listener (ID: %s) failed to close its remote peer: %s",
				t.info.ID, err.Error())
		}

	}
	// Remove from trackers map
	listenersTCP.Remove(t.info.ID)
	return
}

var (
	// listenersTCP - All instantiated active connection listenersTCP. These listenersTCP
	// are either tied to a listener running on an implant host, or the C2 server.
	listenersTCP = &tcpListeners{
		active: map[string]*tcpListener{},
		mutex:  &sync.RWMutex{},
	}
)

type tcpListeners struct {
	active map[string]*tcpListener
	mutex  *sync.RWMutex
}

// Get - Get a session by ID
func (c *tcpListeners) Get(id string) *tcpListener {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.active[id]
}

// Add - Add a sliver to the hive (atomically)
func (c *tcpListeners) Add(l *tcpListener) *tcpListener {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.active[l.info.ID] = l
	return l
}

// Remove - Remove a sliver from the hive (atomically)
func (c *tcpListeners) Remove(id string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	l := c.active[id]
	if l != nil {
		delete(c.active, id)
	}
}
