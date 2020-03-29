package c2

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
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/core"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/golang/protobuf/proto"
)

var (
	pivotLog        = log.NamedLogger("c2", "pivot")
	pivotSliverTEST *core.Sliver

	// Pivots - holds the pivots, provides atomic access
	Pivots = &PivotsMap{
		Pivots: &map[uint32]*core.Sliver{},
		mutex:  &sync.RWMutex{},
	}
)

// StartPivotListener - Starts listening for pivot messages
func StartPivotListener() error {
	serverHandlers.AddSliverHandlers(pb.MsgPivotData, HandlePivotData)
	serverHandlers.AddSliverHandlers(pb.MsgPivotOpen, HandlePivotOpen)
	serverHandlers.AddSliverHandlers(pb.MsgPivotClose, HandlePivotClose)
	return nil
}

// HandlePivotData - Handles a PivotData message
func HandlePivotData(sliver *core.Sliver, data []byte) {
	envi := &pb.PivotData{}
	err2 := proto.Unmarshal(data, envi)
	if err2 != nil {
		pivotLog.Errorf("unmarshaling envelope error: %v", err2)
		return
	}
	envelope := &pb.Envelope{}
	err := proto.Unmarshal(envi.Data, envelope)
	if err != nil {
		pivotLog.Errorf("unmarshaling envelope error: %v", err)
		return
	}
	sliverPivoted := Pivots.Sliver(envi.GetPivotID())
	handlers := serverHandlers.GetSliverHandlers()
	if envelope.ID != 0 {
		sliverPivoted.RespMutex.RLock()
		if resp, ok := sliverPivoted.Resp[envelope.ID]; ok {
			resp <- envelope // Could deadlock, maybe want to investigate better solutions
			pivotLog.Printf("[PIVOT] Found envelope: %v\n", envelope)
		} else {
			pivotLog.Printf("[PIVOT] NotFound envelope: %v\n", envelope)
		}
		sliverPivoted.RespMutex.RUnlock()
	} else if handler, ok := handlers[envelope.Type]; ok {
		go handler.(func(*core.Sliver, []byte))(sliverPivoted, envelope.Data)
	}

}

// HandlePivotOpen - Handles a PivotOpen message
func HandlePivotOpen(sliver *core.Sliver, data []byte) {
	pivotOpen := &pb.PivotOpen{}
	err := proto.Unmarshal(data, pivotOpen)
	if err != nil {
		pivotLog.Errorf("unmarshaling envelope error: %v", err)
		return
	}
	pivotLog.Printf("HandlePivotOpen %v\n", pivotOpen)
	sliverPivoted := &core.Sliver{
		ID:            core.GetHiveID(),
		Transport:     pivotOpen.GetPivotType() + " (PIVOT)",
		RemoteAddress: pivotOpen.GetRemoteAddress(),
		Send:          make(chan *pb.Envelope),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *pb.Envelope{},
	}
	go func() {
		for envelope := range sliverPivoted.Send {
			originalEnvlopeData, _ := proto.Marshal(envelope)
			data, _ = proto.Marshal(&pb.PivotData{
				PivotID: pivotOpen.GetPivotID(),
				Data:    originalEnvlopeData,
			})
			sliver.Send <- &pb.Envelope{
				Type: pb.MsgPivotData,
				Data: data,
			}
		}
	}()
	Pivots.AddSliver(pivotOpen.GetPivotID(), sliverPivoted)
}

// HandlePivotClose - Handles a PivotClose message
func HandlePivotClose(sliver *core.Sliver, data []byte) {
	pivotClose := &pb.PivotClose{}
	err := proto.Unmarshal(data, pivotClose)
	if err != nil {
		pivotLog.Errorf("unmarshaling envelope error: %v", err)
		return
	}
	sliverPivoted := Pivots.Sliver(pivotClose.GetPivotID())
	pivotLog.Debugf("Cleaning up for %s", sliverPivoted.Name)
	core.Hive.RemoveSliver(sliverPivoted)
	core.EventBroker.Publish(core.Event{
		EventType: consts.DisconnectedEvent,
		Sliver:    sliverPivoted,
	})
}

// PivotsMap - Mananges the pivots, provides atomic access
type PivotsMap struct {
	mutex  *sync.RWMutex
	Pivots *map[uint32]*core.Sliver
}

// Sliver - Get Sliver by ID
func (h *PivotsMap) Sliver(pivotID uint32) *core.Sliver {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return (*h.Pivots)[pivotID]
}

// AddSliver - Add a sliver to the hive (atomically)
func (h *PivotsMap) AddSliver(pivotID uint32, sliver *core.Sliver) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	(*h.Pivots)[pivotID] = sliver
}

// RemoveSliver - Add a sliver to the hive (atomically)
func (h *PivotsMap) RemoveSliver(pivotID uint32, sliver *core.Sliver) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete((*h.Pivots), pivotID)
}
