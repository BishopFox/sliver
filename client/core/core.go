package core

import (
	"crypto/rand"
	"encoding/binary"
	"log"
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

type tunnels struct {
	server  *SliverServer
	tunnels *map[uint64]*tunnel
	mutex   *sync.RWMutex
}

func (t *tunnels) BindTunnel(SliverID uint32, TunnelID uint64) *tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	(*t.tunnels)[TunnelID] = &tunnel{
		server:   t.server,
		SliverID: SliverID,
		ID:       TunnelID,
		Recv:     make(chan []byte),
	}

	return (*t.tunnels)[TunnelID]
}

// RecvTunnelData - Routes a TunnelData protobuf msg to the correct tunnel object
func (t *tunnels) RecvTunnelData(tunnelData *sliverpb.TunnelData) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	tunnel := (*t.tunnels)[tunnelData.TunnelID]
	(*tunnel).Recv <- tunnelData.Data
}

func (t *tunnels) RemoveTunnel(ID uint64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(*t.tunnels, ID)
}

// tunnel - Duplex data tunnel
type tunnel struct {
	server   *SliverServer
	SliverID uint32
	ID       uint64
	Recv     chan []byte
}

func (t *tunnel) Send(data []byte) {
	tunnelData := &sliverpb.TunnelData{
		SliverID: t.SliverID,
		TunnelID: t.ID,
		Data:     data,
	}
	rawTunnelData, _ := proto.Marshal(tunnelData)
	t.server.Send <- &sliverpb.Envelope{
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
	Events    chan *clientpb.Event
	Tunnels   *tunnels
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
		} else {
			switch envelope.Type {

			case clientpb.MsgEvent:
				event := &clientpb.Event{}
				err := proto.Unmarshal(envelope.Data, event)
				if err != nil {
					log.Printf("Failed to decode event envelope")
					continue
				}
				ss.Events <- event

			case sliverpb.MsgTunnelData:
				tunnelData := &sliverpb.TunnelData{}
				err := proto.Unmarshal(envelope.Data, tunnelData)
				if err != nil {
					log.Printf("Failed to decode tunnel data envelope")
					continue
				}
				ss.Tunnels.RecvTunnelData(tunnelData)
			}
		}
	}
}

// RPC - Send a request envelope and wait for a response (blocking)
func (ss *SliverServer) RPC(envelope *sliverpb.Envelope, timeout time.Duration) chan *sliverpb.Envelope {
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
func BindSliverServer(send, recv chan *sliverpb.Envelope) *SliverServer {
	server := &SliverServer{
		Send:      send,
		recv:      recv,
		responses: &map[uint64]chan *sliverpb.Envelope{},
		mutex:     &sync.RWMutex{},
		Events:    make(chan *clientpb.Event, 1),
	}
	server.Tunnels = &tunnels{
		server:  server,
		tunnels: &map[uint64]*tunnel{},
		mutex:   &sync.RWMutex{},
	}
	return server
}

// EnvelopeID - Generate random ID
func EnvelopeID() uint64 {
	randBuf := make([]byte, 8) // 64 bits of randomness
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
