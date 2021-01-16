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
	"io"
	"net"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
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

	// Prevent localhost panics
	if addr.Hostname() == "localhost" {
		addr.Host = "127.0.0.1" + ":" + addr.Port()
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
	info     *commpb.Handler
	comm     *Comm
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
		info: &commpb.Handler{
			ID:        id.String(),
			Type:      commpb.HandlerType_Reverse,
			Transport: commpb.Transport_TCP,
			RHost:     uri.Hostname(), // We use RHost, the implant knows (portfwd compatibility)
			RPort:     int32(port),    // We use RPort, the implant knows (portfwd compatibility)
		},
		comm:     route.comm,
		pending:  make(chan net.Conn, 100),
		errClose: make(chan error, 1),
		addr:     &net.TCPAddr{IP: ip, Port: port},
	}

	// Application Protocol
	switch uri.Scheme {
	case "mtls":
		ln.info.Application = commpb.Application_MTLS
	case "http":
		ln.info.Application = commpb.Application_HTTP
	case "https":
		ln.info.Application = commpb.Application_HTTPS
	case "socks5":
		ln.info.Application = commpb.Application_Socks5
	case "pipe":
		ln.info.Application = commpb.Application_NamedPipe
	default:
		ln.info.Application = commpb.Application_None
	}

	// We first register the abstracted listener: we don't know how fast the implant
	// comm might send us a connection back. We deregister if failure.
	listeners.Add(ln)

	// Forge request to implant.
	req := &commpb.HandlerStartReq{
		Handler: ln.info,
		Request: &commonpb.Request{SessionID: route.Gateway.ID},
	}

	// We request to the implant to start its listener.
	res := &commpb.HandlerStart{}
	err := remoteHandlerRequest(route.Gateway, req, res)
	if err != nil {
		listeners.Remove(ln.info.ID)
		return nil, fmt.Errorf("comm listener RPC error: %s", err.Error())
	}
	if !res.Success {
		listeners.Remove(ln.info.ID)
		return nil, fmt.Errorf("comm listener: %s", res.Response.Err)
	}

	return ln, nil
}

// Accept - Implements net.Listener Accept(), by providing connections from the routing system.
func (t *tcpListener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn, ok = <-t.pending:
		if !ok {
			return nil, errors.New("accept on closed listener")
		}
	}
	return conn, nil
}

// Addr - Implements net.Listener Addr().
func (t *tcpListener) Addr() net.Addr {
	return t.addr
}

// Close - Implements net.Listener Close(), and closes the remote handler working on the implant.
func (t *tcpListener) Close() (err error) {

	// Does not accept more connections and
	// notify callers the listener is closed.
	close(t.pending)

	// Close all pending connections
	for cc := range t.pending {
		err = cc.Close()
		if err != nil {
			rLog.Errorf("Listener (ID: %s) failed to close pending connection: %s",
				t.info.ID, err.Error())
		}
	}

	// Send request to implant to close
	// the actual listener, and log errors.
	if t.comm.session != nil {
		lnReq := &commpb.HandlerCloseReq{
			Handler: t.info,
			Request: &commonpb.Request{SessionID: t.comm.session.ID},
		}
		lnRes := &commpb.HandlerClose{}
		err = remoteHandlerRequest(t.comm.session, lnReq, lnRes)
		if err != nil {
			rLog.Errorf("Listener (ID: %s) failed to close its remote peer (RPC error): %s",
				t.info.ID, err.Error())
		}
		if !lnRes.Success {
			rLog.Errorf("Listener (ID: %s) failed to close its remote peer: %s",
				t.info.ID, err.Error())
		}

	}
	// Remove from listeners map
	listeners.Remove(t.info.ID)
	return
}

// handleReverse - The listener is being fed a connection by the implant Comm, so it forges a
// pseudo-net.Conn and pushes it to its channel of pending connections, processed with ln.Accept()
func (t *tcpListener) handleReverse(info *commpb.Conn, ch ssh.NewChannel) error {

	// Accept the stream
	sshChan, reqs, err := ch.Accept()
	if err != nil {
		rLog.Errorf("failed to accept stream (%s)", string(ch.ExtraData()))
		return err
	}
	go ssh.DiscardRequests(reqs)

	// Forge a conn with info (implements net.Conn)
	conn := newConnInboundTCP(info, io.ReadWriteCloser(sshChan))

	// Push the stream to its listener (non-blocking)
	t.pending <- conn
	rLog.Infof("Processed TCP inbound stream: %s <-- %s", conn.lAddr.String(), conn.rAddr.String())

	return nil
}

// base - Implements listener base()
func (t *tcpListener) base() *commpb.Handler {
	return t.info
}

// comms - Implements listener comms()
func (t *tcpListener) comms() *Comm {
	return t.comm
}
