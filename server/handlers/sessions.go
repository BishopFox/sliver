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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"

	"github.com/golang/protobuf/proto"
)

var (
	handlerLog = log.NamedLogger("handlers", "sessions")

	sessionHandlers = map[uint32]interface{}{
		sliverpb.MsgRegister:    registerSessionHandler,
		sliverpb.MsgTunnelData:  tunnelDataHandler,
		sliverpb.MsgTunnelClose: tunnelCloseHandler,
	}
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


	handlerLog.Warnf("%v", session)
	handlerLog.Warnf("%v", register)

	session.Name = register.Name
	session.Hostname = register.Hostname
	session.Username = register.Username
	session.UID = register.Uid
	session.GID = register.Gid
	session.Os = register.Os
	session.Arch = register.Arch
	session.PID = register.Pid
	session.Filename = register.Filename
	session.ActiveC2 = register.ActiveC2
	session.Version = register.Version
	core.Sessions.Add(session)
}

func tunnelDataHandler(session *core.Session, data []byte) {
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(data, tunnelData)
	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			tunnel.FromImplant <- tunnelData.GetData()
		} else {
			handlerLog.Warnf("Warning: Session %d attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		handlerLog.Warnf("Data sent on nil tunnel %d", tunnelData.TunnelID)
	}
}

func tunnelCloseHandler(session *core.Session, data []byte) {
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
