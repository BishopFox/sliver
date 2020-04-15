package core

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
	// Tunnels - Interating with duplex tunnels
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
func TunnelLoop(rpc rpcpb.SliverRPCClient) error {
	log.Println("Starting tunnel data loop ...")
	stream, err := rpc.TunnelData(context.Background())
	if err != nil {
		return err
	}
	Tunnels = tunnels{
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		stream:  stream,
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			log.Printf("EOF Error: Tunnel data stream closed")
			return nil
		}
		if err != nil {
			log.Printf("Tunnel data read error: %s", err)
			return err
		}
		tunnel := Tunnels.Get(data.TunnelID)
		if tunnel != nil {
			if !data.Closed {
				log.Printf("Received data on tunnel %d", tunnel.ID)
				tunnel.Recv <- data.GetData()
			} else {
				tunnel.IsOpen = false
				tunnel.Recv <- make([]byte, 0)
			}
		} else {
			log.Printf("Received tunnel data for non-existent tunnel id %d", data.TunnelID)
		}
	}
}
