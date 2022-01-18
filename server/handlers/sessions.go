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
	------------------------------------------------------------------------

	WARNING: These functions can be invoked by remote implants without user interaction

*/

import (
	"encoding/json"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"

	"google.golang.org/protobuf/proto"

	"github.com/google/uuid"
)

var (
	sessionHandlerLog = log.NamedLogger("handlers", "sessions")
)

func registerSessionHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	if implantConn == nil {
		return nil
	}
	register := &sliverpb.Register{}
	err := proto.Unmarshal(data, register)
	if err != nil {
		sessionHandlerLog.Errorf("Error decoding session registration message: %s", err)
		return nil
	}

	session := core.NewSession(implantConn)

	// Parse Register UUID
	sessionUUID, err := uuid.Parse(register.Uuid)
	if err != nil {
		sessionUUID = uuid.New() // Generate Random UUID
	}
	session.Name = register.Name
	session.Hostname = register.Hostname
	session.UUID = sessionUUID.String()
	session.Username = register.Username
	session.UID = register.Uid
	session.GID = register.Gid
	session.OS = register.Os
	session.Arch = register.Arch
	session.PID = register.Pid
	session.Filename = register.Filename
	session.ActiveC2 = register.ActiveC2
	session.Version = register.Version
	session.ReconnectInterval = register.ReconnectInterval
	session.ProxyURL = register.ProxyURL
	session.ConfigID = register.ConfigID
	session.PeerID = register.PeerID
	core.Sessions.Add(session)
	implantConn.Cleanup = func() {
		core.Sessions.Remove(session.ID)
	}
	go auditLogSession(session, register)
	return nil
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
		sessionHandlerLog.Errorf("Failed to log new session to audit log %s", err)
	} else {
		log.AuditLogger.Warn(string(msg))
	}
}

// The handler mutex prevents a send on a closed channel, without it
// two handlers calls may race when a tunnel is quickly created and closed.
func tunnelDataHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(data, tunnelData)
	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			tunnel.FromImplant <- tunnelData
		} else {
			sessionHandlerLog.Warnf("Warning: Session %d attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		sessionHandlerLog.Warnf("Data sent on nil tunnel %d", tunnelData.TunnelID)
	}
	return nil
}

func tunnelCloseHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()

	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(data, tunnelData)
	if !tunnelData.Closed {
		return nil
	}
	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			sessionHandlerLog.Infof("Closing tunnel %d", tunnel.ID)
			core.Tunnels.Close(tunnel.ID)
		} else {
			sessionHandlerLog.Warnf("Warning: Session %d attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		sessionHandlerLog.Warnf("Close sent on nil tunnel %d", tunnelData.TunnelID)
	}
	return nil
}

func pingHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	sessionHandlerLog.Debugf("ping from session %d", session.ID)
	return nil
}

func socksDataHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()
	socksData := &sliverpb.SocksData{}

	proto.Unmarshal(data, socksData)
	//if socksData.CloseConn{
	//	core.SocksTunnels.Close(socksData.TunnelID)
	//	return nil
	//}
	sessionHandlerLog.Debugf("socksDataHandler:", len(socksData.Data), socksData.Data)
	SocksTunne := core.SocksTunnels.Get(socksData.TunnelID)
	if SocksTunne != nil {
		if session.ID == SocksTunne.SessionID {
			SocksTunne.FromImplant <- socksData
		} else {
			sessionHandlerLog.Warnf("Warning: Session %d attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		sessionHandlerLog.Warnf("Data sent on nil tunnel %d", socksData.TunnelID)
	}
	return nil
}
