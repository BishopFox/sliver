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
	"crypto/x509"
	"sync"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
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
	ID          int
	Operator    string
	Certificate *x509.Certificate
	Send        chan *sliverpb.Envelope
	Resp        map[uint64]chan *sliverpb.Envelope
	mutex       *sync.RWMutex
}

// ToProtobuf - Get the protobuf version of the object
func (c *Client) ToProtobuf() *clientpb.Client {
	return &clientpb.Client{
		ID:       int32(c.ID),
		Operator: c.Operator,
	}
}

// Response - Drop an evelope into a response channel
func (c *Client) Response(envelope *sliverpb.Envelope) {
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
func GetClient(certificate *x509.Certificate) *Client {
	var operator string
	if certificate != nil {
		operator = certificate.Subject.CommonName
	} else {
		operator = "server"
	}
	return &Client{
		ID:          GetClientID(),
		Operator:    operator,
		Certificate: certificate,
		mutex:       &sync.RWMutex{},
		Send:        make(chan *sliverpb.Envelope),
		Resp:        map[uint64]chan *sliverpb.Envelope{},
	}
}
