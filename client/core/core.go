package core

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"sliver/client/assets"
	"sync"
	"time"

	pb "sliver/protobuf/client"
)

const (
	randomIDSize = 16 // 64bits
)

var (
	// Events - Connect/Disconnect events
	Events = make(chan Event, 64)
)

// Event - Sliver connect/disconnect
type Event struct {
	Sliver    *pb.Sliver
	Job       *pb.Job
	EventType string
}

// SliverServer - Server info
type SliverServer struct {
	Send      chan *pb.Envelope
	recv      chan *pb.Envelope
	responses *map[string]chan *pb.Envelope
	mutex     *sync.RWMutex
	Config    *assets.ClientConfig
}

// ResponseMapper - Maps recv'd envelopes to response channels
func (ss *SliverServer) ResponseMapper() {
	for envelope := range ss.recv {
		if envelope.ID != "" {
			ss.mutex.Lock()
			if resp, ok := (*ss.responses)[envelope.ID]; ok {
				resp <- envelope
			}
			ss.mutex.Unlock()
		}
	}
}

// RequestResponse - Send a request envelope and wait for a response (blocking)
func (ss *SliverServer) RequestResponse(envelope *pb.Envelope, timeout time.Duration) *pb.Envelope {
	reqID := RandomID()
	envelope.ID = reqID
	resp := make(chan *pb.Envelope)
	ss.AddRespListener(reqID, resp)
	defer ss.RemoveRespListener(reqID)
	ss.Send <- envelope
	select {
	case respEnvelope := <-resp:
		return respEnvelope
	case <-time.After(timeout):
		return nil
	}
}

// AddRespListener - Add a response listener
func (ss *SliverServer) AddRespListener(requestID string, resp chan *pb.Envelope) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	(*ss.responses)[requestID] = resp
}

// RemoveRespListener - Remove a listener
func (ss *SliverServer) RemoveRespListener(requestID string) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	close((*ss.responses)[requestID])
	delete((*ss.responses), requestID)
}

// BindSliverServer - Bind send/recv channels to a server
func BindSliverServer(send chan *pb.Envelope, recv chan *pb.Envelope) *SliverServer {
	return &SliverServer{
		Send:      send,
		recv:      recv,
		responses: &map[string]chan *pb.Envelope{},
		mutex:     &sync.RWMutex{},
	}
}

// RandomID - Generate random ID of randomIDSize bytes
func RandomID() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:randomIDSize])
}
