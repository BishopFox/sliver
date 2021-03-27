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
	"errors"
	"sync"

	"github.com/bishopfox/sliver/protobuf/commpb"
	"golang.org/x/crypto/ssh"
)

var (
	// portForwarders - All active port Forwarders on all client consoles.
	portForwarders = &forwarders{
		active: map[string]forwarder{},
		mutex:  &sync.RWMutex{},
	}
)

// forwarder - A simple mapping between a Client Comm and a handler ID, so that the Comm
// system knows how to route forward/reverse any traffic coming to/from a Client console,
// without this traffic being routed (resolved) by the Routes system.
type forwarder interface {
	Info() *commpb.Handler                             // Associated information
	start() error                                      // May do nothing, or request a stream from implant
	handle(info *commpb.Conn, ch ssh.NewChannel) error // Bind or reverse direction.
	close() error                                      // The client has requested to close the forwarder.
	notifyClose() error                                // The implant disconnected, notify client client to close it.
	comms() (client *Comm, server *Comm)               // The forwarder may send requests to them.
}

func newForwarder(client *Comm, info *commpb.Handler, sessionID uint32) (fwd forwarder, err error) {

	// Get the session for this forwarder.
	implantComm := Comms.GetBySession(sessionID)
	if implantComm == nil {
		return nil, errors.New("No implant Comm found (invalid Session ID)")
	}

	// Instantiate the correct port forwarder, depending on direction and protocol.
	switch info.Type {
	case commpb.HandlerType_Bind:
		if info.Transport == commpb.Transport_TCP {
			fwd = newDirectForwarderTCP(info, client, implantComm)
		}
		if info.Transport == commpb.Transport_UDP {
			fwd = newDirectForwarderUDP(info, client, implantComm)
		}
	case commpb.HandlerType_Reverse:
		if info.Transport == commpb.Transport_TCP {
			fwd = newReverseForwarderTCP(info, client, implantComm)
		}
		if info.Transport == commpb.Transport_UDP {
			fwd = newReverseForwarderUDP(info, client, implantComm)
		}
	}

	// Add to map now: we are ready to process any reverse connection request.
	// This registration is common to all forwarders, direct or reverse.
	portForwarders.Add(fwd)

	// Start the port forwarder: this sends a request to the implant, in case it
	// needs to start a listener (for reverse forwarders) or return a dedicated
	// stream (UDP bind/reverse forwarders)
	err = fwd.start()
	if err != nil {
		// None of the start functions will remove
		// the forwarder if an error is thrown.
		portForwarders.Remove(fwd.Info().ID)
		return nil, err
	}

	return
}

// forwarders - All active port forwarders for Client consoles.
type forwarders struct {
	active map[string]forwarder
	mutex  *sync.RWMutex
}

// Get - Get a port fowader by ID
func (c *forwarders) Get(id string) forwarder {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	f, found := c.active[id]
	if !found {
		return nil
	}
	return f
}

// Add - Add a port forwarder to the map.
func (c *forwarders) Add(l forwarder) forwarder {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.active[l.Info().ID] = l
	return l
}

// Remove - Remove a forwarder from the map.
func (c *forwarders) Remove(id string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	l := c.active[id]
	if l != nil {
		delete(c.active, id)
	}
}
