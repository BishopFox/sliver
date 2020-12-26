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

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
)

// listener - An abstract listener tied to a chain of implant nodes and listen for any incoming connection.
// The remote listener is always tied to a network route, and when it starts listening it inits all nodes
// with appropriate information.
// A listener may be used directly as an abstracted listener by other components, because it implements
// the net.Listener interface. It is thus useful for getting what is treated as a generic connection.
type listener struct {
	id       string // ID passed by the handler
	network  string
	host     string
	port     int
	pending  chan net.Conn // Processed streams, out as net.Conn
	errClose chan error
	addr     net.Addr
}

// newListener - Creates a new tracker tied to an ID, and with basic network/address information.
func newListener(uri *url.URL) *listener {

	id, _ := uuid.NewGen().NewV1() // New route always has a new UUID.
	rln := &listener{
		id:       id.String(),
		network:  uri.Scheme,
		host:     uri.Hostname(),
		pending:  make(chan net.Conn, 100),
		errClose: make(chan error, 1),
	}
	ip := net.ParseIP(uri.Hostname())
	rln.port, _ = strconv.Atoi(uri.Port())

	switch uri.Scheme {
	case "tcp", "mtls", "http", "https", "h2", "socks", "socks5":
		rln.addr = &net.TCPAddr{IP: ip, Port: rln.port}
	case "udp", "dns", "named_pipe":
		rln.addr = &net.UDPAddr{IP: ip, Port: rln.port}
	case "unix":
		rln.addr = &net.UnixAddr{Net: uri.Scheme, Name: uri.Hostname() + uri.Path}
	default:
		rln.addr = &net.TCPAddr{IP: ip, Port: rln.port}
	}
	return rln
}

// Accept - Implements net.Listener Accept(), by providing connections from the routing system.
func (t *listener) Accept() (conn net.Conn, err error) {
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

// Close - Implements net.Listener Close()
func (t *listener) Close() (err error) {
	close(t.pending)            // Does not accept more connections
	for cc := range t.pending { // Close all pending connections
		err = cc.Close()
		if err != nil {
			rLog.Errorf("Tracker (ID: %s) failed to close pending connection: %s", t.id, err.Error())
		}
	}
	t.errClose <- errors.New("closed") // Notify listener is closed.
	listeners.Remove(t.id)             // Remove from trackers map
	return
}

// Addr - Implements net.Listener Addr().
func (t *listener) Addr() net.Addr {
	return t.addr
}

// Listen - Returns a listener started on a valid network address anywhere in either the server interfaces,
// or any implant's interface if the latter is served by an active route. Valid networks are "tcp" and "udp".
func Listen(network, host string) (ln net.Listener, err error) {

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

	switch network {
	case "tcp":
		// No route, use server interfaces
		if route == nil {
			return net.Listen("tcp", host)
		}
		// Else the comm will do its job with routes and implants
		return ListenTCP(network, host)

	case "udp":
		// No route, use server interfaces
		if route == nil {
			return net.Listen("udp", host)
		}
	default:
		return nil, errors.New("server listener: invalid protocol used")
	}

	return
}

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

	// This produces a valid TCPAddr
	tcpLn := newListener(addr)
	port, _ := strconv.Atoi(addr.Port())

	// No route, use server interfaces, just use the address of the tcpLn template
	if route == nil {
		return net.ListenTCP("tcp", tcpLn.Addr().(*net.TCPAddr))
	}

	// Forge request to implant.
	lnReq := &sliverpb.HandlerStartReq{
		Handler: &sliverpb.Handler{
			Transport: sliverpb.TransportProtocol_TCP,
			ID:        tcpLn.id,
			LHost:     addr.Hostname(),
			LPort:     int32(port),
		},
		Request: &commonpb.Request{SessionID: route.Gateway.ID},
	}

	// We first register the abstracted listener, because we don't know how fast
	// the implant comm might send us a connection back. We deregister if failure.
	listeners.Add(tcpLn)

	// We request to the implant to start its listener.
	lnRes := &sliverpb.HandlerStart{}
	err = remoteHandlerRequest(route.Gateway, lnReq, lnRes)
	if err != nil {
		listeners.Remove(tcpLn.id)
		return nil, fmt.Errorf("comm listener RPC error: %s", err.Error())
	}
	if !lnRes.Success {
		listeners.Remove(tcpLn.id)
		return nil, fmt.Errorf("comm listener: %s", lnRes.Response.Err)
	}

	// Return the tcp listener, which will be fed by the comm system.
	return tcpLn, nil
}

// remoteHandlerRequest - Send a protobuf request to gateway session and get response.
func remoteHandlerRequest(sess *core.Session, req, resp proto.Message) (err error) {
	reqData, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	data, err := sess.Request(sliverpb.MsgNumber(req), defaultNetTimeout, reqData)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return err
	}
	return nil
}
