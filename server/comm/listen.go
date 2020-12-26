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
	"net"
)

// Listen - Returns a listener started on a valid network address anywhere in either the server interfaces,
// or any implant's interface if the latter is served by an active route. Valid networks are "tcp".
func Listen(network, host string) (ln net.Listener, err error) {
	return ListenTCP(network, host)
}

// ListenPacket - Returns a packet connection (message oriented: UDP or IP) that is fed by an actual packet
// connection/listener in either the server interfaces, or any implant's interface. Valid networks are "udp".
func ListenPacket(network, host string) (conn net.PacketConn, err error) {
	return ListenUDP(network, host)
}
