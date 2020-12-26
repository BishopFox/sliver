package comm

import (
	"net"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

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

// listener - A listener started on the implant, with more thorough handler information.
// This listener is registered so that we can stop it from the server, like jobs.
// It can be both a UDP (message-oriented) or a TCP (stream-oriented) listener.
type listener struct {
	// Core
	info  *sliverpb.Handler
	close chan bool

	// Protocol-specific
	stream net.Listener   // Stream-oriented listeners (TCP, Named pipes)
	packet net.PacketConn // Message-oriented listeners (UDP, Unixgram)
}

// newStreamListener - Stream-oriented listener
func newStreamListener(info *sliverpb.Handler, base net.Listener) (ln *listener) {
	ln = &listener{
		info:   info,
		close:  make(chan bool),
		stream: base,
	}
	return
}

// newPacketListener - Message-oriented listener
func newPacketListener(info *sliverpb.Handler, packetConn net.PacketConn) (ln *listener) {
	ln = &listener{
		info:   info,
		close:  make(chan bool),
		packet: packetConn,
	}
	return
}

func (ln *listener) Close() (err error) {
	if ln.stream != nil {
		return ln.stream.Close()
	}

	// TODO: UDP check if we need to close Comm stream.
	if ln.packet != nil {
		return ln.packet.Close()
	}
	return
}
