package core

import (
	"crypto/rand"
	"encoding/binary"
	sliverpb "sliver/protobuf/sliver"
	"sync"

	"github.com/golang/protobuf/proto"
)

var (
	// Tunnels - Interating with duplex tunnels
	Tunnels = tunnels{
		tunnels: &map[uint64]*tunnel{},
		mutex:   &sync.RWMutex{},
	}
)

type tunnels struct {
	tunnels *map[uint64]*tunnel
	mutex   *sync.RWMutex
}

func (t *tunnels) CreateTunnel(client *Client, sliverID uint32) *tunnel {
	tunID := newTunnelID()
	sliver := Hive.Sliver(sliverID)
	tun := &tunnel{
		ID:     tunID,
		Client: client,
		Sliver: sliver,
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()
	(*t.tunnels)[tun.ID] = tun

	return tun
}

func (t *tunnels) CloseTunnel(tunnelID uint64, reason string) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	tunnel := (*t.tunnels)[tunnelID]
	if tunnel != nil {
		tunnelClose, _ := proto.Marshal(&sliverpb.TunnelClose{
			TunnelID: tunnelID,
			Err:      reason,
		})
		tunnel.Client.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
		tunnel.Sliver.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
		delete(*t.tunnels, tunnelID)
		return true
	}
	return false
}

func (t *tunnels) Tunnel(tunnelID uint64) *tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return (*t.tunnels)[tunnelID]
}

// A tunnel is essentially just a mapping between a specific client and sliver
// with an identifier, these tunnels are full duplex. The server doesn't really
// care what data gets passed back and forth it just facilitates the connection
type tunnel struct {
	ID     uint64
	Sliver *Sliver
	Client *Client
}

// tunnelID - New 32bit identifier
func newTunnelID() uint64 {
	randBuf := make([]byte, 8)
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
