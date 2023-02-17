package core

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"log"
	"sync"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// tunnelsStorage - Holds refs to all tunnels
	tunnelsStorage *tunnels

	tunnelsSingletonLock = &sync.Mutex{}
)

// GetTunnels - singleton function that returns or initializes all tunnels
func GetTunnels() *tunnels {
	tunnelsSingletonLock.Lock()
	defer tunnelsSingletonLock.Unlock()

	if tunnelsStorage == nil {
		log.Println("Creating single instance of tunnels.")

		tunnelsStorage = &tunnels{
			tunnels:     &map[uint64]*TunnelIO{},
			mutex:       &sync.RWMutex{},
			streamMutex: &sync.Mutex{},
		}
	}

	return tunnelsStorage
}

// Holds the tunnels locally so we can map incoming data
// messages to the tunnel
type tunnels struct {
	tunnels     *map[uint64]*TunnelIO
	mutex       *sync.RWMutex
	streamMutex *sync.Mutex
	stream      rpcpb.SliverRPC_TunnelDataClient
}

func (t *tunnels) SetStream(stream rpcpb.SliverRPC_TunnelDataClient) {
	t.streamMutex.Lock()
	defer t.streamMutex.Unlock()

	log.Printf("Set stream")

	t.stream = stream
}

// Get - Get a tunnel
func (t *tunnels) Get(tunnelID uint64) *TunnelIO {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	log.Printf("Get tunnel %d", tunnelID)

	return (*t.tunnels)[tunnelID]
}

// send - safe way to send a message to the stream
// protobuf stream allow only one writer at a time, so just in case there is a mutex for it
func (t *tunnels) send(tunnelData *sliverpb.TunnelData) error {
	t.streamMutex.Lock()
	defer t.streamMutex.Unlock()

	if t.stream == nil {
		return errors.New("uninitizlied stream")
	}

	log.Printf("Private send to stream, tunnelId: %d", tunnelData.TunnelID)

	return t.stream.Send(tunnelData)
}

// Start - Add a tunnel to the core mapper
func (t *tunnels) Start(tunnelID uint64, sessionID string) *TunnelIO {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	tunnel := NewTunnelIO(tunnelID, sessionID)

	(*t.tunnels)[tunnelID] = tunnel

	go func(tunnel *TunnelIO) {
		tunnel.Open()
		log.Printf("Tunnel now is open, %d", tunnelID)

		for data := range tunnel.Send {
			log.Printf("Send %d bytes on tunnel %d", len(data), tunnel.ID)

			err := t.send(&sliverpb.TunnelData{
				TunnelID:  tunnel.ID,
				SessionID: tunnel.SessionID,
				Data:      data,
			})

			if err != nil {
				log.Printf("Error sending, %s", err)
			}
		}

		log.Printf("Tunnel Send channel looks closed now. %d", tunnelID)
	}(tunnel)

	tunnel.Send <- make([]byte, 0) // Send "zero" message to bind client to tunnel
	return tunnel
}

// Close - Close the tunnel channels
func (t *tunnels) Close(tunnelID uint64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	log.Printf("Closing tunnel %d", tunnelID)

	tunnel := (*t.tunnels)[tunnelID]

	if tunnel != nil {
		tunnel.Close()

		delete((*t.tunnels), tunnelID)
	}
}

// CloseForSession - closing all tunnels for specified session id
func (t *tunnels) CloseForSession(sessionID string) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	log.Printf("Closing all tunnels for session %s", sessionID)

	for tunnelID, tunnel := range *t.tunnels {
		if tunnel.SessionID == sessionID {
			// Weird way to avoid deadlocks but let it be
			go func(tunnelID uint64) {
				GetTunnels().Close(tunnelID)
			}(tunnelID)
		}
	}
}
