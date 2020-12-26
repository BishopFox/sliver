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
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ReadFrom - Implements PacketConn ReadFrom(). Parses a UDP address and encodes the packet
// to the Comm stream bound to this UDP listener/connection.
// It returns the number of bytes read (0 <= n <= len(p))
// and any error encountered. Callers should always process
// the n > 0 bytes returned before considering the error err.
// ReadFrom can be made to time out and return an error after a
// fixed time limit; see SetDeadline and SetReadDeadline.
func (ln *udpListener) ReadFrom(p []byte) (n int, addr net.Addr, err error) {

	// Wait for a packet from the Comm stream
	packet := &udpPacket{}
	ln.stream.decode(packet)

	// Forge source addr for this packet
	host := strings.Split(packet.Src, ":")[0]
	sPort := strings.Split(packet.Src, ":")[1]
	ip := net.ParseIP(host)
	port, err := strconv.Atoi(sPort)
	if err != nil {
		rLog.Errorf("Dropped packet because of malformed source header: %s", packet.Src)
	}
	addr = &net.UDPAddr{IP: ip, Port: port}

	// Copy payload in buffer.
	copy(p[:], packet.Payload)

	return len(packet.Payload), addr, nil
}

// WriteTo writes a packet with payload p to addr.
// WriteTo can be made to time out and return an Error after a
// fixed time limit; see SetDeadline and SetWriteDeadline.
// On packet-oriented connections, write timeouts are rare.
func (ln *udpListener) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	err = ln.stream.encode(addr.String(), p)
	if err != nil {
		return 0, fmt.Errorf("failed to encode UDP packet to stream: %v", err)
	}
	return len(p), nil
}

// Close closes the connection.
// Any blocked ReadFrom or WriteTo operations will be unblocked and return errors.
// This function also closes the Comm stream that wires the abstract and the real listener.
func (ln *udpListener) Close() error {
	// Notify callers that the listener is closed.

	// Send request to implant to close the actual listener, and log errors.
	if ln.sess != nil {
		lnReq := &sliverpb.HandlerCloseReq{
			Handler: ln.info,
			Request: &commonpb.Request{SessionID: ln.sess.ID},
		}
		lnRes := &sliverpb.HandlerClose{}
		err := remoteHandlerRequest(ln.sess, lnReq, lnRes)
		if err != nil {
			rLog.Errorf("Listener (ID: %s) failed to close its remote peer (RPC error): %s", ln.info.ID, err.Error())
		}
		if !lnRes.Success {
			rLog.Errorf("Listener (ID: %s) failed to close its remote peer: %s", ln.info.ID, err.Error())
		}

	}

	// Close Comm stream.
	err := ln.stream.c.Close()
	if err != nil {
		return err
	}

	return nil
}

// LocalAddr returns the local network address.
func (ln *udpListener) LocalAddr() (addr net.Addr) {
	return ln.addr
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail instead of blocking. The deadline applies to all future
// and pending I/O, not just the immediately following call to
// Read or Write. After a deadline has been exceeded, the
// connection can be refreshed by setting a deadline in the future.
//
// If the deadline is exceeded a call to Read or Write or to other
// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
// The error's Timeout method will return true, but note that there
// are other possible errors for which the Timeout method will
// return true even if the deadline has not been exceeded.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful ReadFrom or WriteTo calls.
//
// A zero value for t means I/O operations will not time out.
func (ln *udpListener) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline sets the deadline for future ReadFrom calls
// and any currently-blocked ReadFrom call.
// A zero value for t means ReadFrom will not time out.
func (ln *udpListener) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline sets the deadline for future WriteTo calls
// and any currently-blocked WriteTo call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means WriteTo will not time out.
func (ln *udpListener) SetWriteDeadline(t time.Time) error {
	return nil
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
