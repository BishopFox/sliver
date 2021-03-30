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
	"io"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
)

// udpDialer - An objet that dials a single UDP address and writes all its routed
// traffic to it, no matter the src address of the packets. Therefore, this handler
// is hardly ever used alone, except when we want to target specific addresses for
// a specific connection, like routing a QUIC-transported implant.
//
// This handler should therefore not be used alone for pure UDP port forwarding.
type reverseForwarderUDP struct {
	// Base
	info      *commpb.Handler
	sessionID uint32
	pending   chan io.ReadWriteCloser
	log       *logrus.Entry
	// UDP
	inbound  *udpStream
	outbound *udpConns
	sent     int64
	recv     int64
}

// newDialerUDP - Creates UDP handler that is connected to an addess:port on the
// implant's host network. If specified, we dial with given source host:ports as well.
func newReverseForwarderUDP(info *commpb.Handler) (f *reverseForwarderUDP, err error) {

	// Handler with a working conn, and wired up to the Comms.
	f = &reverseForwarderUDP{
		info: info,
		outbound: &udpConns{
			m: map[string]*udpConn{},
		},
		pending: make(chan io.ReadWriteCloser),
		log:     ClientComm.Log.WithField("comm", "portfwd"),
	}
	id, _ := uuid.NewGen().NewV1()
	f.info.ID = id.String()
	f.info.Type = commpb.HandlerType_Reverse
	f.info.Transport = commpb.Transport_UDP

	// We add the forwarder now, because it must be ready to handle its dedicated (incoming) stream.
	Forwarders.Add(f)

	return
}

// Info - Implements Forwarder Info()
func (f *reverseForwarderUDP) Info() *commpb.Handler {
	return f.info
}

// SessionID- Implements Forwarder SessionID()
func (f *reverseForwarderUDP) SessionID() uint32 {
	return f.sessionID
}

// ConnStats - Implements Forwarder ConnStats()
func (f *reverseForwarderUDP) ConnStats() string {
	return fmt.Sprintf("%d sent / %d recv", f.sent, f.recv)
}

// LocalAddr - Implements Forwarder LocalAddr()
func (f *reverseForwarderUDP) LocalAddr() string {
	return ""
}

// handleReverse - Pass a single stream given by the Comm system, in which we pass UDP traffic.
func (f *reverseForwarderUDP) handleReverse(ch ssh.NewChannel) {

	stream, reqs, err := ch.Accept()
	if err != nil {
		ch.Reject(ssh.ConnectionFailed, "")
	}
	go ssh.DiscardRequests(reqs)

	f.pending <- stream
}

// Implements Forwarder serve()
func (f *reverseForwarderUDP) serve() {

	// We simply "dial" the destination as if we were to write,
	// and this will also start reading from the destination as well.
	go f.handleWrite()
}

// handleRead - Read all UDP packets coming at the host address:port
// and route (encode) them back to the server.
func (f *reverseForwarderUDP) handleRead(p *udpPacket, conn *udpConn) error {
	const maxMTU = 9012
	buff := make([]byte, maxMTU)
	for {
		//response must arrive within 15 seconds
		deadline := 15 * time.Second
		conn.SetReadDeadline(time.Now().Add(deadline))

		//read response
		n, err := conn.Read(buff)
		if err != nil {
			if !os.IsTimeout(err) && err != io.EOF {
			}
			break
		}
		b := buff[:n]

		//encode back over ssh connection
		err = f.inbound.encode(p.Src, b)
		if err != nil {
			return err
		}
		//stats
		atomic.AddInt64(&f.sent, int64(n))
	}
	return nil
}

// handleWrite - Reads all packets forwarded by the server and write them to the UDP conn.
func (f *reverseForwarderUDP) handleWrite() error {
	for {
		// Decode a UDP packet
		packet := &udpPacket{}
		if err := f.inbound.r.Decode(&packet); err != nil {
			return err
		}

		// Dial the address or get the current UDP connection to it
		hostPort := fmt.Sprintf("%s:%d", f.info.LHost, f.info.LPort)
		conn, exists, err := f.outbound.dial(packet.Src, hostPort)
		if err != nil {
			return fmt.Errorf("failed to dial UDP address %s: %s", hostPort, err.Error())
		}

		const maxConns = 100
		if !exists {
			if f.outbound.len() <= maxConns {
				go f.handleRead(packet, conn)
			} else {
				// {{if .Config.Debug}}
				log.Printf("exceeded max udp connections (%d)", maxConns)
				// {{end}}
			}
		}
		n, err := conn.Write(packet.Payload)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Done writing to UDP %s", hostPort)
			// {{end}}
			return err
		}

		//stats
		atomic.AddInt64(&f.recv, int64(n))
	}
}

// Close - Close the port forwarder, and optionally connections for TCP forwarders
func (f *reverseForwarderUDP) Close(activeConns bool) (err error) {

	// Close the UDP connection on the client
	f.outbound.closeAll()

	// Close the listener's Comm channel
	f.inbound.c.Close()

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
