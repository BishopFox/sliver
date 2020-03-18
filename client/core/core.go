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
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

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

func (t *tunnels) bindTunnel(SliverID uint32, TunnelID uint64) *tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	(*t.tunnels)[TunnelID] = &tunnel{
		server:   t.server,
		SliverID: SliverID,
		ID:       TunnelID,
		Recv:     make(chan []byte),
		isOpen:   true,
	}

	return (*t.tunnels)[TunnelID]
}

// RecvTunnelData - Routes a TunnelData protobuf msg to the correct tunnel object
func (t *tunnels) RecvTunnelData(tunnelData *sliverpb.TunnelData) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	tunnel := (*t.tunnels)[tunnelData.TunnelID]
	if tunnel != nil {
		(*tunnel).Recv <- tunnelData.Data
	} else {
		log.Printf("No client tunnel with ID %d", tunnelData.TunnelID)
	}
}

func (t *tunnels) Close(ID uint64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	(*t.tunnels)[ID].isOpen = false
	close((*t.tunnels)[ID].Recv)
	delete(*t.tunnels, ID)
}

// tunnel - Duplex data tunnel
type tunnel struct {
	server   *SliverServer
	SliverID uint32
	ID       uint64
	Recv     chan []byte
	isOpen   bool
}

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

func (t *tunnel) Write(data []byte) (n int, err error) {
	log.Printf("Sending %d bytes on tunnel %d (sliver %d)", len(data), t.ID, t.SliverID)
	if !t.isOpen {
		return 0, io.EOF
	}
	tunnelData := &sliverpb.TunnelData{
		SliverID: t.SliverID,
		TunnelID: t.ID,
		Data:     data,
	}
	rawTunnelData, err := proto.Marshal(tunnelData)
	t.server.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgTunnelData,
		Data: rawTunnelData,
	}
	n = len(data)
	return
}

func (t *tunnel) Read(data []byte) (n int, err error) {
	var buff bytes.Buffer
	if !t.isOpen {
		return 0, io.EOF
	}
	select {
	case msg := <-t.Recv:
		buff.Write(msg)
	default:
		break
	}
	n = copy(data, buff.Bytes())
	return
}

func (t *tunnel) Close() error {
	tunnelClose, err := proto.Marshal(&sliverpb.ShellReq{
		TunnelID: t.ID,
	})
	t.server.RPC(&sliverpb.Envelope{
		Type: sliverpb.MsgTunnelClose,
		Data: tunnelClose,
	}, 30*time.Second)
	close(t.Recv)
	return err
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

// CreateTunnel - Create a new tunnel on the server, returns tunnel metadata
func (ss *SliverServer) CreateTunnel(sliverID uint32, defaultTimeout time.Duration) (*tunnel, error) {
	tunReq := &clientpb.TunnelCreateReq{SliverID: sliverID}
	tunReqData, _ := proto.Marshal(tunReq)

	tunResp := <-ss.RPC(&sliverpb.Envelope{
		Type: clientpb.MsgTunnelCreate,
		Data: tunReqData,
	}, defaultTimeout)
	if tunResp.Err != "" {
		return nil, fmt.Errorf("Error: %s", tunResp.Err)
	}

	tunnelCreated := &clientpb.TunnelCreate{}
	proto.Unmarshal(tunResp.Data, tunnelCreated)

	tunnel := ss.Tunnels.bindTunnel(tunnelCreated.SliverID, tunnelCreated.TunnelID)

	log.Printf("Created new tunnel with ID %d", tunnel.ID)

	return tunnel, nil
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
			// If the message does not have an envelope ID then we route it based on type
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

			case sliverpb.MsgTunnelClose:
				tunnelClose := &sliverpb.TunnelClose{}
				err := proto.Unmarshal(envelope.Data, tunnelClose)
				if err != nil {
					log.Printf("Failed to decode tunnel data envelope")
					continue
				}
				ss.Tunnels.Close(tunnelClose.TunnelID)

			}
		}
	}
}

// RPC - Send a request envelope and wait for a response (blocking)
func (ss *SliverServer) RPC(envelope *sliverpb.Envelope, timeout time.Duration) chan *sliverpb.Envelope {
	reqID := EnvelopeID()
	envelope.ID = reqID
	envelope.Timeout = timeout.Nanoseconds()
	resp := make(chan *sliverpb.Envelope)
	ss.AddRespListener(reqID, resp)
	ss.Send <- envelope
	respCh := make(chan *sliverpb.Envelope)
	go func() {
		defer ss.RemoveRespListener(reqID)
		select {
		case respEnvelope := <-resp:
			respCh <- respEnvelope
		case <-time.After(timeout + time.Second):
			respCh <- &sliverpb.Envelope{Err: "Timeout"}
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
