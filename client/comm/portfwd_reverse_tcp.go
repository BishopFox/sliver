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

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
)

// reverseForwarderTCP - An abstract reverseForwarderTCP tied
// to an implant and listening for any incoming connection.
type reverseForwarderTCP struct {
	info        *commpb.Handler
	sessionID   uint32
	pending     chan ssh.NewChannel
	connections map[string]net.Conn
}

// newReverseForwarderTCP - Creates a new listener tied to an ID, and with basic network/address information.
// This listener is tied to an implant listener, and the latter feeds the former with connections
// that are routed back to the server. Also able to stop the remote listener like server jobs.
func newReverseForwarderTCP(info *commpb.Handler) (f *reverseForwarderTCP, err error) {

	// Listener object, kept for printing current port forwards.
	f = &reverseForwarderTCP{
		info:        info,
		pending:     make(chan ssh.NewChannel),
		connections: map[string]net.Conn{},
	}

	id, _ := uuid.NewGen().NewV1()
	f.info.ID = id.String()
	f.info.Type = commpb.HandlerType_Reverse
	f.info.Transport = commpb.Transport_TCP

	// If no local host, assume 0.0.0.0 on client
	localIP := net.ParseIP(info.LHost)
	if localIP == nil && info.LHost == "" {
		localIP = net.ParseIP("0.0.0.0")
	}
	if localIP == nil {
		return nil, fmt.Errorf("Could not parse local host address: %s", info.LHost)
	}

	// We first register the abstracted listener: we don't know how fast the implant
	// comm might send us a connection back. We deregister if failure.
	Forwarders.Add(f)

	return f, nil
}

// Info - Implements Forwarder Info()
func (f *reverseForwarderTCP) Info() *commpb.Handler {
	return f.info
}

// SessionID- Implements Forwarder SessionID()
func (f *reverseForwarderTCP) SessionID() uint32 {
	return f.sessionID
}

// ConnStats - Implements Forwarder ConnStats()
func (f *reverseForwarderTCP) ConnStats() string {
	return fmt.Sprintf("%d conns", len(f.connections))
}

// LocalAddr - Implements Forwarder LocalAddr()
func (f *reverseForwarderTCP) LocalAddr() string {
	return ""
}

// handleReverse - Implements Forwarder handleReverse(). Not used for direct TCP forwarding.
func (f *reverseForwarderTCP) handleReverse(ch ssh.NewChannel) {
	f.pending <- ch // Non-blocking
	return
}

// serve - Implements Forwarder serve()
func (f *reverseForwarderTCP) serve() {

	for {
		// For each connection,
		ch := <-f.pending
		info := &commpb.Conn{}
		proto.Unmarshal(ch.ExtraData(), info)

		// dial on the console host and pipe the connection
		go func() {
			// Dial TCP destination
			dst, err := net.Dial("tcp", fmt.Sprintf("%s:%d", f.info.LHost, f.info.LPort))
			if err != nil {
				ch.Reject(ssh.ConnectionFailed, "")
			}

			// Get SSH stream
			src, reqs, err := ch.Accept()
			if err != nil {
				ch.Reject(ssh.ConnectionFailed, "")
			}
			go ssh.DiscardRequests(reqs)

			// Add connection to active
			f.connections[dst.LocalAddr().String()+dst.RemoteAddr().String()] = dst

			// Pipe connections
			transportConn(src, dst)

			// Close connections once we're done
			closeConnections(src, dst)

			// Remove connection from list, if still there
			if dst != nil {
				delete(f.connections, dst.LocalAddr().String()+dst.RemoteAddr().String())
			}
		}()
	}
}

// Close - For reverse forwarders, delete the forwarder server-side, and
// close the listener on the implant. Optionally closes active connections.
func (f *reverseForwarderTCP) Close(activeConns bool) (err error) {

	// Close listener on the implant, and its mapping on the server.
	req := &commpb.PortfwdCloseReq{
		Handler: f.info,
		Request: &commonpb.Request{SessionID: f.sessionID},
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
	Forwarders.Remove(f.info.ID)

	// Close connections if asked so
	if activeConns {
		for _, conn := range f.connections {
			conn.Close()
		}
	}

	return
}
