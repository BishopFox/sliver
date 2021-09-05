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
	"encoding/json"
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/protobuf/proto"
)

var (
	pivotLog = log.NamedLogger("c2", "pivot")

	// Pivots - holds the pivots, provides atomic access
	Pivots = &PivotsMap{
		Pivots: &map[uint32]*core.Session{},
		mutex:  &sync.RWMutex{},
	}
)

// StartPivotListener - Starts listening for pivot messages
func StartPivotListener() error {
	// serverHandlers.AddHandler(sliverpb.MsgPivotData, HandlePivotData)
	// serverHandlers.AddHandler(sliverpb.MsgPivotOpen, HandlePivotOpen)
	// serverHandlers.AddHandler(sliverpb.MsgPivotClose, HandlePivotClose)
	return nil
}

// HandlePivotData - Handles a PivotData message
func HandlePivotData(session *core.Session, data []byte) {
	// envi := &sliverpb.PivotData{}
	// err2 := proto.Unmarshal(data, envi)
	// if err2 != nil {
	// 	pivotLog.Errorf("unmarshal envelope error: %v", err2)
	// 	return
	// }
	// envelope := &sliverpb.Envelope{}
	// err := proto.Unmarshal(envi.Data, envelope)
	// if err != nil {
	// 	pivotLog.Errorf("unmarshal envelope error: %v", err)
	// 	return
	// }
	// pivotLog.Printf("[PIVOT] XXXX: %v\n", envelope)
	// sliverPivoted := Pivots.Session(envi.GetPivotID())
	// handlers := serverHandlers.GetHandlers()
	// if envelope.ID != 0 {
	// 	sliverPivoted.RespMutex.RLock()
	// 	if resp, ok := sliverPivoted.Resp[envelope.ID]; ok {
	// 		resp <- envelope // Could deadlock, maybe want to investigate better solutions
	// 		pivotLog.Printf("[PIVOT] Found envelope: %v\n", envelope)
	// 	} else {
	// 		pivotLog.Printf("[PIVOT] NotFound envelope: %v\n", envelope)
	// 	}
	// 	sliverPivoted.RespMutex.RUnlock()
	// } else if handler, ok := handlers[envelope.Type]; ok {
	// 	go handler.(func(*core.Session, []byte))(sliverPivoted, envelope.Data)
	// }

}

// HandlePivotOpen - Handles a PivotOpen message
func HandlePivotOpen(session *core.Session, data []byte) {
	// pivotOpen := &sliverpb.PivotOpen{}
	// err := proto.Unmarshal(data, pivotOpen)
	// if err != nil {
	// 	pivotLog.Errorf("unmarshal envelope error: %v", err)
	// 	return
	// }
	// registerEnvelope := &sliverpb.Envelope{}
	// err = proto.Unmarshal(pivotOpen.RegisterMsg, registerEnvelope)
	// if err != nil {
	// 	pivotLog.Warnf("error decoding message: %v", err)
	// 	return
	// }
	// register := &sliverpb.Register{}
	// err = proto.Unmarshal(registerEnvelope.Data, register)
	// if err != nil {
	// 	pivotLog.Warnf("error decoding message: %v", err)
	// 	return
	// }
	// pivotLog.Warnf("HandlePivotOpen %v %v %s\n", pivotOpen, register, register.Name)
	// sliverPivoted := &core.Session{
	// 	ID:            core.NextSessionID(),
	// 	Transport:     pivotOpen.GetPivotType() + " (PIVOT)",
	// 	RemoteAddress: pivotOpen.GetRemoteAddress(),
	// 	Send:          make(chan *sliverpb.Envelope),
	// 	RespMutex:     &sync.RWMutex{},
	// 	Resp:          map[uint64]chan *sliverpb.Envelope{},
	// 	Name:          register.Name,
	// 	Hostname:      register.Hostname,
	// 	UUID:          register.Uuid,
	// 	Username:      register.Username,
	// 	UID:           register.Uid,
	// 	GID:           register.Gid,
	// 	Os:            register.Os,
	// 	Arch:          register.Arch,
	// 	PID:           register.Pid,
	// 	Filename:      register.Filename,
	// 	ActiveC2:      register.ActiveC2,
	// 	Version:       register.Version,
	// }
	// go func() {
	// 	for envelope := range sliverPivoted.Send {
	// 		originalEnvlopeData, _ := proto.Marshal(envelope)
	// 		data, _ = proto.Marshal(&sliverpb.PivotData{
	// 			PivotID: pivotOpen.GetPivotID(),
	// 			Data:    originalEnvlopeData,
	// 		})
	// 		session.Send <- &sliverpb.Envelope{
	// 			Type: sliverpb.MsgPivotData,
	// 			Data: data,
	// 		}
	// 	}
	// }()
	// core.Sessions.Add(sliverPivoted)
	// go auditLogSession(sliverPivoted, register)
	// Pivots.AddSession(pivotOpen.GetPivotID(), sliverPivoted)
}

type auditLogNewSessionMsg struct {
	Session  *clientpb.Session
	Register *sliverpb.Register
}

func auditLogSession(session *core.Session, register *sliverpb.Register) {
	msg, err := json.Marshal(auditLogNewSessionMsg{
		Session:  session.ToProtobuf(),
		Register: register,
	})
	if err != nil {
		pivotLog.Errorf("Failed to log new session to audit log %s", err)
	} else {
		log.AuditLogger.Warn(string(msg))
	}
}

// HandlePivotClose - Handles a PivotClose message
func HandlePivotClose(session *core.Session, data []byte) {
	pivotClose := &sliverpb.PivotClose{}
	err := proto.Unmarshal(data, pivotClose)
	if err != nil {
		pivotLog.Errorf("unmarshal envelope error: %v", err)
		return
	}
	sliverPivoted := Pivots.Session(pivotClose.GetPivotID())
	if sliverPivoted != nil {
		pivotLog.Debugf("Cleaning up for %s", sliverPivoted.Name)
		core.Sessions.Remove(sliverPivoted.ID)
	}
	Pivots.RemoveSession(pivotClose.GetPivotID())
	/*core.EventBroker.Publish(core.Event{
		EventType: consts.DisconnectedEvent,
		Sliver:    sliverPivoted,
	})*/
}

// PivotsMap - Manages the pivots, provides atomic access
type PivotsMap struct {
	mutex  *sync.RWMutex
	Pivots *map[uint32]*core.Session
}

// Session - Get Session by ID
func (h *PivotsMap) Session(pivotID uint32) *core.Session {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return (*h.Pivots)[pivotID]
}

// AddSession - Add a sliver to the core (atomically)
func (h *PivotsMap) AddSession(pivotID uint32, session *core.Session) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	(*h.Pivots)[pivotID] = session
}

// RemoveSession - Remove a session from the core (atomically)
func (h *PivotsMap) RemoveSession(pivotID uint32) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete((*h.Pivots), pivotID)
}
