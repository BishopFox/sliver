package core

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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/golang/protobuf/proto"
)

var (
	// Tunnels - Interating with duplex tunnels
	Tunnels = tunnels{
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
	}
)

// Tunnel  - Essentially just a mapping between a specific client and sliver
// with an identifier, these tunnels are full duplex. The server doesn't really
// care what data gets passed back and forth it just facilitates the connection
type Tunnel struct {
	ID      uint64
	Session *Session
	Client  *Client
}

type tunnels struct {
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex
}

func (t *tunnels) CreateTunnel(client *Client, sessionID uint32) *Tunnel {
	tunnelID := NewTunnelID()
	session := Sessions.Get(sessionID)
	tunnel := &Tunnel{
		ID:      tunnelID,
		Client:  client,
		Session: session,
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()
	(*t.tunnels)[tunnel.ID] = tunnel

	return tunnel
}

func (t *tunnels) CloseTunnel(tunnelID uint64, reason string) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	tunnel := (*t.tunnels)[tunnelID]
	if tunnel != nil {
		tunnelClose, _ := proto.Marshal(&sliverpb.TunnelClose{
			TunnelID: tunnelID,
		})
		// tunnel.ClientSend <- &sliverpb.Envelope{
		// 	Type: sliverpb.MsgTunnelClose,
		// 	Data: tunnelClose,
		// }
		tunnel.Session.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
		delete(*t.tunnels, tunnelID)
		return true
	}
	return false
}

// Get - Get a tunnel
func (t *tunnels) Get(tunnelID uint64) *Tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return (*t.tunnels)[tunnelID]
}

// NewTunnelID - New 32bit identifier
func NewTunnelID() uint64 {
	randBuf := make([]byte, 8)
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
