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
	tunnels          *map[uint64]*TunnelIO
	mutex            *sync.RWMutex
	streamMutex      *sync.Mutex
	stream           rpcpb.SliverRPC_TunnelDataClient
	streamGeneration uint64
}

func (t *tunnels) SetStream(stream rpcpb.SliverRPC_TunnelDataClient) uint64 {
	t.streamMutex.Lock()
	defer t.streamMutex.Unlock()

	log.Printf("Set stream")

	t.streamGeneration++
	t.stream = stream
	return t.streamGeneration
}

// CloseStream clears one specific stream generation and closes the tunnels it
// owned. A stale loop cannot clear tunnels created by a replacement stream.
func (t *tunnels) CloseStream(generation uint64) {
	t.streamMutex.Lock()
	defer t.streamMutex.Unlock()
	if generation != t.streamGeneration {
		return
	}

	t.stream = nil
	t.closeAll()
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
		if err := tunnel.Open(); err != nil {
			log.Printf("Failed to open tunnel %d: %v", tunnelID, err)
			return
		}
		log.Printf("Tunnel now is open, %d", tunnelID)

		for {
			select {
			case <-tunnel.Done():
				log.Printf("Tunnel send loop stopped. %d", tunnelID)
				return
			case data := <-tunnel.Send:
				log.Printf("Send %d bytes on tunnel %d", len(data), tunnel.ID)

				err := t.send(&sliverpb.TunnelData{
					TunnelID:  tunnel.ID,
					SessionID: tunnel.SessionID,
					Data:      data,
				})

				if err != nil {
					log.Printf("Error sending, %s", err)
					t.Close(tunnel.ID)
					return
				}
			}
		}
	}(tunnel)

	select {
	case tunnel.Send <- make([]byte, 0): // Send "zero" message to bind client to tunnel
	case <-tunnel.Done():
	}
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

// Reset closes all open tunnels and clears the underlying storage/stream.
// This is used when switching servers inside a single client process.
func (t *tunnels) Reset() {
	t.streamMutex.Lock()
	defer t.streamMutex.Unlock()
	t.streamGeneration++
	t.stream = nil
	t.closeAll()
}

// closeAll runs with streamMutex held so a replacement stream cannot publish
// new tunnels until the previous generation has been fully torn down.
func (t *tunnels) closeAll() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for tunnelID, tunnel := range *t.tunnels {
		if tunnel != nil {
			_ = tunnel.Close()
		}
		delete(*t.tunnels, tunnelID)
	}
}
