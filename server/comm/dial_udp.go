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
	"net/url"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

// DialUDP - Get a UDP connection to a host reachable anywhere either on the server's interfaces,
// or of any implant's if there is an active route serving it. Valid networks are "udp", "udp4" and "udp6".
func DialUDP(network, host string) (conn net.PacketConn, err error) {

	addr, err := url.Parse(fmt.Sprintf("%s://%s", network, host))
	if err != nil {
		return nil, fmt.Errorf("could not parse URL: %s://%s", network, host)
	}

	// At this level, we need a port, we don't intend to contact the application layer of a URL.
	if addr.Port() == "" || addr.Port() == "0" {
		return nil, fmt.Errorf("invalid port number (nil or 0)")
	}

	// Resolve the route
	route, err := ResolveURL(addr)
	if err != nil {
		return nil, fmt.Errorf("Address lookup failed: %s", err.Error())
	}

	// If no route and no error, dial on the server's interfaces.
	if route == nil {
		raddr, err := net.ResolveUDPAddr(network, host)
		if err != nil {
			return nil, err
		}
		return net.DialUDP(network, nil, raddr)
	}

	cc := newConnOutboundUDP(addr)
	cc.ID = route.ID.String()

	// No timeouts other than the OS-level timeouts.
	ctx := context.Background()

	return route.comm.dialUDP(ctx, cc)
}

// dialContextUDP - Get a working UDP (packet) connection to a destination host reachable from the implant.
func (comm *Comm) dialContextUDP(ctx context.Context, network, host string) (conn net.PacketConn, err error) {

	// Get RHost/RPort
	uri, _ := url.Parse(fmt.Sprintf("%s://%s", network, host))
	if uri == nil {
		return nil, fmt.Errorf("Address parsing failed: %s", host)
	}

	info := newConnOutboundUDP(uri)      // Prepare connection info.
	info.ID = strconv.Itoa(int(comm.ID)) // The comm is in itself a route, so we give its ID, just in case.

	// Do the actual work in the function returned here.
	return comm.dialUDP(ctx, info)
}

// dialUDP - Get a working UDP (packet) connection to a destination host reachable from the implant, with a context.
// Here we directly pass the connection info. Remarks on passed context are the same as comm.dialContextTCP() above.
func (comm *Comm) dialUDP(ctx context.Context, info *commpb.Conn) (conn net.PacketConn, err error) {

	// Normally the context is never nil, but just in case.
	if ctx == nil {
		ctx = context.Background()
	}

	// The timeout is passed as info for the implant dialer to set the OS-level timeout of the connection.
	if deadline, exists := ctx.Deadline(); exists {
		info.Timeout = time.Until(deadline).Milliseconds()
	}

	// We make a UDP listener, because it implements the net.PacketConn interface and that
	// the direction of the connection does not matter (due to it being packet-oriented).
	lip := net.ParseIP(info.LHost)
	rip := net.ParseIP(info.RHost)
	cc := &udpConn{
		info: &commpb.Handler{
			ID:        info.ID,
			Type:      commpb.HandlerType_Bind,
			Transport: commpb.Transport_UDP,
			LHost:     info.LHost,
			LPort:     info.LPort,
			RHost:     info.RHost,
			RPort:     info.RPort,
		},
		comm:    comm,
		pending: make(chan io.ReadWriteCloser),
		laddr:   &net.UDPAddr{IP: lip, Port: int(info.LPort)},
		raddr:   &net.UDPAddr{IP: rip, Port: int(info.RPort)}, // We are dialing a dest
	}

	// We'll either get an error from opening a connection on the implant, or a working stream.
	errOpen := make(chan error, 1)
	err = fmt.Errorf("Failed to dial %s://%s: ", "udp", fmt.Sprintf("%s:%d", info.RHost, info.RPort))

	// Get a working channel (io.ReadWriteCloser) from the implant Comm SSH, or an error
	go func(info *commpb.Conn) {
		data, _ := proto.Marshal(info)
		stream, reqs, err := comm.sshConn.OpenChannel(commpb.Request_RouteConn.String(), data)
		if err != nil {
			errOpen <- err
			return
		}
		go ssh.DiscardRequests(reqs)

		// Pass the stream to be used by the UDP listener (as PacketConn)
		cc.pending <- stream
	}(info)

	select {
	// We receive either a context timeout or a cancellation before a stream.
	case <-ctx.Done():
		switch ctx.Err() {
		case context.Canceled:
			err = errors.WithMessage(err, "context cancelled")
		case context.DeadlineExceeded:
			err = errors.WithMessage(err, "context timeout exceeded")
		}
		return nil, err

	// An error is thrown by the implant, close the pending channel and return no conn.
	case openErr := <-errOpen:
		err = errors.WithMessage(err, openErr.Error())
		return nil, err

	// Or we get the stream before timeout/cancel, wrap it into a UDP decoder
	case connection := <-cc.pending:
		cc.stream = &udpStream{
			r: gob.NewDecoder(connection),
			w: gob.NewEncoder(connection),
			c: connection,
		}

		comm.active = append(comm.active, connection)

		// rLog.Infof("[route] Dialing (%s/%s) %s --> %s (ID: %s)", info.Transport.String(),
		// info.Application.String(), conn.LocalAddr().String(), conn.RemoteAddr().String(), info.ID)
		return cc, nil
	}
}

// dialClientUDP - A client console is making use of its UDP proxy to route some traffic.
// As we don't need a PacketConn (the Comm API is not used here), this function wires the client up.
// (In the end this function means a given Comm can find another Comm and route its connection through it)
func dialClientUDP(info *commpb.Conn, ch ssh.NewChannel) error {

	// If not found, check routes: return if not found,
	// as portfwds and proxies are supposedly not allowed
	// to contact on the server's interfaces.
	hostPort := fmt.Sprintf("%s:%d", info.RHost, info.RPort)
	route, err := ResolveAddress(hostPort)
	if err != nil || route == nil {
		err := ch.Reject(ssh.Prohibited, "NOROUTE")
		if err != nil {
			rLog.Errorf("Error: rejecting UDP stream: %s", err)
		}
		return fmt.Errorf("rejected Client Comm UDP connection: (bad destination: %s)", hostPort)
	}

	// Add the ID of the route we resolved
	info.ID = route.ID.String()
	data, _ := proto.Marshal(info)

	// We don't pass through the comm DialUDP() method because we don't need a PacketConn here.
	dst, dReqs, err := route.comm.sshConn.OpenChannel(commpb.Request_RouteConn.String(), data)
	if err != nil {
		err = ch.Reject(ssh.ConnectionFailed, err.Error())
		if err != nil {
			rLog.Errorf("Error rejecting UDP connection: %s", err.Error())
		}
		return fmt.Errorf("rejected Client Comm UDP connection: %s", err.Error())
	}
	go ssh.DiscardRequests(dReqs)

	// Accept the Comm Client stream
	src, sReqs, err := ch.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept stream (%s)", string(ch.ExtraData()))
	}
	go ssh.DiscardRequests(sReqs)

	// Pipe UDP connections. Blocking
	err = transportPacketConn(src, dst)
	if err != nil {
		rLog.Warnf("Error transporting UDP connection (%s --> %s:%d): %v",
			hostPort, info.LHost, info.LPort, err)
	}

	// Close connections once we're done, with a delay left so our
	// custom RPC tunnel has time to transmit the remaining data.
	closeConnections(src, dst)

	rLog.Infof("[route] Closed UDP stream %s:%d --> %s:%d",
		info.LHost, info.LPort, info.RHost, info.RPort)

	return nil
}
