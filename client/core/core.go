package core

import (
	"crypto/rand"
	"encoding/binary"
	"sliver/client/assets"

	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"

	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	randomIDSize = 16 // 64bits
)

var (
	// Events - Connect/Disconnect events
	Events = make(chan *clientpb.Event, 16)

	// Tunnels - Duplex data tunnels with atomic wrappers
	Tunnels = &tunnels{
		tunnels: &map[uint64]*tunnel{},
		mutex:   &sync.RWMutex{},
	}
)

type tunnels struct {
	tunnels *map[uint64]*tunnel
	mutex   *sync.RWMutex
}

func (t *tunnels) Tunnel(ID uint64) *tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return (*t.tunnels)[ID]
}

func (t *tunnels) RecvTunnel(ID uint64, data []byte) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	tunnel := (*t.tunnels)[ID]
	(*tunnel).Recv <- data
}

func (t *tunnels) AddTunnel(tunnel *tunnel) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	(*t.tunnels)[tunnel.ID] = tunnel
}

func (t *tunnels) RemoveTunnel(ID uint64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(*t.tunnels, ID)
}

// tunnel - Duplex data tunnel
type tunnel struct {
	SliverID uint32
	ID       uint64
	Recv     chan []byte
}

func (t *tunnel) Send(data []byte) *sliverpb.Envelope {
	tunnelData := &sliverpb.TunnelData{
		SliverID: t.SliverID,
		TunnelID: t.ID,
		Data:     data,
	}
	rawTunnelData, _ := proto.Marshal(tunnelData)
	return &sliverpb.Envelope{
		Type: sliverpb.MsgTunnelData,
		Data: rawTunnelData,
	}
}

// SliverServer - Server info
type SliverServer struct {
	Send      chan *sliverpb.Envelope
	recv      chan *sliverpb.Envelope
	responses *map[uint64]chan *sliverpb.Envelope
	mutex     *sync.RWMutex
	Config    *assets.ClientConfig
}

// ResponseMapper - Maps recv'd envelopes to response channels
func (ss *SliverServer) ResponseMapper() {
	for envelope := range ss.recv {
		if envelope.ID != 0 {
			ss.mutex.Lock()
			if resp, ok := (*ss.responses)[envelope.ID]; ok {
				resp <- envelope
			}
			ss.mutex.Unlock()
		} else if envelope.Type == clientpb.MsgEvent {
			event := &clientpb.Event{}
			proto.Unmarshal(envelope.Data, event)
			Events <- event
		}
	}
}

// RequestResponse - Send a request envelope and wait for a response (blocking)
func (ss *SliverServer) RequestResponse(envelope *sliverpb.Envelope, timeout time.Duration) chan *sliverpb.Envelope {
	reqID := EnvelopeID()
	envelope.ID = reqID
	resp := make(chan *sliverpb.Envelope)
	ss.AddRespListener(reqID, resp)
	ss.Send <- envelope
	respCh := make(chan *sliverpb.Envelope)
	go func() {
		defer ss.RemoveRespListener(reqID)
		select {
		case respEnvelope := <-resp:
			respCh <- respEnvelope
		case <-time.After(timeout):
			respCh <- nil
		}
	}()
	return respCh
}

// AddRespListener - Add a response listener
func (ss *SliverServer) AddRespListener(envelopeID uint64, resp chan *sliverpb.Envelope) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	(*ss.responses)[envelopeID] = resp
}

// RemoveRespListener - Remove a listener
func (ss *SliverServer) RemoveRespListener(envelopeID uint64) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	close((*ss.responses)[envelopeID])
	delete((*ss.responses), envelopeID)
}

// BindSliverServer - Bind send/recv channels to a server
func BindSliverServer(send chan *sliverpb.Envelope, recv chan *sliverpb.Envelope) *SliverServer {
	return &SliverServer{
		Send:      send,
		recv:      recv,
		responses: &map[uint64]chan *sliverpb.Envelope{},
		mutex:     &sync.RWMutex{},
	}
}

// EnvelopeID - Generate random ID of randomIDSize bytes
func EnvelopeID() uint64 {
	randBuf := make([]byte, 8) // 64 bytes of randomness
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
