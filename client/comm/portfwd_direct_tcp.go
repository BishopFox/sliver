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
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
)

// directForwarderTCP - An abstract directForwarderTCP tied
// to an implant and listening for any incoming connection.
type directForwarderTCP struct {
	sessionID   uint32
	info        *commpb.Handler
	errClose    chan error
	addr        net.Addr
	inbound     net.Listener
	connections map[string]net.Conn
	localAddr   string
}

// newListenerTCP - Creates a new listener tied to an ID, with complete network/address information.
// This listener is tied to an implant listener, and the latter feeds the former with connections
// that are routed back to the server. Automatically starts/stops the remote actual listener.
func newDirectForwarderTCP(info *commpb.Handler, lhost string, lport int) (f *directForwarderTCP, err error) {

	// Listener object, kept for printing current port forwards.
	f = &directForwarderTCP{
		info:        info,
		connections: map[string]net.Conn{},
	}
	id, _ := uuid.NewGen().NewV1()
	f.info.ID = id.String()
	f.info.Type = commpb.HandlerType_Bind
	f.info.Transport = commpb.Transport_TCP
	f.localAddr = fmt.Sprintf("%s:%d", lhost, lport)

	// If no local host, assume 0.0.0.0 on client
	localIP := net.ParseIP(lhost)
	if localIP == nil && lhost == "" {
		localIP = net.ParseIP("0.0.0.0")
	}
	if localIP == nil {
		return nil, fmt.Errorf("Could not parse local host address: %s", lhost)
	}

	// Check the TCP address is valid
	a, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", lhost, lport))
	if err != nil {
		return nil, fmt.Errorf("resolve: %s", err)
	}
	// Listen on it.
	f.inbound, err = net.ListenTCP("tcp", a)
	if err != nil {
		return nil, fmt.Errorf("listen: %s", err)
	}

	// We first register the abstracted listener: we don't know how fast the implant
	// comm might send us a connection back. We deregister if failure.
	Forwarders.Add(f)

	return
}

// Info - Implements Forwarder Info()
func (f *directForwarderTCP) Info() *commpb.Handler {
	return f.info
}

// SessionID- Implements Forwarder SessionID()
func (f *directForwarderTCP) SessionID() uint32 {
	return f.sessionID
}

// ConnStats - Implements Forwarder ConnStats()
func (f *directForwarderTCP) ConnStats() string {
	return fmt.Sprintf("%d conns", len(f.connections))
}

// LocalAddr - Implements Forwarder LocalAddr()
func (f *directForwarderTCP) LocalAddr() string {
	return f.localAddr
}

// handleReverse - Implements Forwarder handleReverse(). Not used for direct TCP forwarding.
func (f *directForwarderTCP) handleReverse(ch ssh.NewChannel) { return }

// server - For each connection accepted by the listener, concurrently wrap it with info and send it.
// Implements Forwarder serve()
func (f *directForwarderTCP) serve() {
	for {
		conn, err := f.inbound.Accept()
		if err != nil {
			// If the error arises from the accepted connection
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			// Else the error is likely to be a closed listener, so return.
			return
		}
		go f.handle(conn)
	}
}

// handle - Make some info with the connection and its handler, and send through Comm. Server will dispatch.
func (f *directForwarderTCP) handle(conn net.Conn) (err error) {

	uri, _ := url.Parse(fmt.Sprintf("%s://%s", conn.LocalAddr().Network(), conn.LocalAddr().String()))
	if uri == nil {
		return fmt.Errorf("Address parsing failed: %s", uri.String())
	}

	// New connection info with source address added
	info := newConn(f.info, uri)
	info.LHost = strings.Split(conn.RemoteAddr().String(), ":")[0]
	port, _ := strconv.Atoi(strings.Split(conn.RemoteAddr().String(), ":")[1])
	info.LPort = int32(port)
	data, _ := proto.Marshal(info)

	// Create muxed channel and pipe.
	dst, reqs, err := ClientComm.ssh.OpenChannel(commpb.Request_PortfwdStream.String(), data)
	if err != nil {
		return fmt.Errorf("Connection failed: %s", err.Error())
	}
	go ssh.DiscardRequests(reqs)

	// Add connection to active
	f.connections[conn.LocalAddr().String()+conn.RemoteAddr().String()] = conn

	// Pipe
	err = transportConn(conn, dst)
	if err != nil {
		return err
	}

	// Close connections once we're done
	closeConnections(conn, dst)

	// Remove connection from list, if still there
	if conn != nil {
		delete(f.connections, conn.LocalAddr().String()+conn.RemoteAddr().String())
	}

	return nil
}

// Close - Close the port forwarder, and optionally connections for TCP forwarders
func (ln *directForwarderTCP) Close(activeConns bool) (err error) {

	// Close the listener on the client
	err = ln.inbound.Close()
	if err != nil {
		return fmt.Errorf("TCP listener error: %s", err.Error())
	}

	// Remove the listener mapping on the server
	req := &commpb.PortfwdCloseReq{
		Handler: ln.info,
		Request: &commonpb.Request{SessionID: ln.sessionID},
	}
	data, _ := proto.Marshal(req)

	// Send request
	_, resp, err := ClientComm.ssh.SendRequest(commpb.Request_PortfwdStop.String(), true, data)
	if err != nil {
		return fmt.Errorf("Comm error: %s", err.Error())
	}
	res := &commpb.PortfwdClose{}
	proto.Unmarshal(resp, res)

	// The server might give us an error
	if !res.Success {
		return fmt.Errorf("Portfwd error: %s", res.Response.Err)
	}

	// Else remove the forwarder from the map
	Forwarders.Remove(ln.info.ID)

	// Close connections if asked so
	if activeConns && len(ln.connections) > 0 {
		for _, conn := range ln.connections {
			if conn != nil {
				conn.Close()
			}
		}
	}

	return
}
