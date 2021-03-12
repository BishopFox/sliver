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

	---
	WARNING: These functions can be invoked by remote implants without user interaction
	---
*/

import (
	"encoding/json"
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"

	"github.com/golang/protobuf/proto"

	"github.com/google/uuid"
)

var (
	handlerLog = log.NamedLogger("handlers", "sessions")

	sessionHandlers = map[uint32]interface{}{
		sliverpb.MsgRegister:    registerSessionHandler,
		sliverpb.MsgTunnelData:  tunnelDataHandler,
		sliverpb.MsgTunnelClose: tunnelCloseHandler,
		sliverpb.MsgPing:        pingHandler,
	}

	tunnelHandlerMutex = &sync.Mutex{}
)

// GetSessionHandlers - Returns a map of server-side msg handlers
func GetSessionHandlers() map[uint32]interface{} {
	return sessionHandlers
}

// AddSessionHandlers -  Adds a new handler to the map of server-side msg handlers
func AddSessionHandlers(key uint32, value interface{}) {
	sessionHandlers[key] = value
}

func registerSessionHandler(session *core.Session, data []byte) {
	register := &sliverpb.Register{}
	err := proto.Unmarshal(data, register)
	if err != nil {
		handlerLog.Warnf("error decoding message: %v", err)
		return
	}

	if session == nil {
		return
	}

	if session.ID == 0 {
		session.ID = core.NextSessionID()
	}

	// Parse Register UUID
	sessionUUID, err := uuid.Parse(register.Uuid)
	if err != nil {
		// Generate Random UUID
		sessionUUID = uuid.New()
	}

	session.Name = register.Name
	session.Hostname = register.Hostname
	session.UUID = sessionUUID.String()
	session.Username = register.Username
	session.UID = register.Uid
	session.GID = register.Gid
	session.Os = register.Os
	session.Arch = register.Arch
	session.PID = register.Pid
	session.Filename = register.Filename
	session.ActiveC2 = register.ActiveC2
	session.Version = register.Version
	session.ReconnectInterval = register.ReconnectInterval
	session.ProxyURL = register.ProxyURL
	core.Sessions.Add(session)
	go auditLogSession(session, register)
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
		handlerLog.Errorf("Failed to log new session to audit log %s", err)
	} else {
		log.AuditLogger.Warn(string(msg))
	}
}

// The handler mutex prevents a send on a closed channel, without it
// two handlers calls may race when a tunnel is quickly created and closed.
func tunnelDataHandler(session *core.Session, data []byte) {
	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()

	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(data, tunnelData)
	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			tunnel.FromImplant <- tunnelData
		} else {
			handlerLog.Warnf("Warning: Session %d attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		handlerLog.Warnf("Data sent on nil tunnel %d", tunnelData.TunnelID)
	}
}

func tunnelCloseHandler(session *core.Session, data []byte) {
	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()

	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(data, tunnelData)
	if !tunnelData.Closed {
		return
	}
	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			handlerLog.Infof("Closing tunnel %d", tunnel.ID)
			core.Tunnels.Close(tunnel.ID)
		} else {
			handlerLog.Warnf("Warning: Session %d attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		handlerLog.Warnf("Close sent on nil tunnel %d", tunnelData.TunnelID)
	}
}

func pingHandler(session *core.Session, data []byte) {
	handlerLog.Infof("ping from session %d", session.ID)
}
