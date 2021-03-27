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
	"io"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

// ------------------------------------------------------------------------------------------------------------
// Inbound Connections:
// A stream is routed from the implant back to the server, which needs to see it as a normal net.Conn
// ------------------------------------------------------------------------------------------------------------

// tcpConn - An abstracted connection that implements the net.Conn interface.
// The remote and local addresses are set from the payload info attached in the SSH Channel.
type tcpConn struct {
	io.ReadWriteCloser
	lAddr net.Addr
	rAddr net.Addr
}

// newConnInboundTCP can produce more specialized types of connections. Eventually the type returned
// should not be the connection but an interface net.Conn, net.TCPConn, or whatever.
func newConnInboundTCP(info *commpb.Conn, stream io.ReadWriteCloser) *tcpConn {

	lhost := net.ParseIP(info.LHost)
	rhost := net.ParseIP(info.RHost)

	conn := &tcpConn{
		stream,
		&net.TCPAddr{IP: lhost, Port: int(info.LPort)},
		&net.TCPAddr{IP: rhost, Port: int(info.RPort)},
	}

	return conn
}

// RemoteAddr - Implements net.Conn, returns specialized addr type
func (c *tcpConn) RemoteAddr() net.Addr {
	return c.rAddr
}

// LocalAddr - Implements net.Conn, returns specialized addr type
func (c *tcpConn) LocalAddr() net.Addr {
	return c.lAddr
}

// SetDeadline- Implements net.Conn
func (c *tcpConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline - Implements net.Conn
func (c *tcpConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline - Implements net.Conn
func (c *tcpConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// Close - Implements net.Conn
func (c *tcpConn) Close() error {
	if c.ReadWriteCloser != nil {
		return c.ReadWriteCloser.Close()
	}
	return nil
}

// ------------------------------------------------------------------------------------------------------------
// Outbound Connections:
// Connections (generally satisfying net.Conn) routed TO the implant, need to pass some information to Comms
// ------------------------------------------------------------------------------------------------------------

func newConnOutboundTCP(uri *url.URL) *commpb.Conn {

	conn := &commpb.Conn{
		RHost:     uri.Hostname(),
		Transport: commpb.Transport_TCP,
	}

	// Port
	rport, _ := strconv.Atoi(uri.Port())
	if rport != 0 {
		conn.RPort = int32(rport)
	}

	// Transport / Application protocols. Used in case handlers at the implant must verify details
	// and fields, or for clearer logging when debugging heavy traffic. Also, sometimes we may
	// directly dial TCP from the Comm system, but handled by the implant slightly differently.
	switch uri.Scheme {
	case "mtls":
		conn.Application = commpb.Application_MTLS
	case "http":
		conn.Application = commpb.Application_HTTP
	case "https":
		conn.Application = commpb.Application_HTTPS
	case "socks", "socks5":
		conn.Application = commpb.Application_Socks5
	case "ftp":
		conn.Application = commpb.Application_FTP
	case "smtp":
		conn.Application = commpb.Application_SMTP
	case "named_pipe", "named_pipes", "namedpipe", "pipe":
		conn.Application = commpb.Application_NamedPipe
	default:
		conn.Application = commpb.Application_None
	}

	return conn
}
