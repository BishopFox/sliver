package transport

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
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	randomIDSize = 16 // 64bits
)

var (
	// Tunnels - Holds refs to all tunnels
	Tunnels tunnels
)

type tunnelAddr struct {
	network string
	addr    string
}

func (a *tunnelAddr) Network() string {
	return a.network
}

func (a *tunnelAddr) String() string {
	return fmt.Sprintf("%s://%s", a.network, a.addr)
}

// Holds the tunnels locally so we can map incoming data
// messages to the tunnel
type tunnels struct {
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex
	stream  rpcpb.SliverRPC_TunnelDataClient
}

// Get - Get a tunnel
func (t *tunnels) Get(tunnelID uint64) *Tunnel {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return (*t.tunnels)[tunnelID]
}

// Start - Add a tunnel to the core mapper
func (t *tunnels) Start(tunnelID uint64, sessionID uint32) *Tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	tunnel := &Tunnel{
		ID:        tunnelID,
		SessionID: sessionID,
		Send:      make(chan []byte),
		Recv:      make(chan []byte),
	}
	(*t.tunnels)[tunnelID] = tunnel
	go func() {
		tunnel.IsOpen = true
		for data := range tunnel.Send {
			log.Printf("Send %d bytes on tunnel %d", len(data), tunnel.ID)
			t.stream.Send(&sliverpb.TunnelData{
				TunnelID:  tunnel.ID,
				SessionID: tunnel.SessionID,
				Data:      data,
			})
		}
	}()
	tunnel.Send <- make([]byte, 0) // Send "zero" message to bind client to tunnel
	return tunnel
}

// Close - Close the tunnel channels
func (t *tunnels) Close(tunnelID uint64) {
	log.Printf("Closing tunnel %d", tunnelID)
	t.mutex.Lock()
	defer t.mutex.Unlock()
	tunnel := (*t.tunnels)[tunnelID]
	if tunnel != nil {
		delete((*t.tunnels), tunnelID)
		tunnel.IsOpen = false
		close(tunnel.Recv)
		close(tunnel.Send)
	}
}

// Tunnel - Duplex data tunnel
type Tunnel struct {
	ID        uint64
	IsOpen    bool
	SessionID uint32

	Send chan []byte
	Recv chan []byte
}

// Write - Writer method for interface
func (tun *Tunnel) Write(data []byte) (int, error) {
	log.Printf("Write %d bytes", len(data))
	if !tun.IsOpen {
		return 0, io.EOF
	}
	tun.Send <- data
	n := len(data)
	return n, nil
}

// Read - Reader method for interface
func (tun *Tunnel) Read(data []byte) (int, error) {
	var buff bytes.Buffer
	if !tun.IsOpen {
		log.Printf("Warning: Read on closed tunnel %d", tun.ID)
		return 0, io.EOF
	}
	select {
	case data := <-tun.Recv:
		log.Printf("Read %d bytes", len(data))
		buff.Write(data)
	default:
		break
	}
	n := copy(data, buff.Bytes())
	return n, nil
}

// TunnelLoop - Parses incoming tunnel messages and distributes them
//              to session/tunnel objects
func TunnelLoop() error {
	log.Println("Starting tunnel data loop ...")
	defer log.Printf("Warning: TunnelLoop exited")
	stream, err := RPC.TunnelData(context.Background())
	if err != nil {
		return err
	}
	Tunnels = tunnels{
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		stream:  stream,
	}
	for {

		log.Printf("Waiting for TunnelData ...")
		incoming, err := stream.Recv()
		log.Printf("Recv stream msg: %v", incoming)
		if err == io.EOF {
			log.Printf("EOF Error: Tunnel data stream closed")
			return nil
		}
		if err != nil {
			log.Printf("Tunnel data read error: %s", err)
			return err
		}
		log.Printf("Received TunnelData for tunnel %d", incoming.TunnelID)
		tunnel := Tunnels.Get(incoming.TunnelID)
		if tunnel != nil {
			if !incoming.Closed {
				log.Printf("Received data on tunnel %d", tunnel.ID)
				tunnel.Recv <- incoming.GetData()
			} else {
				log.Printf("Closing tunnel %d", tunnel.ID)
				tunnel.IsOpen = false
				close(tunnel.Recv)
			}
		} else {
			log.Printf("Received tunnel data for non-existent tunnel id %d", incoming.TunnelID)
		}
	}
}
