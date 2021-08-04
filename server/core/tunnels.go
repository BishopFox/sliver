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
	"errors"
	"sync"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	// Tunnels - Interating with duplex tunnels
	Tunnels = tunnels{
		tunnels: map[uint64]*Tunnel{},
		mutex:   &sync.Mutex{},
	}

	// ErrInvalidTunnelID - Invalid tunnel ID value
	ErrInvalidTunnelID = errors.New("Invalid tunnel ID")
)

// Tunnel  - Essentially just a mapping between a specific client and sliver
// with an identifier, these tunnels are full duplex. The server doesn't really
// care what data gets passed back and forth it just facilitates the connection
type Tunnel struct {
	ID        uint64
	SessionID uint32

	ToImplant         chan []byte
	ToImplantSequence uint64

	FromImplant         chan *sliverpb.TunnelData
	FromImplantSequence uint64

	Client rpcpb.SliverRPC_TunnelDataServer
}

type tunnels struct {
	tunnels map[uint64]*Tunnel
	mutex   *sync.Mutex
}

func (t *tunnels) Create(sessionID uint32) *Tunnel {
	tunnelID := NewTunnelID()
	session := Sessions.Get(sessionID)
	tunnel := &Tunnel{
		ID:          tunnelID,
		SessionID:   session.ID,
		ToImplant:   make(chan []byte),
		FromImplant: make(chan *sliverpb.TunnelData),
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tunnels[tunnel.ID] = tunnel

	return tunnel
}

func (t *tunnels) Close(tunnelID uint64) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	tunnel := t.tunnels[tunnelID]
	if tunnel == nil {
		return ErrInvalidTunnelID
	}
	tunnelClose, err := proto.Marshal(&sliverpb.TunnelData{
		TunnelID:  tunnel.ID,
		SessionID: tunnel.SessionID,
		Closed:    true,
	})
	if err != nil {
		return err
	}
	data, err := proto.Marshal(&sliverpb.Envelope{
		Type: sliverpb.MsgTunnelClose,
		Data: tunnelClose,
	})
	if err != nil {
		return err
	}
	tunnel.ToImplant <- data // Send an in-band close to implant
	delete(t.tunnels, tunnelID)
	close(tunnel.ToImplant)
	close(tunnel.FromImplant)
	return nil
}

// Get - Get a tunnel
func (t *tunnels) Get(tunnelID uint64) *Tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.tunnels[tunnelID]
}

// NewTunnelID - New 64-bit identifier
func NewTunnelID() uint64 {
	randBuf := make([]byte, 8)
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
