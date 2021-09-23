package handlers

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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
)

var (
	pivotLog = log.NamedLogger("handlers", "pivot")
	pivots   = &Pivots{
		mutex:  &sync.RWMutex{},
		Pivots: make(map[uint32]*core.Session),
	}
)

// Pivots - Manages the pivots, provides atomic access
type Pivots struct {
	mutex  *sync.RWMutex
	Pivots map[uint32]*core.Session
}

// AddSession - Add a sliver to the core (atomically)
func (p *Pivots) AddSession(pivotID uint32, session *core.Session) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.Pivots[pivotID] = session
}

// RemoveSession - Remove a session from the core (atomically)
func (p *Pivots) RemoveSession(pivotID uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.Pivots, pivotID)
}

// StartPivotHandlers - Starts listening for pivot messages
func StartPivotHandlers() error {
	serverHandlers[sliverpb.MsgPivotOpen] = HandlePivotOpen
	serverHandlers[sliverpb.MsgPivotData] = HandlePivotData
	serverHandlers[sliverpb.MsgPivotClose] = HandlePivotClose
	return nil
}

// StopPivotHandlers - Starts listening for pivot messages
func StopPivotHandlers() error {
	if _, ok := serverHandlers[sliverpb.MsgPivotOpen]; ok {
		delete(serverHandlers, sliverpb.MsgPivotOpen)
		delete(serverHandlers, sliverpb.MsgPivotData)
		delete(serverHandlers, sliverpb.MsgPivotClose)
	}
	return nil
}

// HandlePivotOpen - Handles a PivotOpen message
func HandlePivotOpen(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	// if implantConn == nil {
	// 	return
	// }
	// pivotOpen := &sliverpb.PivotOpen{}
	// err := proto.Unmarshal(data, pivotOpen)
	// if err != nil {
	// 	pivotLog.Errorf("unmarshal envelope error: %v", err)
	// 	return
	// }
	// _, privateKeyPEM, err := certs.GetCertificateAuthorityPEM(certs.PivotCA)
	// if err != nil {
	// 	pivotLog.Errorf("error retrieving pivot public key: %v", err)
	// 	return
	// }
	// privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	// privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	// cryptography.RSADecrypt(pivotOpen.RegisterMsg, privateKey)

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
	// 	Name:     register.Name,
	// 	Hostname: register.Hostname,
	// 	UUID:     register.Uuid,
	// 	Username: register.Username,
	// 	UID:      register.Uid,
	// 	GID:      register.Gid,
	// 	Os:       register.Os,
	// 	Arch:     register.Arch,
	// 	PID:      register.Pid,
	// 	Filename: register.Filename,
	// 	ActiveC2: register.ActiveC2,
	// 	Version:  register.Version,
	// }
	// go func() {
	// 	for envelope := range sliverPivoted.Send {
	// 		originalEnvelope, _ := proto.Marshal(envelope)
	// 		data, _ = proto.Marshal(&sliverpb.PivotData{
	// 			PivotID: pivotOpen.GetPivotID(),
	// 			Data:    originalEnvelope,
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
	return nil
}

// HandlePivotData - Handles a PivotData message
func HandlePivotData(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
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
	return nil
}

// HandlePivotClose - Handles a PivotClose message
func HandlePivotClose(conn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	// pivotClose := &sliverpb.PivotClose{}
	// err := proto.Unmarshal(data, pivotClose)
	// if err != nil {
	// 	pivotLog.Errorf("unmarshal envelope error: %v", err)
	// 	return
	// }
	// sliverPivoted := Pivots.Session(pivotClose.GetPivotID())
	// if sliverPivoted != nil {
	// 	pivotLog.Debugf("Cleaning up for %s", sliverPivoted.Name)
	// 	core.Sessions.Remove(sliverPivoted.ID)
	// }
	// Pivots.RemoveSession(pivotClose.GetPivotID())
	return nil
}
