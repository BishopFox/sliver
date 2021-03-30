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
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
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
	conn, err = newListenerUDP(addr, route.comm)
	if err != nil {
		return nil, fmt.Errorf("comm listener: %s", err.Error())
	}

	return
}

// newListenerUDP - Creates a new listener tied to an ID, and with basic network/address information.
// This listener is tied to an implant listener, and the latter feeds the former with connections
// that are routed back to the server. Also able to stop the remote listener like server jobs.
func newListenerUDP(uri *url.URL, comm *Comm) (ln *udpConn, err error) {

	id, _ := uuid.NewGen().NewV1()
	ip := net.ParseIP(uri.Hostname())
	port, _ := strconv.Atoi(uri.Port())

	// Listener object.
	ln = &udpConn{
		info: &commpb.Handler{
			ID:        id.String(),
			Type:      commpb.HandlerType_Reverse,
			LHost:     uri.Hostname(),
			LPort:     int32(port),
			Transport: commpb.Transport_UDP,
		},
		comm:    comm,
		pending: make(chan io.ReadWriteCloser),
		laddr:   &net.UDPAddr{IP: ip, Port: port},
		// No raddr, because we are listening UDP, not dialing.
	}

	// We first register the abstracted listener: we don't know how fast the implant
	// comm might send us a connection back. We deregister if failure.
	listeners.Add(ln)

	// Forge request to implant.
	req := &commpb.HandlerStartReq{
		Handler: ln.info,
		Request: &commonpb.Request{SessionID: comm.session.ID},
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
	res := &commpb.HandlerStart{}
	err = remoteHandlerRequest(comm.session, req, res)
	if err != nil {
		listeners.Remove(ln.info.ID)
		return nil, fmt.Errorf("comm listener RPC error: %s", err.Error())
	}
	if !res.Success {
		listeners.Remove(ln.info.ID)
		return nil, fmt.Errorf("comm listener: %s", res.Response.Err)
	}

	// Wait for the listener to aquire its stream from the Comm system.
	wg.Wait()
	rLog.Infof("Bound UDP inbound stream: server <-- %s", ln.laddr.String())

	return ln, nil
}

// handleReverse - The UDP connection needs to satisfy our listener interface,
// when we listen UDP via the Comm library, or with client reverse port forwarders.
func (c *udpConn) handleReverse(info *commpb.Conn, ch ssh.NewChannel) error {

	// Accept the stream and make it a conn.
	stream, reqs, err := ch.Accept()
	if err != nil {
		rLog.Errorf("failed to accept stream (%s)", string(ch.ExtraData()))
		return err
	}
	go ssh.DiscardRequests(reqs)

	// Push the stream to its listener. This is blocking, because the udpListener is
	// actually just a UDPConn, which needs to be wired up and working.
	c.pending <- stream
	return nil
}

// base - Implements the listener interface.
func (c *udpConn) base() *commpb.Handler {
	return c.info
}

// comms - Implements the listener comms()
func (c *udpConn) comms() *Comm {
	return c.comm
}
