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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// handleTCP - A connection coming from the server is destined to one of the implant's networks.
// Forge local and/or remote TCP addresses and pass them to the dialer. Pipe the connection.
func handleTCP(info *sliverpb.ConnectionInfo, src io.ReadWriteCloser) error {
	var srcAddr *net.TCPAddr
	if info.LHost != "" || info.LPort == 0 {
		srcAddr = &net.TCPAddr{
			IP:   net.ParseIP(info.LHost),
			Port: int(info.LPort),
		}
	}

	dstAddr := &net.TCPAddr{
		IP:   net.ParseIP(info.RHost),
		Port: int(info.RPort),
	}

	dst, err := net.DialTCP("tcp", srcAddr, dstAddr)
	if err != nil {
		return err
	}
	transport(src, dst)
	return nil
}
