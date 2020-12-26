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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// conn - An abstracted connection that implements the net.Conn interface.
// The remote and local addresses are set from the payload info attached in the SSH Channel.
type conn struct {
	io.ReadWriteCloser
	rAddr    net.Addr
	lAddr    net.Addr
	errClose chan error // The connection has been closed by one of its users.
}

// newConn can produce more specialized types of connections. Eventually the type returned
// should not be the connection but an interface net.Conn, net.TCPConn, or whatever.
func newConn(info *sliverpb.ConnectionInfo, stream io.ReadWriteCloser) *conn {
	conn := &conn{
		stream,
		nil,
		nil,
		make(chan error, 1)}

	lhost := net.ParseIP(info.LHost)
	rhost := net.ParseIP(info.RHost)

	conn.rAddr = &net.TCPAddr{IP: rhost, Port: int(info.RPort)}
	conn.lAddr = &net.TCPAddr{IP: lhost, Port: int(info.LPort)}

	// Maybe add eventual more precise branching with info.Application protocol.

	return conn
}

// newConnInfo - Populate a ConnectionInfo message to be used by the routing system along with a connection stream.
func newConnInfo(uri *url.URL, route *Route) *sliverpb.ConnectionInfo {

	info := &sliverpb.ConnectionInfo{
		ID: route.ID.String(),
	}
	// Get RHost/RPort
	info.RHost = uri.Hostname()
	rport, _ := strconv.Atoi(uri.Port())
	if rport != 0 {
		info.RPort = int32(rport)
	}
	// Transport / Application protocols
	switch uri.Scheme {
	case "mtls", "http", "https", "socks", "socks5", "ftp", "smtp":
		// Transport
		info.Transport = sliverpb.TransportProtocol_TCP
		// Application
		switch uri.Scheme {
		case "mtls":
			info.Application = sliverpb.ApplicationProtocol_MTLS
		case "http":
			info.Application = sliverpb.ApplicationProtocol_HTTP
		case "https":
			info.Application = sliverpb.ApplicationProtocol_HTTPS
		case "socks", "socks5":
			info.Application = sliverpb.ApplicationProtocol_Socks5
		case "ftp":
			info.Application = sliverpb.ApplicationProtocol_FTP
		case "smtp":
			info.Application = sliverpb.ApplicationProtocol_SMTP
		case "named_pipe", "named_pipes", "namedpipe", "pipe":
			info.Application = sliverpb.ApplicationProtocol_NamedPipe
		}
	case "dns", "udp", "quic":
		// Transport
		info.Transport = sliverpb.TransportProtocol_UDP
		// Application
		switch uri.Scheme {
		case "dns":
			info.Application = sliverpb.ApplicationProtocol_DNS
		case "quic":
			info.Application = sliverpb.ApplicationProtocol_QUIC
		}
	}

	return info
}

// RemoteAddr - Implements net.Conn, returns specialized addr type
func (c *conn) RemoteAddr() net.Addr {
	return c.rAddr
}

// LocalAddr - Implements net.Conn, returns specialized addr type
func (c *conn) LocalAddr() net.Addr {
	return c.lAddr
}

// SetDeadline- Implements net.Conn
func (c *conn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline - Implements net.Conn
func (c *conn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline - Implements net.Conn
func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}

// Close - Implements net.Conn
// func (c *conn) Close() error {
//         return c.ReadWriteCloser.Close()
// }
