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
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
)

// Forwarder - a port forwarder either bind/direct or reverse/remote, and TCP or UDP
// This interface is used for easier management by client commands and printers.
type Forwarder interface {
	Info() *commpb.Handler        // Information about this forwarder
	SessionID() uint32            // The session on which this forwarder works
	ConnStats() string            // TCP returns the number of active conns, UDP gives sent/recv values
	LocalAddr() string            // The address on which direct forwarders listen
	Close(bool) error             // The user can close the port forwarder, optionally with connections.
	serve()                       // The forwarder is fully working (not called by client commands)
	handleReverse(ssh.NewChannel) // TCP forwarders push streams to a queue, UDP use them for passing data
}

var (
	// Forwarders - All port forwarders, (bind/reverse, TCP/UDP) started
	// between this client console and one or more implant sessions
	Forwarders = &forwarders{
		active: map[string]Forwarder{},
		mutex:  &sync.RWMutex{},
	}
)

type forwarders struct {
	active map[string]Forwarder
	mutex  *sync.RWMutex
}

// Get - Get a port forwarder by ID
func (c *forwarders) Get(id string) Forwarder {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.active[id]
}

// Add - Add a port forwarder to the map
func (c *forwarders) Add(l Forwarder) Forwarder {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.active[l.Info().ID] = l
	return l
}

// Remove - Remove a port forwarder from the map.
func (c *forwarders) Remove(id string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	l := c.active[id]
	if l != nil {
		delete(c.active, id)
	}
}

// All - Get all port forwarders
func (c *forwarders) All() map[string]Forwarder {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.active
}

// -------------------------------------------- Direct Port Forwarders -------------------------------------------------

// PortfwdDirectTCP - Start a direct (console -> implant) TCP port forwarder  on this client.
func PortfwdDirectTCP(sessID uint32, info *commpb.Handler, lhost string, lport int) (err error) {

	// Creates a TCP forwader listening on the console host, not yet wired to comms
	forwarder, err := newDirectForwarderTCP(info, lhost, lport)
	if err != nil {
		return err
	}
	forwarder.sessionID = sessID

	// The request sent to server includes an optional remote lhost and remote lport which,
	// if specified (non "" or 0), will be used a source address for the TCP connection.
	req := &commpb.PortfwdOpenReq{
		Handler: info,
		Request: &commonpb.Request{SessionID: sessID},
	}
	data, _ := proto.Marshal(req)

	// We now request the server Comms to map this direct portfwd.
	_, resp, err := ClientComm.ssh.SendRequest(commpb.Request_PortfwdStart.String(), true, data)
	if err != nil {
		Forwarders.Remove(forwarder.info.ID)
		return fmt.Errorf("Comm error: %s", err.Error())
	}
	res := &commpb.PortfwdOpen{}
	proto.Unmarshal(resp, res)

	// The server might give us an error
	if !res.Success {
		Forwarders.Remove(forwarder.info.ID)
		return fmt.Errorf("Portfwd error: %s", res.Response.Err)
	}

	// Else listen for incoming connections
	go forwarder.serve()

	return
}

// PortfwdDirectUDP - Start a direct (console -> implant) UDP port forwarder, passing the following arguments:
// sessID - The Id of the session we currently interacting with, and on which we set the portfwd
// info - The remote host:port, with an optional lhost:lport for specifying a source address when dialing.
// lhost/lport - host:port on which to listen on this console.
func PortfwdDirectUDP(sessID uint32, info *commpb.Handler, lhost string, lport int) error {

	// Creates a UDP forwader listening on the console host, not yet wired to comms
	forwarder, err := newDirectForwarderUDP(info, lhost, lport)
	if err != nil {
		return err
	}
	forwarder.sessionID = sessID

	// The request sent to server includes an optional remote lhost and remote lport which,
	// if specified (non "" or 0), will be used a source address for the UDP connection.
	req := &commpb.PortfwdOpenReq{
		Handler: info,
		Request: &commonpb.Request{SessionID: sessID},
	}
	data, _ := proto.Marshal(req)

	// We now request the server Comms to start everything on the implant, and to yield us a stream.
	_, resp, err := ClientComm.ssh.SendRequest(commpb.Request_PortfwdStart.String(), true, data)
	if err != nil {
		Forwarders.Remove(forwarder.info.ID)
		return fmt.Errorf("Comm error: %s", err.Error())
	}
	res := &commpb.PortfwdOpen{}
	proto.Unmarshal(resp, res)

	// The server might give us an error
	if !res.Success {
		Forwarders.Remove(forwarder.info.ID)
		return fmt.Errorf("Portfwd error: %s", res.Response.Err)
	}

	// Serve read/write
	go forwarder.serve()

	return nil
}

// -------------------------------------------- Reverse Port Forwarders -------------------------------------------------

// PortfwdReverseTCP - Start a reverse port forward TCP handler on this client
func PortfwdReverseTCP(sessID uint32, info *commpb.Handler) (err error) {

	forwarder, err := newReverseForwarderTCP(info)
	if err != nil {
		return err
	}
	forwarder.sessionID = sessID

	// The request sent to server
	req := &commpb.PortfwdOpenReq{
		Handler: info,
		Request: &commonpb.Request{SessionID: sessID},
	}
	data, _ := proto.Marshal(req)

	// Request to server -> server requests TCP handler to implant
	// We now request the server Comms to start everything on the implant, and to yield us a stream.
	_, resp, err := ClientComm.ssh.SendRequest(commpb.Request_PortfwdStart.String(), true, data)
	if err != nil {
		Forwarders.Remove(forwarder.info.ID)
		return fmt.Errorf("Comm error: %s", err.Error())
	}
	res := &commpb.PortfwdOpen{}
	proto.Unmarshal(resp, res)

	// The server might give us an error
	if !res.Success {
		Forwarders.Remove(forwarder.info.ID)
		return fmt.Errorf("Portfwd error: %s", res.Response.Err)
	}

	// Process connections coming from the server.
	go forwarder.serve()

	return
}

// PortfwdReverseUDP - Start a reverse port forward UDP handler on this client
func PortfwdReverseUDP(sessID uint32, info *commpb.Handler) (err error) {

	forwarder, err := newReverseForwarderUDP(info)
	if err != nil {
		return err
	}
	forwarder.sessionID = sessID

	// The request sent to server
	req := &commpb.PortfwdOpenReq{
		Handler: info,
		Request: &commonpb.Request{SessionID: sessID},
	}
	data, _ := proto.Marshal(req)

	// Start waiting for the Comm system to return us the stream  that
	// will be created by the handler upon arrival of the request below
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {

		// Wait for the channel.
		stream := <-forwarder.pending

		// Wrap the connection's stream into a UDP stream, with encoding
		forwarder.inbound = &udpStream{
			r: gob.NewDecoder(stream),
			w: gob.NewEncoder(stream),
			c: stream,
		}

		// The listener streams are ready.
		wg.Done()
	}()

	// We now request the server Comms to start everything on the implant, and to yield us a stream.
	_, resp, err := ClientComm.ssh.SendRequest(commpb.Request_PortfwdStart.String(), true, data)
	if err != nil {
		Forwarders.Remove(forwarder.info.ID)
		return fmt.Errorf("Comm error: %s", err.Error())
	}
	res := &commpb.PortfwdOpen{}
	proto.Unmarshal(resp, res)

	// The server might give us an error
	if !res.Success {
		wg.Done() // just in case
		Forwarders.Remove(forwarder.info.ID)
		return fmt.Errorf("Portfwd error: %s", res.Response.Err)
	}

	// Wait until the server Comm has sent us back the stream.
	wg.Wait()

	go forwarder.serve()

	return
}
