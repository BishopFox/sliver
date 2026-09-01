package core

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
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports/mtls"
	"github.com/bishopfox/sliver/implant/sliver/transports/wireguard"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"

	consts "github.com/bishopfox/sliver/client/constants"
)

var (
	sessionsLog = log.NamedLogger("core", "sessions")

	// Sessions - Manages implant connections
	Sessions = &sessions{
		sessions: &sync.Map{},
	}

	// ErrUnknownMessageType - Returned if the implant did not understand the message for
	//                         example when the command is not supported on the platform
	ErrUnknownMessageType = errors.New("unknown message type")

	// ErrImplantTimeout - The implant did not respond prior to timeout deadline
	ErrImplantTimeout = errors.New("implant timeout")
)

// Session - Represents a connection to an implant
type Session struct {
	ID                string
	Name              string
	Hostname          string
	Username          string
	UUID              string
	UID               string
	GID               string
	OS                string
	Version           string
	Arch              string
	PID               int32
	Filename          string
	Connection        *ImplantConnection
	ActiveC2          string
	ReconnectInterval int64
	ProxyURL          string
	PollTimeout       int64
	Burned            bool
	Extensions        []string
	ConfigID          string
	PeerID            int64
	Locale            string
	FirstContact      int64
	Integrity         string
	Capabilities      uint64
}

// LastCheckin - Get the last time a session message was received
func (s *Session) LastCheckin() time.Time {
	return s.Connection.GetLastMessage()
}

// IsDead - See if last check-in is within expected variance
func (s *Session) IsDead() bool {
	sessionsLog.Debugf("Checking health of %s", s.ID)
	sessionsLog.Debugf("Last checkin was %v", s.LastCheckin())
	padding := time.Duration(10 * time.Second) // Arbitrary margin of error
	timePassed := time.Since(s.LastCheckin())
	reconnect := time.Duration(s.ReconnectInterval)
	pollTimeout := time.Duration(s.PollTimeout)
	if timePassed < reconnect+padding && timePassed < pollTimeout+padding {
		sessionsLog.Debugf("Last message within reconnect interval / poll timeout + padding")
		return false
	}
	if s.Connection.Transport == consts.MtlsStr {
		if timePassed < mtls.PingInterval+padding {
			sessionsLog.Debugf("Last message within ping interval with padding")
			return false
		}
	}
	if s.Connection.Transport == consts.WGStr {
		if timePassed < wireguard.PingInterval+padding {
			sessionsLog.Debugf("Last message with ping interval with padding")
			return false
		}
	}
	if s.Connection.Transport == "pivot" {
		if time.Since(s.Connection.GetLastMessage()) < time.Duration(time.Minute)+padding {
			sessionsLog.Debugf("Last message within pivot/server ping interval with padding")
			return false
		}
	}
	return true
}

// ToProtobuf - Get the protobuf version of the object
func (s *Session) ToProtobuf() *clientpb.Session {
	var transport, remoteAddress string
	var lastCheckin int64
	var isDead bool
	if s.Connection != nil {
		transport = s.Connection.Transport
		remoteAddress = s.Connection.RemoteAddress
		lastCheckin = s.LastCheckin().Unix()
		isDead = s.IsDead()
	}
	return &clientpb.Session{
		ID:                s.ID,
		Name:              s.Name,
		Hostname:          s.Hostname,
		Username:          s.Username,
		UUID:              s.UUID,
		UID:               s.UID,
		GID:               s.GID,
		OS:                s.OS,
		Version:           s.Version,
		Arch:              s.Arch,
		Transport:         transport,
		RemoteAddress:     remoteAddress,
		PID:               int32(s.PID),
		Filename:          s.Filename,
		LastCheckin:       lastCheckin,
		ActiveC2:          s.ActiveC2,
		IsDead:            isDead,
		ReconnectInterval: s.ReconnectInterval,
		ProxyURL:          s.ProxyURL,
		Burned:            s.Burned,
		PeerID:            s.PeerID,
		Locale:            s.Locale,
		FirstContact:      s.FirstContact,
		Integrity:         s.Integrity,
		Capabilities:      s.Capabilities,
	}
}

// Request - Sends a protobuf request to the active sliver and returns the response
func (s *Session) Request(msgType uint32, timeout time.Duration, data []byte) ([]byte, error) {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	response, err := s.RequestContext(ctx, msgType, data)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrImplantSendTimeout) {
		return nil, ErrImplantTimeout
	}
	return response, err
}

// RequestContext sends a request and uses one context budget across both
// outbound queueing and the response wait. A canceled context is checked before
// a response waiter is installed, and every return path removes that waiter.
func (s *Session) RequestContext(ctx context.Context, msgType uint32, data []byte) ([]byte, error) {
	if s == nil || s.Connection == nil {
		return nil, ErrImplantConnectionClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	connection := s.Connection
	resp := make(chan *sliverpb.Envelope, 1)
	reqID := EnvelopeID()
	connection.RespMutex.Lock()
	connection.Resp[reqID] = resp
	connection.RespMutex.Unlock()
	defer func() {
		connection.RespMutex.Lock()
		defer connection.RespMutex.Unlock()
		// close(resp)
		delete(connection.Resp, reqID)
	}()

	sendTimeout := 60 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		sendTimeout = time.Until(deadline)
		if sendTimeout <= 0 {
			return nil, context.DeadlineExceeded
		}
	}
	err := connection.SendEnvelopeUntil(&sliverpb.Envelope{
		ID:   reqID,
		Type: msgType,
		Data: data,
	}, ctx.Done(), sendTimeout)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	select {
	case respEnvelope := <-resp:
		if respEnvelope.UnknownMessageType {
			return nil, ErrUnknownMessageType
		}
		return respEnvelope.Data, nil
	case <-connection.Done():
		return nil, ErrImplantConnectionClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sessions - Manages the slivers, provides atomic access
type sessions struct {
	sessions *sync.Map // map[uint32]*Session
}

// All - Return a list of all sessions
func (s *sessions) All() []*Session {
	all := []*Session{}
	s.sessions.Range(func(key, value interface{}) bool {
		all = append(all, value.(*Session))
		return true
	})
	return all
}

// Get - Get a session by ID
func (s *sessions) Get(sessionID string) *Session {
	if val, ok := s.sessions.Load(sessionID); ok {
		return val.(*Session)
	}
	return nil
}

// Add - Add a sliver to the hive (atomically)
func (s *sessions) Add(session *Session) *Session {
	s.sessions.Store(session.ID, session)
	EventBroker.Publish(Event{
		EventType: consts.SessionOpenedEvent,
		Session:   session,
	})
	return session
}

// Remove - Remove a sliver from the hive (atomically)
func (s *sessions) Remove(sessionID string) {
	val, ok := s.sessions.Load(sessionID)
	if !ok {
		return
	}
	parentSession := val.(*Session)
	children := findAllChildrenByPeerID(parentSession.PeerID)
	if !s.sessions.CompareAndDelete(sessionID, val) {
		return
	}
	cleanupReversePortForwards(parentSession.ID)
	coreLog.Debugf("Removing %d children of session %s (%v)", len(children), parentSession.ID, children)
	for _, child := range children {
		childSession, ok := s.sessions.LoadAndDelete(child.SessionID)
		if ok {
			removedChild := childSession.(*Session)
			cleanupReversePortForwards(removedChild.ID)
			if removedChild.Connection != nil {
				closePivotForConnection(removedChild.Connection)
				removedChild.Connection.Close()
			}
			EventBroker.Publish(Event{
				EventType: consts.SessionClosedEvent,
				Session:   removedChild,
			})
		}
	}
	if parentSession.Connection != nil && parentSession.Connection.Transport == PivotTransportName {
		closePivotForConnection(parentSession.Connection)
		parentSession.Connection.Close()
	}

	// Remove the parent session
	EventBroker.Publish(Event{
		EventType: consts.SessionClosedEvent,
		Session:   parentSession,
	})
}

func cleanupReversePortForwards(sessionID string) {
	// Revoke first to cancel pending broker dials, then detach any registered
	// relays. Both operations are idempotent because disconnect paths can race.
	rtunnels.DefaultRegistry.RevokeSession(sessionID)
	rtunnels.CloseSession(sessionID)
}

// NewSession - Create a new session
func NewSession(implantConn *ImplantConnection) *Session {
	implantConn.UpdateLastMessage()
	return &Session{
		ID:           nextSessionID(),
		Connection:   implantConn,
		FirstContact: time.Now().Unix(),
		Integrity:    "-",
	}
}

// FromImplantConnection - Find the session associated with an implant connection
func (s *sessions) FromImplantConnection(conn *ImplantConnection) *Session {
	var found *Session
	s.sessions.Range(func(key, value interface{}) bool {
		if value.(*Session).Connection.ID == conn.ID {
			found = value.(*Session)
			return false
		}
		return true
	})
	return found
}

// nextSessionID - Returns an incremental nonce as an id
func nextSessionID() string {
	id, _ := uuid.NewV4()
	return id.String()
}

// UpdateSession - In place update of a session pointer
func (s *sessions) UpdateSession(session *Session) *Session {
	s.sessions.Store(session.ID, session)
	EventBroker.Publish(Event{
		EventType: consts.SessionUpdateEvent,
		Session:   session,
	})
	return session
}
