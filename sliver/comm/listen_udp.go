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

	"encoding/gob"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ListenUDP - Starts a UDP listener (only slightly different behavior from a UDP handler).
func ListenUDP(handler *sliverpb.Handler) error {

	// Get a working stream from the comm system. We transmit data over it.
	stream := Comms.server.getStream(handler)

	// Create the UDP stream to wire the ReadWriteCloser and the UDP listener
	udpStream := &udpStream{
		r: gob.NewDecoder(stream),
		w: gob.NewEncoder(stream),
		c: stream,
	}

	// Create a new UDP listener: this starts listening on the host.
	ln, err := newUDPListener(fmt.Sprintf("%s:%d", handler.LHost, handler.LPort), udpStream)
	if err != nil {
		return err
	}

	// Add listener to jobs
	listener := newPacketListener(handler, ln.inbound)
	Listeners.Add(listener)

	// Start processing packets coming from the host and route them back to server.
	go ln.handleWrite()

	// Only then we write the data coming from the server,
	// as we're ready to get the response.
	go ln.handleRead()

	return nil
}

// udpListener - A slightly modified udpHandler, which is used for reverse UDP portforwarding,
// or reverse UDP handlers in general. The listener has only one "UDP Conn", which in fact is
// just a UDP listener which can receive packets from any source address.
type udpListener struct {
	hostPort string
	inbound  *net.UDPConn
	stream   *udpStream
	sent     int64
	recv     int64
}

// newUDPListener - A listener wired up to the Comm system. Listens on the host for UDP packets.
func newUDPListener(hostPort string, stream *udpStream) (ln *udpListener, err error) {

	// Check the UDP address is valid
	a, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		return nil, fmt.Errorf("resolve: %s", err)
	}
	// Listen on it.
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return nil, fmt.Errorf("listen: %s", err)
	}
	// Create the listener
	ln = &udpListener{
		hostPort: hostPort,
		inbound:  conn,
		stream:   stream,
	}

	return
}

// handleRead - Reads all packets forwarded by the server and write them to the UDP listener.
func (ln *udpListener) handleRead() error {
	for {
		//receive from channel, including source address
		p := udpPacket{}
		if err := ln.stream.decode(&p); err == io.EOF {
			// {{if .Config.Debug}}
			log.Printf("decode error: %v", err)
			// {{end}}

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
		n, err := ln.inbound.WriteToUDP(p.Payload, addr)
		if err != nil {
			return fmt.Errorf("write error: %w", err)
		}

		//stats
		atomic.AddInt64(&ln.recv, int64(n))
	}
}

// handleWrite - Read all UDP packets coming at the host address:port and encode them back to the server.
func (ln *udpListener) handleWrite() error {
	const maxMTU = 9012
	buff := make([]byte, maxMTU)
	for {
		// Read with deadline
		ln.inbound.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := ln.inbound.ReadFromUDP(buff)
		if e, ok := err.(net.Error); ok && (e.Timeout() || e.Temporary()) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		//send over channel, including source address
		b := buff[:n]
		if err := ln.stream.encode(addr.String(), b); err != nil {
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
