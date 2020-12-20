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
	"crypto/rand"
	"encoding/binary"
	"sync"
)

// Comms - (SSH-multiplexing) ------------------------------------------------------------

var (
	// Comms - All multiplexers currently running in Sliver, providing connection routing.
	Comms = &comms{
		active: map[uint32]*Comm{},
		mutex:  &sync.RWMutex{},
	}
)

type comms struct {
	active map[uint32]*Comm
	mutex  *sync.RWMutex
}

// Get - Get a session by ID
func (c *comms) Get(commID uint32) *Comm {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.active[commID]
}

// Add - Add a sliver to the hive (atomically)
func (c *comms) Add(mux *Comm) *Comm {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.active[mux.ID] = mux
	return mux
}

// Remove - Remove a sliver from the hive (atomically)
func (c *comms) Remove(commID uint32) {
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
	Tunnels = &muxTunnels{
		tunnels: map[uint64]*tunnel{},
		mutex:   &sync.RWMutex{},
	}
)

type muxTunnels struct {
	tunnels map[uint64]*tunnel
	mutex   *sync.RWMutex
}

// Tunnel - Add tunnel to mapping
func (c *muxTunnels) Tunnel(ID uint64) *tunnel {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tunnels[ID]
}

// AddTunnel - Add tunnel to mapping
func (c *muxTunnels) AddTunnel(tun *tunnel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tunnels[tun.ID] = tun
}

// RemoveTunnel - Add tunnel to mapping
func (c *muxTunnels) RemoveTunnel(ID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.tunnels, ID)
}

// newTunnelID - New 64-bit identifier
func newTunnelID() uint64 {
	randBuf := make([]byte, 8)
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
