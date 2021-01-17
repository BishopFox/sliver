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

	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

const (
	defaultNetTimeout = time.Second * 60
)

// Comms - (SSH-multiplexing) ------------------------------------------------------------

var (
	// Comms - All multiplexers currently running in Sliver, providing connection routing.
	Comms = &comms{
		active: map[string]*Comm{},
		mutex:  &sync.RWMutex{},
	}
	commID = uint32(0)
)

type comms struct {
	active map[string]*Comm
	server *Comm
	mutex  *sync.RWMutex
}

// Get - Get a Session Comm by ID
func (c *comms) Get(commID string) *Comm {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.active[commID]
}

// Add - Add a Comm to the map
func (c *comms) Add(mux *Comm) *Comm {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.active[mux.RemoteAddress] = mux
	if c.server == nil {
		c.server = mux
	}
	return mux
}

// Remove - Remove a Comm from the map
func (c *comms) Remove(commID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	mux := c.active[commID]
	if mux != nil {
		delete(c.active, commID)
	}
}

// Tunnels - ReadWriteClosers over Sliver Session ----------------------------------------

var (
	// Tunnels - Stores all Tunnels used by the Comm system to route traffic.
	Tunnels = &tunnels{
		tunnels: map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
	}
)

type tunnels struct {
	tunnels map[uint64]*Tunnel
	mutex   *sync.RWMutex
}

// Tunnel - Add tunnel to mapping
func (c *tunnels) Tunnel(ID uint64) *Tunnel {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tunnels[ID]
}

// AddTunnel - Add tunnel to mapping
func (c *tunnels) AddTunnel(tun *Tunnel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tunnels[tun.ID] = tun
}

// RemoveTunnel - Add tunnel to mapping
func (c *tunnels) RemoveTunnel(ID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.tunnels, ID)
}

// Listeners ----------------------------------------------------------------------------

var (
	// Listeners - All instantiated active connection listeners.
	// They route their connections back to the C2 server.
	Listeners = &commListeners{
		active: map[string]listener{},
		mutex:  &sync.RWMutex{},
	}
)

// listener - A listener started on the implant, with more thorough handler information.
// This listener is registered so that we can stop it from the server, like jobs.
// It can be a UDP (message-oriented) or a TCP (stream-oriented) listener.
type listener interface {
	Info() *commpb.Handler
	close() error
}

type commListeners struct {
	active map[string]listener
	mutex  *sync.RWMutex
}

// Get - Get a listener by ID
func (c *commListeners) Get(id string) listener {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.active[id]
}

// Add - Add a listener to the map.
func (c *commListeners) Add(l listener) listener {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.active[l.Info().ID] = l
	return l
}

// Remove - Remove a listener from the map (listener closed)
func (c *commListeners) Remove(id string) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	l := c.active[id]
	if l != nil {
		// Depending on the transport protocol (TCP or UDP),
		// closing the listener has different effects:
		// TCP: kills listener goroutines but NOT connections
		// UDP: closes connections.
		err := l.close()
		if err == nil {
			// {{if .Config.Debug}}
			log.Printf("Stopped %s/%s listener (%s) ...",
				l.Info().Transport, l.Info().Application, l.Info().ID)
			// {{end}}
		}
		delete(c.active, id)
		return err
	}
	return fmt.Errorf("no handler for ID %s", id)
}

// Piping & Utils ------------------------------------------------------------

// transportConn - Original function taked from the gost project, with comments added
// and without an error group to wait for both sides to declare an error/EOF.
// This is used to transport stream-oriented traffic like TCP, because EOF matters here.
func transportConn(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)

	// Source reads from
	go func() {
		errc <- copyBuffer(rw1, rw2)
	}()

	// Source writes to
	go func() {
		errc <- copyBuffer(rw2, rw1)
	}()

	// Any error arising from either the source
	// or the destination connections and we return.
	// Connections are not automatically closed
	// so a function is called after.
	err := <-errc
	if err != nil && err == io.EOF {
		err = nil
	}
	return err
}

func copyBuffer(dst io.Writer, src io.Reader) error {
	buf := lPool.Get().([]byte)
	defer lPool.Put(buf)

	_, err := io.CopyBuffer(dst, src, buf)
	return err
}

var (
	sPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, smallBufferSize)
		},
	}
	mPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, mediumBufferSize)
		},
	}
	lPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, largeBufferSize)
		},
	}
)

var (
	tinyBufferSize   = 512
	smallBufferSize  = 2 * 1024  // 2KB small buffer
	mediumBufferSize = 8 * 1024  // 8KB medium buffer
	largeBufferSize  = 32 * 1024 // 32KB large buffer
)

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// closeConnections - When a src (SSH channel) is done piping to/from a net.Conn, we close both.
func closeConnections(src io.Closer, dst io.Closer) {

	// We always leave some time before closing the connections,
	// because some of the traffic might still be processed by
	// the SSH RPC tunnel, which can be a bit slow to process data.
	time.Sleep(200 * time.Millisecond)

	// Close connections
	if dst != nil {
		dst.Close()
	}
	if src != nil {
		src.Close()
	}
}
