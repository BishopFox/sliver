package core

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"sync"

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
	Resp          map[string]chan *sliverpb.Envelope
	RespMutex     *sync.RWMutex
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
	if resp, ok := c.Resp[envelope.Id]; ok {
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
	Clients = &clientConns{}

	// HiveMutex - Controls access to Hive map
	HiveMutex = &sync.RWMutex{}
	// Hive - Holds all the slivers pointers
	Hive   = &map[int]*Sliver{}
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
