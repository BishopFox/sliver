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
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
)

// directForwarderUDP - An object similar to Sliver's comm package "udpListener", slightly
// modified so that it acts as a UDP direct port forwarder between the client console and an implant.
type directForwarderUDP struct {
	// Base
	info      *commpb.Handler
	ctx       context.Context    // Gives us a cancel function
	closed    context.CancelFunc // Notify goroutines the forwader is closed
	mutex     sync.Mutex
	sessionID uint32
	// Direct
	localAddr string
	// UDP
	inbound  *net.UDPConn // host <-> server
	outbound *udpStream   // server <-> (implant)
	sent     int64
	recv     int64
}

// newDirectForwarderUDP - Creates a new listener tied to an ID, and with basic network/address information.
// This listener is tied to an implant listener, and the latter feeds the former with connections
// that are routed back to the server. Also able to stop the remote listener like server jobs.
func newDirectForwarderUDP(info *commpb.Handler, lhost string, lport int) (f *directForwarderUDP, err error) {

	// Listener object, kept for printing current port forwards.
	f = &directForwarderUDP{
		info:  info,
		mutex: sync.Mutex{},
	}
	// Additional fields
	id, _ := uuid.NewGen().NewV1()
	f.info.ID = id.String()
	f.info.Type = commpb.HandlerType_Bind
	f.info.Transport = commpb.Transport_UDP
	f.localAddr = fmt.Sprintf("%s:%d", lhost, lport)

	// Context
	f.ctx, f.closed = context.WithCancel(context.Background())

	// Check the UDP address is valid
	a, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", lhost, lport))
	if err != nil {
		return nil, fmt.Errorf("resolve: %s", err)
	}
	// Listen on it.
	f.inbound, err = net.ListenUDP("udp", a)
	if err != nil {
		return nil, fmt.Errorf("listen: %s", err)
	}

	// We first register the abstracted listener: we don't know how fast the implant
	// comm might send us a connection back. We deregister if failure.
	Forwarders.Add(f)

	return f, nil
}

// Info - Implements Forwarder Info()
func (f *directForwarderUDP) Info() *commpb.Handler {
	return f.info
}

// SessionID- Implements Forwarder SessionID()
func (f *directForwarderUDP) SessionID() uint32 {
	return f.sessionID
}

// ConnStats - Implements Forwarder ConnStats()
func (f *directForwarderUDP) ConnStats() string {
	return fmt.Sprintf("%d sent / %d recv", f.sent, f.recv)
}

// LocalAddr - Implements Forwarder LocalAddr()
func (f *directForwarderUDP) LocalAddr() string {
	return f.localAddr
}

// handleReverse - Pass a single stream given by the Comm system, in which we pass UDP traffic.
func (f *directForwarderUDP) handleReverse(ch ssh.NewChannel) { return }

// serve - Implements Forwarder serve()
func (f *directForwarderUDP) serve() {

	// Start processing packets coming from the host and route them back to server.
	go f.handleWrite()

	// Only then we write the data coming from the server,
	// as we're ready to get the response.
	go f.handleRead()
}

// handleRead - Reads all packets forwarded by the server and write them to the UDP listener.
func (f *directForwarderUDP) handleRead() error {
	for !isDone(f.ctx) {
		// Get a SSH channel and bind it to the forwarder
		uc, err := f.getStreamUDP()
		if err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				continue
			}
		}

		//receive from channel, including source address
		p := udpPacket{}
		if err := uc.decode(&p); err == io.EOF {
			// We don't care about end of file.
			continue

			// return err
		} else if err != nil {
			return fmt.Errorf("decode error: %w", err)
		}

		//write back to inbound udp
		addr, err := net.ResolveUDPAddr("udp", p.Src)
		if err != nil {
			return fmt.Errorf("resolve error: %w", err)
		}
		n, err := f.inbound.WriteToUDP(p.Payload, addr)
		if err != nil {
			return fmt.Errorf("write error: %w", err)
		}

		//stats
		atomic.AddInt64(&f.recv, int64(n))
	}
	return nil
}

// handleWrite - Read all UDP packets coming at the host address:port and encode them back to the server.
func (f *directForwarderUDP) handleWrite() error {
	const maxMTU = 9012
	buff := make([]byte, maxMTU)

	for !isDone(f.ctx) {
		// Read with deadline
		f.inbound.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := f.inbound.ReadFromUDP(buff)
		if e, ok := err.(net.Error); ok && (e.Timeout() || e.Temporary()) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		// Get a SSH channel and bind it to the forwarder
		uc, err := f.getStreamUDP()
		if err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				continue
			}
			fmt.Printf("Error")
		}

		//send over channel, including source address
		b := buff[:n]
		if err := uc.encode(addr.String(), b); err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				continue //dropped packet...
			}
			return fmt.Errorf("encode error: %w", err)
		}
		//stats
		atomic.AddInt64(&f.sent, int64(n))
	}
	return nil
}

// getStreamUDP - Quickely get a stream which we wrap with UDP encoding/decoding
func (f *directForwarderUDP) getStreamUDP() (*udpStream, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.outbound != nil {
		return f.outbound, nil
	}

	port, _ := strconv.Atoi(strings.Split(f.localAddr, ":")[1])

	// The server Comm only handle streams requests when they have
	// a cmmpb.Conn information attached.
	conn := &commpb.Conn{
		ID:          f.info.ID,
		Transport:   f.info.Transport,
		Application: f.info.Application,
		LHost:       strings.Split(f.localAddr, ":")[0],
		LPort:       int32(port),
		RHost:       f.info.RHost,
		RPort:       f.info.RPort,
	}

	data, _ := proto.Marshal(conn)
	stream, reqs, err := ClientComm.ssh.OpenChannel(commpb.Request_PortfwdStream.String(), data)
	if err != nil {
		return nil, fmt.Errorf("Error opening channel: %s", err)
	}
	go ssh.DiscardRequests(reqs)

	f.outbound = &udpStream{
		r: gob.NewDecoder(stream),
		w: gob.NewEncoder(stream),
		c: stream,
	}

	return f.outbound, nil
}

// Close - Close the port forwarder, and optionally connections for TCP forwarders
func (f *directForwarderUDP) Close(activeConns bool) (err error) {

	// Notidy goroutines we are closed
	f.closed()

	// Close the UDP connection (listener) on the client
	f.inbound.Close()

	// Close the UDP channel to the server
	f.outbound.c.Close()

	// Remove the listener mapping on the server
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

	return
}
