package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	clientpb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
)

const (
	// randomIDSize - Size of the TunnelID in bytes
	randomIDSize = 8
)

// Sliver implant
type Sliver struct {
	ID            int
	Name          string
	Hostname      string
	Username      string
	UID           string
	GID           string
	Os            string
	Arch          string
	Transport     string
	RemoteAddress string
	PID           int32
	Filename      string
	Send          chan *sliverpb.Envelope
	Resp          map[uint64]chan *sliverpb.Envelope
	RespMutex     *sync.RWMutex
}

// Request - Sends a protobuf request to the active sliver and returns the response
func (s *Sliver) Request(msgType uint32, timeout time.Duration, data []byte) ([]byte, error) {

	resp := make(chan *sliverpb.Envelope)
	reqID := EnvelopeID()
	s.RespMutex.Lock()
	s.Resp[reqID] = resp
	s.RespMutex.Unlock()
	defer func() {
		s.RespMutex.Lock()
		defer s.RespMutex.Unlock()
		close(resp)
		delete(s.Resp, reqID)
	}()
	s.Send <- &sliverpb.Envelope{
		ID:   reqID,
		Type: msgType,
		Data: data,
	}

	var respEnvelope *sliverpb.Envelope
	select {
	case respEnvelope = <-resp:
	case <-time.After(timeout):
		return nil, errors.New("timeout")
	}
	return respEnvelope.Data, nil
}

// SliverHive - Mananges the slivers, provides atomic access
type SliverHive struct {
	mutex   *sync.RWMutex
	Slivers *map[int]*Sliver
}

// AddSliver - Add a sliver to the hive (atomically)
func (h *SliverHive) AddSliver(sliver *Sliver) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	(*h.Slivers)[sliver.ID] = sliver
}

// RemoveSliver - Add a sliver to the hive (atomically)
func (h *SliverHive) RemoveSliver(sliver *Sliver) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete((*h.Slivers), sliver.ID)
}

// Job - Manages background jobs
type Job struct {
	ID          int
	Name        string
	Description string
	Protocol    string
	Port        uint16
	JobCtrl     chan bool
}

// Event - Sliver connect/disconnect
type Event struct {
	Sliver    *Sliver
	Job       *Job
	EventType string
}

// Client - Single client connection
type Client struct {
	ID       int
	Operator string

	Send  chan *clientpb.Envelope
	Resp  map[string]chan *clientpb.Envelope
	mutex *sync.RWMutex
}

// Response - Drop an evelope into a response channel
func (c *Client) Response(envelope *clientpb.Envelope) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if resp, ok := c.Resp[envelope.ID]; ok {
		resp <- envelope
	}
}

// clientConns - Manage client connections
type clientConns struct {
	mutex       *sync.RWMutex
	Connections *map[int]*Client
}

// AddClient - Add a client struct atomically
func (cc *clientConns) AddClient(client *Client) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	(*cc.Connections)[client.ID] = client
}

// RemoveClient - Remove a client struct atomically
func (cc *clientConns) RemoveClient(clientID int) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	delete((*cc.Connections), clientID)
}

var (
	// Clients - Manages client connections
	Clients = &clientConns{
		Connections: &map[int]*Client{},
		mutex:       &sync.RWMutex{},
	}

	// Hive - Manages sliver connections
	Hive = &SliverHive{
		Slivers: &map[int]*Sliver{},
		mutex:   &sync.RWMutex{},
	}

	hiveID = new(int)

	// JobMutex - Controls access to the Jobs map
	JobMutex = &sync.RWMutex{}
	// Jobs - Holds pointers to all the current jobs
	Jobs  = &map[int]*Job{}
	jobID = new(int)

	clientID = new(int)

	// Events - Connect/Disconnect events
	Events = make(chan Event, 64)
)

// RandomID - Generate random ID of randomIDSize bytes
func RandomID() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:randomIDSize])
}

// EnvelopeID - Generate random ID of randomIDSize bytes
func EnvelopeID() uint64 {
	randBuf := make([]byte, 8) // 64 bytes of randomness
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}

// GetHiveID - Returns an incremental nonce as an id
func GetHiveID() int {
	newID := (*hiveID) + 1
	(*hiveID)++
	return newID
}

// GetJobID - Returns an incremental nonce as an id
func GetJobID() int {
	newID := (*jobID) + 1
	(*jobID)++
	return newID
}

// GetClientID - Get a client ID
func GetClientID() int {
	newID := (*clientID) + 1
	(*clientID)++
	return newID
}

// GetClient - Create a new client object
func GetClient(operator string) *Client {
	return &Client{
		ID:       GetClientID(),
		Operator: operator,
		mutex:    &sync.RWMutex{},
		Send:     make(chan *clientpb.Envelope),
		Resp:     map[string]chan *clientpb.Envelope{},
	}
}
