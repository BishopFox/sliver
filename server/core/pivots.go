package core

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	PivotTransportName = "pivot"
)

var (
	PivotSessions = &sync.Map{} // ID -> Pivot
)

// Pivot - Wraps an ImplantConnection
type Pivot struct {
	ID                   string
	OriginID             int64
	ImplantConn          *ImplantConnection
	ImmediateImplantConn *ImplantConnection
	CipherCtx            *cryptography.CipherContext
	Peers                []*sliverpb.PivotPeer
}

// Start - Starts the pivot send loop which forwards envelopes from the pivot ImplantConnection
// to the ImmediateImplantConnection (the closest peer in the chain)
func (p *Pivot) Start() {
	go func() {
		defer func() {
			coreLog.Debugf("pivot session %s send loop closing (origin id: %d)", p.ID, p.OriginID)
		}()
		for envelope := range p.ImplantConn.Send {
			envelopeData, err := proto.Marshal(envelope)
			if err != nil {
				coreLog.Errorf("failed to marshal envelope: %v", err)
				continue
			}
			ciphertext, err := p.CipherCtx.Encrypt(envelopeData)
			if err != nil {
				coreLog.Errorf("failed to encrypt envelope: %v", err)
				continue
			}
			peerEnvelopeData, _ := proto.Marshal(&sliverpb.PivotPeerEnvelope{
				Type:  envelope.Type,
				Peers: p.Peers,
				Data:  ciphertext,
			})
			if err != nil {
				coreLog.Errorf("failed to wrap pivot peer envelope: %v", err)
				continue
			}
			p.ImmediateImplantConn.Send <- &sliverpb.Envelope{
				Type: sliverpb.MsgPivotPeerEnvelope,
				Data: peerEnvelopeData,
			}
		}
	}()
}

// NewPivotSession - Creates a new pivot session
func NewPivotSession(chain []*sliverpb.PivotPeer) *Pivot {
	id, _ := uuid.NewV4()
	return &Pivot{
		ID:    id.String(),
		Peers: chain,
	}
}

// PivotGraphEntry - A single entry in the pivot graph
type PivotGraphEntry struct {
	PeerID    int64
	SessionID string
	Name      string

	// PeerID -> Child
	Children map[int64]*PivotGraphEntry
}

// ToProtobuf - Recursively converts the pivot graph to protobuf
func (e *PivotGraphEntry) ToProtobuf() *clientpb.PivotGraphEntry {
	children := []*clientpb.PivotGraphEntry{}
	for _, child := range e.Children {
		children = append(children, child.ToProtobuf())
	}
	return &clientpb.PivotGraphEntry{
		PeerID:   e.PeerID,
		Session:  Sessions.Get(e.SessionID).ToProtobuf(),
		Name:     e.Name,
		Children: children,
	}
}

// Insert - Inserts a pivot into the graph, if it doesn't yet exist
func (e *PivotGraphEntry) Insert(input *PivotGraphEntry) {
	if _, ok := e.Children[input.PeerID]; !ok {
		e.Children[input.PeerID] = input
	}
}

// FindEntryByPeerID - Finds a pivot graph entry by peer ID, recursively
func (e *PivotGraphEntry) FindEntryByPeerID(peerID int64) *PivotGraphEntry {
	if e.PeerID == peerID {
		return e
	}
	for _, entry := range e.Children {
		childEntry := entry.FindEntryByPeerID(peerID)
		if childEntry != nil {
			return childEntry
		}
	}
	return nil
}

// AllChildren - Flat list of all children (including children of children)
func (e *PivotGraphEntry) AllChildren() []*PivotGraphEntry {
	children := []*PivotGraphEntry{}
	coreLog.Debugf("parent: %v", e)
	for _, child := range e.Children {
		children = append(children, child)
		children = append(children, child.AllChildren()...)
	}
	coreLog.Debugf("parent: %v, all children: %v", e, children)
	return children
}

func findAllChildrenByPeerID(peerID int64) []*PivotGraphEntry {
	children := []*PivotGraphEntry{}
	for _, topLevelEntry := range PivotGraph() {
		entry := topLevelEntry.FindEntryByPeerID(peerID)
		coreLog.Debugf("top level entry: %v, found: %v", topLevelEntry, entry)
		if entry != nil {
			children = entry.AllChildren()
			break
		}
	}
	return children
}

// PivotGraph - Creates a graph structure of sessions/pivots
func PivotGraph() []*PivotGraphEntry {
	graph := []*PivotGraphEntry{}

	// Any non-pivot session will be top-level
	for _, session := range Sessions.All() {
		if session.Connection.Transport == PivotTransportName {
			continue
		}
		graph = append(graph, &PivotGraphEntry{
			PeerID:    session.PeerID,
			SessionID: session.ID,
			Name:      session.Name,
			Children:  make(map[int64]*PivotGraphEntry),
		})
	}
	for _, topLevel := range graph {
		coreLog.Debugf("[graph] top level: %v", topLevel)
		insertImmediateChildren(topLevel, 1)
	}
	return graph
}

// Remember that the origin index zero, and we only want this "entry"s immediate children
// the peers list will look something like this, where 2 is the parent of both 3, and 4
// 1
// 2,1
// 3,2,1
// 4,2,1
func insertImmediateChildren(entry *PivotGraphEntry, depth int) {
	// Iterate over pivots and insert them into the graph
	PivotSessions.Range(func(key, value interface{}) bool {
		pivot := value.(*Pivot)
		if len(pivot.Peers) != depth+1 {
			return true
		}
		session := Sessions.FromImplantConnection(pivot.ImplantConn)
		if session == nil {
			coreLog.Warnf("[graph] session not found for pivot: %v", pivot)
			return true
		}
		coreLog.Debugf("[graph] entry: %v, pivot: %v", entry.Name, pivot)
		if pivot.Peers[1].PeerID == entry.PeerID {
			child := &PivotGraphEntry{
				PeerID:    pivot.OriginID,
				SessionID: session.ID,
				Name:      session.Name,
				Children:  make(map[int64]*PivotGraphEntry),
			}
			coreLog.Debugf("[graph] entry: %v, found child: %v", entry.Name, child.Name)
			entry.Insert(child)
		}
		return true
	})

	// Recurse over children
	for _, child := range entry.Children {
		insertImmediateChildren(child, depth+1)
	}
}
