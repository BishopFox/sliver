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
	"errors"
	"net"
	"net/url"
	"strconv"
)

// listener - An abstract listener tied to a chain of implant nodes and listen for any incoming connection.
// The remote listener is always tied to a network route, and when it starts listening it inits all nodes
// with appropriate information.
// A listener may be used directly as an abstracted listener by other components, because it implements
// the net.Listener interface. It is thus useful for getting what is treated as a generic connection.
type listener struct {
	id       string // ID passed by the handler
	network  string
	host     string
	pending  chan net.Conn // Processed streams, out as net.Conn
	errClose chan error
	addr     net.Addr
}

// newListener - Creates a new tracker tied to an ID, and with basic network/address information.
func newListener(id, network, host string) *listener {
	rln := &listener{
		id:       id,
		network:  network,
		host:     host,
		pending:  make(chan net.Conn, 100),
		errClose: make(chan error, 1),
	}

	h, _ := url.Parse(host)
	ip := net.ParseIP(h.Hostname())
	port, _ := strconv.Atoi(h.Port())

	switch network {
	case "tcp", "mtls", "http", "https", "h2", "socks", "socks5":
		rln.addr = &net.TCPAddr{IP: ip, Port: port}
	case "udp", "dns", "named_pipe":
		rln.addr = &net.UDPAddr{IP: ip, Port: port}
	case "unix":
		rln.addr = &net.UnixAddr{Net: network, Name: h.Host}
	default:
		rln.addr = &net.TCPAddr{IP: ip, Port: port}
	}
	return rln
}

// Accept - Implements net.Listener Accept(), by providing connections from the routing system.
func (t *listener) Accept() (conn net.Conn, err error) {
	var ok bool
	select {
	case conn = <-t.pending:
	case err, ok = <-t.errClose:
		if !ok {
			if err.Error() == "closed" {
				err = errors.New("accept on closed listener")
				return
			}
			return
		}
	}
	return
}

// Close - Implements net.Listener Close()
func (t *listener) Close() (err error) {
	close(t.pending)            // Does not accept more connections
	for cc := range t.pending { // Close all pending connections
		err = cc.Close()
		if err != nil {
			rLog.Errorf("Tracker (ID: %s) failed to close pending connection: %s", t.id, err.Error())
		}
	}
	t.errClose <- errors.New("closed") // Notify listener is closed.
	listeners.Remove(t.id)             // Remove from trackers map
	return
}

// Addr - Implements net.Listener Addr().
func (t *listener) Addr() net.Addr {
	return t.addr
}

// Listen - Returns a listener accepting connections from wherever in the Comms.
func Listen(network, host string) (ln net.Listener, err error) {
	return
}
