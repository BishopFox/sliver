package core

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"sync"

	pb "sliver/protobuf"
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
	Send          chan *pb.Envelope
	Resp          map[string]chan *pb.Envelope
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

var (
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
