package core

import (
	pb "sliver/protobuf/client"
	"sync"
)

var (
	// Clients - Manages client connections
	Clients = &clientConns{
		Connections: &map[int]*Client{},
		mutex:       &sync.RWMutex{},
	}

	clientID = new(int)
)

// Client - Single client connection
type Client struct {
	ID       int
	Operator string

	Send  chan *pb.Envelope
	Resp  map[string]chan *pb.Envelope
	mutex *sync.RWMutex
}

// ToProtobuf - Get the protobuf version of the object
func (c *Client) ToProtobuf() *pb.Client {
	return &pb.Client{
		ID:       int32(c.ID),
		Operator: c.Operator,
	}
}

// Response - Drop an evelope into a response channel
func (c *Client) Response(envelope *pb.Envelope) {
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
		Send:     make(chan *pb.Envelope),
		Resp:     map[string]chan *pb.Envelope{},
	}
}
