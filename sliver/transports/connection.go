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
	"io"
	"sync"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Connection - Abstract connection to the server
type Connection struct {
	Send    chan *pb.Envelope
	Recv    chan *pb.Envelope
	IsOpen  bool
	ctrl    chan bool
	cleanup func()
	once    *sync.Once
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex
}

// Cleanup - Execute cleanup once
func (c *Connection) Cleanup() {
	c.once.Do(func() {
		c.cleanup()
		c.IsOpen = false
	})
}

// Tunnel - Duplex byte read/write
type Tunnel struct {
	ID uint64

	Reader       io.ReadCloser
	ReadSequence uint64

	Writer        io.WriteCloser
	WriteSequence uint64
}

// Tunnel - Add tunnel to mapping
func (c *Connection) Tunnel(ID uint64) *Tunnel {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return (*c.tunnels)[ID]
}

// AddTunnel - Add tunnel to mapping
func (c *Connection) AddTunnel(tun *Tunnel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	(*c.tunnels)[tun.ID] = tun
}

// RemoveTunnel - Add tunnel to mapping
func (c *Connection) RemoveTunnel(ID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(*c.tunnels, ID)
}
