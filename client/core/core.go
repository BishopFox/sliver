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
			t.stream.Send(&sliverpb.TunnelData{
				TunnelID:  tunnel.ID,
				SessionID: tunnel.SessionID,
				Data:      data,
			})
		}
	}()
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

// Write -
func (tun *Tunnel) Write(data []byte) (int, error) {
	log.Printf("Sending %d bytes on session %d", len(data), tun.SessionID)
	if !tun.IsOpen {
		return 0, io.EOF
	}
	tun.Send <- data
	n := len(data)
	return n, nil
}

// Read -
func (tun *Tunnel) Read(data []byte) (int, error) {
	var buff bytes.Buffer
	if !tun.IsOpen {
		return 0, io.EOF
	}
	select {
	case msg := <-tun.Recv:
		buff.Write(msg)
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
			return nil
		}
		tunnel := Tunnels.Get(data.TunnelID)
		if tunnel != nil {
			log.Printf("Received data on tunnel %d", tunnel.ID)
			tunnel.Recv <- data.GetData()
		} else {
			log.Printf("Received tunnel data for non-existent tunnel id %d", data.TunnelID)
		}
	}
}
