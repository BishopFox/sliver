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
	"errors"
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
	select {
	case <-implantConn.Done():
		return nil
	default:
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
	registrationReady := make(chan struct{})
	if !implantConn.SetCleanup(func() {
		<-registrationReady
		core.Sessions.Remove(session.ID)
	}) {
		sessionHandlerLog.Warnf("Rejected duplicate or closed session registration on connection %s", implantConn.ID)
		return nil
	}
	defer close(registrationReady)
	core.Sessions.Add(session)
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
	if implantConn == nil {
		return nil
	}
	if len(data) > maxTunnelDataMessageBytes {
		sessionHandlerLog.Warnf("Closing implant connection after oversized tunnel data message (%d bytes)", len(data))
		implantConn.Close()
		return nil
	}
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
		if actual, ok := reverseTunnelOpenings.Load(tunnelData.TunnelID); ok {
			existing := actual.(*reverseTunnelOpening)
			if existing.sessionID != session.ID {
				sessionHandlerLog.Warnf("Session %s attempted to reuse reverse tunnel ID %d owned by another session", session.ID, tunnelData.TunnelID)
				rejectReverseTunnel(implantConn, tunnelData.TunnelID, rtunnels.ErrDuplicateTunnelID)
			}
			// A retransmitted create is idempotent and must never add another
			// goroutine waiting behind the same outbound dial.
			return nil
		}
		if existing := rtunnels.GetRTunnel(tunnelData.TunnelID); existing != nil {
			if existing.SessionID != session.ID {
				sessionHandlerLog.Warnf("Session %s attempted to reuse active reverse tunnel ID %d owned by another session", session.ID, tunnelData.TunnelID)
				rejectReverseTunnel(implantConn, tunnelData.TunnelID, rtunnels.ErrDuplicateTunnelID)
			}
			// The first create frame can be retransmitted. An already-published
			// relay owned by this session makes that retransmission idempotent.
			return nil
		}
		openingAdmission := reverseTunnelOpeningAttempts
		if !openingAdmission.acquire(session.ID) {
			rejectReverseTunnel(implantConn, tunnelData.TunnelID, errReverseTunnelOpeningLimit)
			return nil
		}
		defer openingAdmission.release(session.ID)

		openingContext, cancelOpening := context.WithCancel(context.Background())
		opening := newReverseTunnelOpening(session.ID, cancelOpening)
		actual, loaded := reverseTunnelOpenings.LoadOrStore(tunnelData.TunnelID, opening)
		if loaded {
			cancelOpening()
			existing := actual.(*reverseTunnelOpening)
			if existing.sessionID != session.ID {
				sessionHandlerLog.Warnf("Session %s attempted to reuse reverse tunnel ID %d owned by another session", session.ID, tunnelData.TunnelID)
				rejectReverseTunnel(implantConn, tunnelData.TunnelID, rtunnels.ErrDuplicateTunnelID)
			}
			return nil
		}
		defer func() {
			cancelOpening()
			close(opening.ready)
			reverseTunnelOpenings.CompareAndDelete(tunnelData.TunnelID, opening)
		}()
		switch implantConn.TryClaimReverseTunnelID(tunnelData.TunnelID) {
		case core.ReverseTunnelIDClaimed:
		case core.ReverseTunnelIDDuplicate:
			rejectReverseTunnel(implantConn, tunnelData.TunnelID, rtunnels.ErrDuplicateTunnelID)
			return nil
		case core.ReverseTunnelIDCapacityExhausted, core.ReverseTunnelIDConnectionClosed:
			// Capacity exhaustion already fails the C2 connection closed. Do not
			// queue a rejection behind a connection that can no longer deliver it.
			return nil
		}
		response := createReverseTunnelHandlerWithContext(openingContext, implantConn, tunnelData, rtunnels.DefaultBroker)
		if opening.closing.Load() {
			if tunnel := rtunnels.GetRTunnel(tunnelData.TunnelID); tunnel != nil && tunnel.SessionID == session.ID {
				_ = closeReverseTunnelRemote(tunnel)
			}
		}
		return response
	}

	if value, ok := reverseTunnelOpenings.Load(tunnelData.TunnelID); ok {
		opening := value.(*reverseTunnelOpening)
		if opening.sessionID != session.ID {
			sessionHandlerLog.Warnf("Session %s attempted to send data on opening reverse tunnel %d owned by another session", session.ID, tunnelData.TunnelID)
			return nil
		}
		if err := opening.wait(reverseTunnelOpeningWaitTimeout); err != nil {
			opening.requestClose()
			rejectReverseTunnel(implantConn, tunnelData.TunnelID, err)
			return nil
		}
	}

	rtunnel := rtunnels.GetRTunnel(tunnelData.TunnelID)
	if rtunnel != nil && session.ID == rtunnel.SessionID {
		RTunnelDataHandler(tunnelData, rtunnel, implantConn)
		return nil
	} else if rtunnel != nil && session.ID != rtunnel.SessionID {
		sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on reverse tunnel it did not own", session.ID)
		return nil
	}

	tunnel := core.Tunnels.Get(tunnelData.TunnelID)
	if tunnel != nil {
		if session.ID == tunnel.SessionID {
			if err := tunnel.ProcessDataFromImplant(tunnelData); err != nil {
				if errors.Is(err, core.ErrTunnelClosed) {
					return nil
				}
				sessionHandlerLog.Warnf("Closing session %s after invalid generic tunnel frame on %d: %v", session.ID, tunnel.ID, err)
				core.Tunnels.CloseIf(tunnel)
				implantConn.Close()
			}
		} else {
			sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on tunnel it did not own", session.ID)
		}
	} else {
		sessionHandlerLog.Warnf("Data sent on nil tunnel %d", tunnelData.TunnelID)
	}

	return nil
}

func tunnelCloseHandler(implantConn *core.ImplantConnection, data []byte) *sliverpb.Envelope {
	if implantConn == nil {
		return nil
	}
	if len(data) > maxTunnelDataMessageBytes {
		sessionHandlerLog.Warnf("Closing implant connection after oversized tunnel close message (%d bytes)", len(data))
		implantConn.Close()
		return nil
	}
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
		if !opening.claimTerminal() {
			return nil
		}
		// Capability-bearing implants sequence terminal EOF after all earlier
		// data, so their terminal must not cancel a still-pending authorized
		// dial. Legacy closes have no ordering contract and retain abort behavior.
		if session.Capabilities&sliverpb.CapabilityTunnelTerminalV1 == 0 {
			opening.requestClose()
		}
		if err := opening.waitReady(reverseTunnelOpeningWaitTimeout); err != nil {
			sessionHandlerLog.Warnf("Could not wait for closing reverse tunnel %d: %v", tunnelData.TunnelID, err)
			return nil
		}
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
			go core.Tunnels.ScheduleCloseTunnel(tunnel)
		} else {
			sessionHandlerLog.Warnf("Warning: Session %s attempted to send data on tunnel it did not own", session.ID)
		}
		tunnelHandlerMutex.Unlock()
		return nil
	}
	tunnelHandlerMutex.Unlock()

	rtunnel := rtunnels.GetRTunnel(tunnelData.TunnelID)
	if rtunnel != nil && session.ID == rtunnel.SessionID {
		ready, err := rtunnel.MarkPeerClose(tunnelData.Sequence)
		if err != nil {
			if !errors.Is(err, rtunnels.ErrReverseTunnelClosed) {
				sessionHandlerLog.Warnf("Closing session %s after invalid terminal sequence on reverse tunnel %d: %v", session.ID, rtunnel.ID, err)
				if closeReverseTunnelRemote(rtunnel) {
					implantConn.Close()
				}
			}
			return nil
		}
		if ready {
			_ = closeReverseTunnelRemote(rtunnel)
			return nil
		}
		rtunnel.StartPeerCloseDeadline(reverseTunnelSendTimeout, func() {
			if closeReverseTunnelRemote(rtunnel) {
				sessionHandlerLog.Warnf("Closing session %s after incomplete terminal sequence on reverse tunnel %d", session.ID, rtunnel.ID)
				implantConn.Close()
			}
		})
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
	return createReverseTunnelHandlerWithContext(context.Background(), implantConn, req, broker)
}

//nolint:gocyclo // Opening is a single transaction spanning validation, dialing, publication, and relay cleanup.
func createReverseTunnelHandlerWithContext(openingContext context.Context, implantConn *core.ImplantConnection, req *sliverpb.TunnelData, broker *rtunnels.Broker) *sliverpb.Envelope {
	if implantConn == nil {
		sessionHandlerLog.Warnf("Rejected malformed reverse tunnel creation request")
		return nil
	}
	session := core.Sessions.FromImplantConnection(implantConn)
	if session == nil || req == nil || req.Rportfwd == nil || broker == nil {
		sessionHandlerLog.Warnf("Rejected malformed reverse tunnel creation request")
		return nil
	}

	legacyAddress := ""
	if req.Rportfwd.AuthorizationID == "" {
		legacyAddress = net.JoinHostPort(req.Rportfwd.Host, strconv.FormatUint(uint64(req.Rportfwd.Port), 10)) //nolint:staticcheck // Required for exact legacy wire compatibility.
	}
	if openingContext == nil {
		openingContext = context.Background()
	}
	dst, resolvedAuthorizationID, err := broker.Open(
		openingContext,
		session.ID,
		rtunnels.AuthorizationID(req.Rportfwd.AuthorizationID),
		legacyAddress,
	)
	if err != nil {
		rejectReverseTunnel(implantConn, req.TunnelID, err)
		return nil
	}
	if err := openingContext.Err(); err != nil {
		_ = dst.Close()
		rejectReverseTunnel(implantConn, req.TunnelID, err)
		return nil
	}

	tunnel := rtunnels.NewAuthorizedRTunnel(req.TunnelID, session.ID, resolvedAuthorizationID, dst, dst)
	// Legacy implants dispatch tunnel envelopes from independent transport
	// streams and ignore terminal sequences. A separate close envelope can
	// therefore overtake the final data envelope. Keep their fallback entirely
	// server-side: close and detach the relay, but leave already-queued data as
	// the last wire message. Capability-bearing implants use sequenced close.
	if session.Capabilities&sliverpb.CapabilityTunnelTerminalV1 != 0 {
		tunnel.SetPeerCloseNotifier(func(sequence uint64) error {
			data, err := proto.Marshal(&sliverpb.TunnelData{
				Closed:   true,
				TunnelID: tunnel.ID,
				Sequence: sequence,
			})
			if err != nil {
				return err
			}
			err = queueTunnelEnvelope(implantConn, tunnel, &sliverpb.Envelope{Type: sliverpb.MsgTunnelClose, Data: data})
			if err != nil && !tunnel.PeerTeardownPending() {
				go implantConn.Close()
			}
			return err
		})
	}
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
			_, _ = closeReverseTunnelLocal(tunnel)
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
	rejectionSlots := reverseTunnelRejectionSlots
	select {
	case rejectionSlots <- struct{}{}:
	default:
		sessionHandlerLog.Warnf("Closing implant connection because rejection delivery capacity is exhausted for reverse tunnel %d", tunnelID)
		implantConn.Close()
		return
	}

	// HTTP implants cannot poll until their request handler returns. Deliver the
	// rejection asynchronously, but cap both goroutine count and wait time. If
	// the only close notification cannot be delivered, fail the connection
	// closed so no server-side authorization or relay can be orphaned.
	go func() {
		defer func() { <-rejectionSlots }()
		if err := implantConn.SendEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgTunnelClose, Data: data}, reverseTunnelRejectionTimeout); err != nil {
			if !errors.Is(err, core.ErrImplantConnectionClosed) {
				sessionHandlerLog.Warnf("Closing implant connection after rejection delivery timed out for reverse tunnel %d", tunnelID)
				implantConn.Close()
			}
		}
	}()
}

func RTunnelDataHandler(tunnelData *sliverpb.TunnelData, tunnel *rtunnels.RTunnel, connection *core.ImplantConnection) {
	if tunnelData == nil || tunnel == nil || connection == nil || tunnel.Writer == nil {
		return
	}
	if tunnelData.Resend {
		sessionHandlerLog.Warnf("Closing reverse tunnel %d after unsupported resend control frame", tunnel.ID)
		_, _ = closeReverseTunnelLocal(tunnel)
		return
	}
	pending, err := tunnel.ProcessInbound(tunnelData.Sequence, tunnelData.Data, func(payload []byte) error {
		if deadlineWriter, ok := tunnel.Writer.(interface{ SetWriteDeadline(time.Time) error }); ok {
			_ = deadlineWriter.SetWriteDeadline(time.Now().Add(rtunnels.DefaultDialTimeout))
			defer func() {
				_ = deadlineWriter.SetWriteDeadline(time.Time{})
			}()
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
		if errors.Is(err, rtunnels.ErrReverseTunnelClosed) {
			return
		}
		closed, closeErr := closeReverseTunnelLocal(tunnel)
		if !closed {
			return
		}
		sessionHandlerLog.Warnf("Closing reverse tunnel %d after bounded inbound relay failure: %v", tunnel.ID, err)
		if closeErr != nil {
			sessionHandlerLog.Warnf("Failed to notify implant while closing reverse tunnel %d: %v", tunnel.ID, closeErr)
		}
		if errors.Is(err, rtunnels.ErrReverseTunnelTerminal) {
			connection.Close()
		}
		return
	}

	_ = pending // reliable reverse transports retain bounded delayed frames
	if tunnel.PeerCloseReady() {
		_ = closeReverseTunnelRemote(tunnel)
	}
}

func closeReverseTunnelLocal(tunnel *rtunnels.RTunnel) (bool, error) {
	return rtunnels.CloseLocalIfActive(tunnel)
}

func closeReverseTunnelRemote(tunnel *rtunnels.RTunnel) bool {
	return rtunnels.CloseRemoteIfActive(tunnel)
}
