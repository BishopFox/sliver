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
	"context"
	"encoding/json"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
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
	session.Locale = register.Locale
	session.ConfigID = register.ConfigID
	session.PeerID = register.PeerID
	session.Capabilities = register.Capabilities
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
	if session == nil {
		sessionHandlerLog.Warnf("Received tunnel data from unknown session: %v", implantConn)
		return nil
	}
	tunnelData := &sliverpb.TunnelData{}
	if err := proto.Unmarshal(data, tunnelData); err != nil {
		sessionHandlerLog.Warnf("Failed to decode tunnel data from session %s: %v", session.ID, err)
		return nil
	}

	sessionHandlerLog.Debugf("[DATA] Sequence on tunnel %d, %d, data: %s", tunnelData.TunnelID, tunnelData.Sequence, tunnelData.Data)
	if tunnelData.CreateReverse {
		opening := &reverseTunnelOpening{sessionID: session.ID, ready: make(chan struct{})}
		actual, loaded := reverseTunnelOpenings.LoadOrStore(tunnelData.TunnelID, opening)
		if loaded {
			existing := actual.(*reverseTunnelOpening)
			if existing.sessionID != session.ID {
				sessionHandlerLog.Warnf("Session %s attempted to reuse reverse tunnel ID %d owned by another session", session.ID, tunnelData.TunnelID)
				return nil
			}
			<-existing.ready
			return nil
		}
		defer func() {
			close(opening.ready)
			reverseTunnelOpenings.Delete(tunnelData.TunnelID)
		}()
		return createReverseTunnelHandler(implantConn, tunnelData)
	}

	if value, ok := reverseTunnelOpenings.Load(tunnelData.TunnelID); ok {
		opening := value.(*reverseTunnelOpening)
		if opening.sessionID != session.ID {
			sessionHandlerLog.Warnf("Session %s attempted to send data on opening reverse tunnel %d owned by another session", session.ID, tunnelData.TunnelID)
			return nil
		}
		<-opening.ready
	}

	rtunnel := rtunnels.GetRTunnel(tunnelData.TunnelID)
	if rtunnel != nil && session.ID == rtunnel.SessionID {
		RTunnelDataHandler(tunnelData, rtunnel, implantConn)
		return nil
	} else if rtunnel != nil && session.ID != rtunnel.SessionID {
		sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on reverse tunnel it did not own", session.ID)
		return nil
	}

	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()
	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			tunnel.SendDataFromImplant(tunnelData)
		} else {
			sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		sessionHandlerLog.Warnf("Data sent on nil tunnel %d", tunnelData.TunnelID)
	}

	return nil
}

func tunnelCloseHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	if session == nil {
		sessionHandlerLog.Warnf("Received tunnel close from unknown session: %v", implantConn)
		return nil
	}
	tunnelData := &sliverpb.TunnelData{}
	if err := proto.Unmarshal(data, tunnelData); err != nil {
		sessionHandlerLog.Warnf("Failed to decode tunnel close from session %s: %v", session.ID, err)
		return nil
	}
	sessionHandlerLog.Debugf("[CLOSE] Sequence on tunnel %d, %d, data: %s", tunnelData.TunnelID, tunnelData.Sequence, tunnelData.Data)
	if !tunnelData.Closed {
		return nil
	}
	if value, ok := reverseTunnelOpenings.Load(tunnelData.TunnelID); ok {
		opening := value.(*reverseTunnelOpening)
		if opening.sessionID != session.ID {
			sessionHandlerLog.Warnf("Session %s attempted to close opening reverse tunnel %d owned by another session", session.ID, tunnelData.TunnelID)
			return nil
		}
		<-opening.ready
	}

	tunnelHandlerMutex.Lock()
	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			sessionHandlerLog.Infof("Closing tunnel %d", tunnel.ID)
			// Tunnel close and final data envelopes are dispatched concurrently.
			// Start a fresh quiet period so an overtaking close cannot discard the
			// last frame from an otherwise idle shell.
			tunnel.Touch()
			go core.Tunnels.ScheduleClose(tunnel.ID)
		} else {
			sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on tunnel it did not own", session.ID)
		}
		tunnelHandlerMutex.Unlock()
		return nil
	}
	tunnelHandlerMutex.Unlock()

	rtunnel := rtunnels.GetRTunnel(tunnelData.TunnelID)
	if rtunnel != nil && session.ID == rtunnel.SessionID {
		closeReverseTunnel(rtunnel)
	} else if rtunnel != nil && session.ID != rtunnel.SessionID {
		sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on reverse tunnel it did not own", session.ID)
	} else {
		sessionHandlerLog.Warnf("Close sent on nil tunnel %d", tunnelData.TunnelID)
	}
	return nil
}

func pingHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	if session == nil {
		sessionHandlerLog.Warnf("Received ping from unknown session: %v", implantConn)
		return nil
	}
	sessionHandlerLog.Debugf("ping from session %s", session.ID)
	return nil
}

func socksDataHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	if session == nil {
		sessionHandlerLog.Warnf("Received socks data from unknown session: %v", implantConn)
		return nil
	}
	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()
	socksData := &sliverpb.SocksData{}

	proto.Unmarshal(data, socksData)
	//if socksData.CloseConn{
	//	core.SocksTunnels.Close(socksData.TunnelID)
	//	return nil
	//}
	sessionHandlerLog.Debugf("socksDataHandler: %d bytes: %v", len(socksData.Data), socksData.Data)
	socksTunnel := core.SocksTunnels.Get(socksData.TunnelID)
	if socksTunnel != nil {
		if session.ID == socksTunnel.SessionID {
			socksTunnel.FromImplant <- socksData
		} else {
			sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		sessionHandlerLog.Warnf("Data sent on nil tunnel %d", socksData.TunnelID)
	}
	return nil
}

func createReverseTunnelHandler(implantConn *core.ImplantConnection, req *sliverpb.TunnelData) *sliverpb.Envelope {
	return createReverseTunnelHandlerWithBroker(implantConn, req, rtunnels.DefaultBroker)
}

func createReverseTunnelHandlerWithBroker(implantConn *core.ImplantConnection, req *sliverpb.TunnelData, broker *rtunnels.Broker) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	if session == nil || req == nil || req.Rportfwd == nil || broker == nil {
		sessionHandlerLog.Warnf("Rejected malformed reverse tunnel creation request")
		return nil
	}

	legacyAddress := ""
	if req.Rportfwd.AuthorizationID == "" {
		legacyAddress = net.JoinHostPort(req.Rportfwd.Host, strconv.FormatUint(uint64(req.Rportfwd.Port), 10))
	}
	dst, resolvedAuthorizationID, err := broker.Open(
		context.Background(),
		session.ID,
		rtunnels.AuthorizationID(req.Rportfwd.AuthorizationID),
		legacyAddress,
	)
	if err != nil {
		rejectReverseTunnel(implantConn, req.TunnelID, err)
		return nil
	}

	tunnel := rtunnels.NewAuthorizedRTunnel(req.TunnelID, session.ID, resolvedAuthorizationID, dst, dst)
	tunnelHandlerMutex.Lock()
	if core.Tunnels.Get(req.TunnelID) != nil {
		tunnelHandlerMutex.Unlock()
		tunnel.Close()
		rejectReverseTunnel(implantConn, req.TunnelID, rtunnels.ErrDuplicateTunnelID)
		return nil
	}
	if !rtunnels.TryAddRTunnel(tunnel) {
		tunnelHandlerMutex.Unlock()
		tunnel.Close()
		rejectReverseTunnel(implantConn, req.TunnelID, rtunnels.ErrDuplicateTunnelID)
		return nil
	}
	tunnelHandlerMutex.Unlock()

	var cleanupOnce sync.Once
	cleanup := func(reason error) {
		cleanupOnce.Do(func() {
			// {{if .Config.Debug}}
			sessionHandlerLog.Infof("[portfwd] Closing tunnel %d (%v)", tunnel.ID, reason)
			// {{end}}
			closeReverseTunnel(tunnel)
		})
	}

	go func() {
		tWriter := tunnelWriter{
			tun:  tunnel,
			conn: implantConn,
		}
		// portfwd only uses one reader, hence the tunnel.Readers[0]
		n, err := io.Copy(tWriter, tunnel.Readers[0])
		_ = n // avoid not used compiler error if debug mode is disabled
		// {{if .Config.Debug}}
		sessionHandlerLog.Infof("[tunnel] Tunnel done, wrote %v bytes", n)
		// {{end}}

		cleanup(err)
	}()
	RTunnelDataHandler(req, tunnel, implantConn)
	return nil
}

func rejectReverseTunnel(implantConn *core.ImplantConnection, tunnelID uint64, reason error) {
	sessionHandlerLog.Warnf("Rejected reverse tunnel %d: %v", tunnelID, reason)
	data, err := proto.Marshal(&sliverpb.TunnelData{Closed: true, TunnelID: tunnelID})
	if err != nil || implantConn == nil {
		return
	}
	select {
	case implantConn.Send <- &sliverpb.Envelope{Type: sliverpb.MsgTunnelClose, Data: data}:
	default:
		sessionHandlerLog.Warnf("Could not queue close for rejected reverse tunnel %d", tunnelID)
	}
}

func RTunnelDataHandler(tunnelData *sliverpb.TunnelData, tunnel *rtunnels.RTunnel, connection *core.ImplantConnection) {
	if tunnelData == nil || tunnel == nil || connection == nil || tunnel.Writer == nil {
		return
	}
	pending, err := tunnel.ProcessInbound(tunnelData.Sequence, tunnelData.Data, func(payload []byte) error {
		if deadlineWriter, ok := tunnel.Writer.(interface{ SetWriteDeadline(time.Time) error }); ok {
			_ = deadlineWriter.SetWriteDeadline(time.Now().Add(rtunnels.DefaultDialTimeout))
			defer deadlineWriter.SetWriteDeadline(time.Time{})
		}
		written, writeErr := tunnel.Writer.Write(payload)
		if writeErr != nil {
			return writeErr
		}
		if written != len(payload) {
			return io.ErrShortWrite
		}
		return nil
	})
	if err != nil {
		sessionHandlerLog.Warnf("Closing reverse tunnel %d after bounded inbound relay failure: %v", tunnel.ID, err)
		closeReverseTunnel(tunnel)
		rejectReverseTunnel(connection, tunnel.ID, err)
		return
	}

	// If the bounded per-tunnel cache is building up, request the missing frame.
	if pending > 3 {
		data, err := proto.Marshal(&sliverpb.TunnelData{
			Sequence: tunnel.WriteSequence(), // The tunnel write sequence
			Ack:      tunnel.ReadSequence(),
			Resend:   true,
			TunnelID: tunnel.ID,
			Data:     []byte{},
		})
		if err != nil {
			// {{if .Config.Debug}}
			//sessionHandlerLog.Infof("[shell] Failed to marshal protobuf %s", err)
			// {{end}}
		} else {
			// {{if .Config.Debug}}
			//sessionHandlerLog.Infof("[tunnel] Requesting resend of tunnelData seq: %d", tunnel.ReadSequence())
			// {{end}}
			if err := queueTunnelEnvelope(connection, tunnel, &sliverpb.Envelope{Type: sliverpb.MsgTunnelData, Data: data}); err != nil {
				sessionHandlerLog.Warnf("Closing reverse tunnel %d after resend queue failure: %v", tunnel.ID, err)
				closeReverseTunnel(tunnel)
			}
		}
	}
}

func closeReverseTunnel(tunnel *rtunnels.RTunnel) {
	if tunnel == nil {
		return
	}
	rtunnels.RemoveRTunnelIf(tunnel.ID, tunnel)
	tunnel.Close()
}
